# DDEV on Apple Container + socktainer — investigation notes (2026-08-02)

Working notes for [#7372](https://github.com/ddev/ddev/issues/7372). This documents an
exploratory session, not a finished feature: the code changes here are experiments that only
take effect when the Docker provider is socktainer, and this file is expected to be deleted
before any of it merges.

Environment: Apple `container` 1.2.0, `socktainer` 1.2.1 (Homebrew bottle), docker CLI 29.4.0,
docker context `socktainer`, macOS 27, `ddev` built from
branch `20260802_rfay_apple_container_experiment`. Mutagen was off throughout
(`performance_mode: none`).

## How far it got

**It works end to end**, with `container` installed from the **signed installer** (not
Homebrew — see blocker 4, which turned out to be Local Network authorization scoped to a
binary path):

```text
http://appletest.ddev.site:8080/    → 200
https://appletest.ddev.site:8443/   → 200   (phpinfo, PHP 8.4.24)
http://127.0.0.1:57118/             → 200   (web's own host binding)
https://appletest.ddev.site:33001/  → 200   (Mailpit)
ddev ssh / ddev exec / mysql        → all work
```

`ddev start` exits 0, all three containers healthy in ~10s each, images build in ~1m15s,
reached at DDEV's advertised hostnames from the host — no container IP, no hand-set Host
header.

Getting there took three socktainer fixes beyond the seven already queued (one of them a
**correction to what was open upstream as
[socktainer#347](https://github.com/socktainer/socktainer/pull/347)**), plus switching
`container` from Homebrew to the signed installer. See the verification-run section, which
supersedes what blockers 3, 4 and 9 said before.

Still required by hand: unprivileged router ports,
`omit_containers: [ddev-ssh-agent]`, a non-colliding `traefik_monitor_port`, removing
project containers before `ddev start`, and the cold-start recipe. Mutagen no longer needs
a per-project setting — it is forced off automatically when the provider is socktainer (see
below). `ddev restart` and `ddev debug rebuild` still exit 1 on the router hostname
collision (blocker 12). So: a working proof of concept, not a supported provider.

## Verification run against a merged local build (2026-08-03, later)

All seven queued fixes were merged onto a scratch branch (`tmp/rollout-verify` in the
fork) and built with `make release`, following the recipe in
[the PR comment](https://github.com/ddev/ddev/pull/8656#issuecomment-5167598650).
Three things turned up that the individually-tested branches had not.

### #347 as written breaks all image builds — must be corrected before merge

`fix/exec-hijack-close` bounds the hijacked exec-start channel at 10 s and closes it
whether or not the process has exited. Any exec that legitimately runs longer is
truncated, and buildx's build driver *is* such an exec (`buildctl dial-stdio` lives for
the whole build), so every `ddev start` failed in image build with:

```text
target db: failed to receive status: rpc error: code = Unavailable desc = error reading from server: EOF
```

Isolated directly, unpatched vs. patched:

| socktainer | `docker exec -i keepalive sh -c 'sleep 30; echo done-30'` |
|---|---|
| 1.2.1 Homebrew bottle | `done-30`, 30.2 s, exit 0 |
| local build w/ #347 | no output, **10.1 s**, exit 255 |

The bound is the wrong mechanism: `process.wait()` stalling in XPC cannot be told
apart from a process that is simply still running. Output-pipe EOF *can* — Apple
Container closes the container-side write ends when the exec process exits, so EOF is
the exit signal and it arrives whether or not XPC acknowledges anything. The corrected
shape (now in `tmp/rollout-verify`) closes the channel when every attached output pipe
has hit EOF, and starts the 10 s bound *there*, only to collect the numeric exit code.
The chunked-stream path in the same file already worked this way — #347 should have
copied it rather than introducing a bound. After the correction: 30 s exec returns at
30.1 s with exit 0, a 15 s `exit 7` reports 7, and builds succeed (1m16s).

### The real cause of blocker 9 was upgrade gating, not the hijack close

Even with the corrected close, `ddev start` still hung at
`Getting traefik error output`. `dockerutil.Exec` hangs against socktainer on **every**
container, not just ddev-router — reproduced with a scratch Go test calling it against
`keepalive`, `ddev-appletest-web` and `ddev-router` (all three hung 20 s; the docker
CLI running the identical command returned in 0.06 s).

Cause: socktainer only honors an exec upgrade when stdin is attached —

```swift
let shouldUpgrade = connection == "upgrade" && upgrade == "tcp" && config.attachStdin
```

`dockerutil.Exec` sets `AttachStdout`/`AttachStderr` only, so `shouldUpgrade` is false
and socktainer answers with a chunked response. But the Go client asked to upgrade and
therefore takes over the socket and reads until the *connection* closes — which never
happens on a keep-alive connection whose body has merely ended. moby hijacks whenever
asked, regardless of stdin. Dropping `&& config.attachStdin` fixes it: `dockerutil.Exec`
returns in 0.12 s against all three containers, and `docker exec` with no flags, `-i`,
`-t` and a non-zero exit code all still behave.

This is what [socktainer#346](https://github.com/socktainer/socktainer/issues/346)
should be about. The hijack-close bug is real too, but it is not what DDEV hit.

### Blocker 3 root-caused and fixed: the response echoed the query's OPT record

The 502 was not the multi-homed-address bug. `buildAResponse`/`buildNodataResponse`
built each reply by copying the whole query — including any EDNS0 OPT record in its
additional section — then set ARCOUNT to 0 and appended the answer RR *after* those
leftover OPT bytes. A strict parser reads the answer section where ARCOUNT/ANCOUNT say
it should be, lands on the OPT bytes, and reports no usable data, which is exactly the
`curl: (6) ... DNS server returned answer with no data` in blocker 3, and matches
[socktainer#329](https://github.com/socktainer/socktainer/issues/329)'s
"malformed EDNS0". `getent` worked throughout because it does not send EDNS0.

Fix: truncate the reply at the end of the question before appending answers, and claim
no OPT (omitting OPT is legal — the responder is just treated as not supporting EDNS0).
With it, `curl http://ddev-appletest-web/` from inside ddev-router returns 200 and
traefik stops 502ing.

### `fix/filter-name-ignored` only fixed half the bug

The branch corrected the *matching* semantics but not the *parsing*:
`parseContainerFilters` accepted the Docker CLI's `{"key":{"value":true}}` encoding only
for `label`, so `--filter name=`, `status=`, `id=` were dropped before matching ran.
`docker ps -a --filter name=doesnotexist` still listed every container after the merge.
Fixed by accepting the dict form for all keys. Note DDEV itself was never exposed to
this: `FindContainerByName` re-checks `Names[0]` against the exact name.

### Version reporting now trips DDEV's minimum-version check

With `fix/version-engine-version` in place, `ddev version` reports `docker 0.0.0-dev`
and `ddev start` prints:

```text
Problem with your Docker provider: installed Docker version 0.0.0-dev is not supported,
please update to version 25.0 or newer.
```

A tagged socktainer release would report `1.2.1` and fail the same `>= 25.0` check, so
the fix trades a wrong version string for a wrong warning. DDEV needs a
socktainer-aware branch in that check (as it presumably has for other providers) —
a DDEV-side follow-up, not a socktainer one.

### Not retested

`fix/healthcheck-timing` was in the build, but the project kept its shortened
healthcheck overrides throughout, so whether DDEV's default
`interval 1s / timeout 70s / start_period 120s` now works was **not** tested.
The socktainer unit suite could not be run at all in this environment: the tests
compile, but `swift test` cannot load `Testing.framework` (only the `_Testing_*`
overlays ship with Command Line Tools; running them needs full Xcode).

## Upstream engagement status (2026-08-03)

Fixes for seven of the eleven blockers below are written, tested, and DCO-signed-off
against socktainer, tracked in the
[`rfay/socktainer` fork's rollout plan](https://github.com/rfay/socktainer/blob/fix/exec-hijack-close/UPSTREAM_ROLLOUT.md),
ranked by DDEV relevance vs. diff risk. Submitting upstream one at a time rather than
bundling. Status so far:

- **Filed upstream:** [socktainer/socktainer#346](https://github.com/socktainer/socktainer/issues/346)
  (exec-hijack hang, blocker 9) / [socktainer/socktainer#347](https://github.com/socktainer/socktainer/pull/347)
  (fix, open, DCO passing, awaiting review) — **needs correcting before it merges: as
  written it truncates every exec longer than 10 s and breaks all image builds.** See the
  verification-run section above.
- **Already filed by someone else, corroborated independently and now root-caused:**
  [socktainer/socktainer#329](https://github.com/socktainer/socktainer/issues/329)
  (malformed EDNS0, blocker 3) — fix written, worth offering on that issue.
- **Fix ready, queued to submit next in rollout order:** DNS wrong-network-address
  (blocker 3's second bug), version reporting (blocker 11), archive-on-unstarted-container
  (blocker 6), healthcheck timing (blocker 5), `--filter name=` (blocker 11), `docker cp`
  directory hang (blocker 7)
- **Investigated, could not reproduce:** underscore container names skipped for DNS
  (blocker 11) — worth closing as not-reproducible upstream, or asking the original
  reporter for a sharper repro

Independently, four DDEV-side hardenings came out of this investigation and are open as
their own draft PRs, unrelated to any socktainer fix landing:
[#8657](https://github.com/ddev/ddev/pull/8657) (bound host ports fallback),
[#8658](https://github.com/ddev/ddev/pull/8658) (`GetExistingDBType` via exec — design
option 5 below), [#8659](https://github.com/ddev/ddev/pull/8659) (non-fatal global-cache
chown — design option 6 below), [#8660](https://github.com/ddev/ddev/pull/8660) (stop
hijacking the exec stream in `dockerutil.Exec()` — the DDEV-side counterpart to #347, so
`GetRouterConfigErrors()` stops hanging even before the socktainer fix merges).

## Cold-start recipe

The environment does not survive idling — the `default` network's host bridge is torn down
once nothing is running on it, which kills DNS (blocker 10), and the buildkit node exits and
then cannot be restarted (blocker 6). Run these in order before trying `ddev start`, and
expect to repeat them after any `container system` restart:

```bash
# 1. keep the default network (and therefore 192.168.64.1) alive
docker run -d --name keepalive ddev/ddev-utilities:latest sleep 100000

# 2. resolver on the gateway, forwarding to socktainer's DNS. Must come AFTER step 1;
#    it fails with "Can't assign requested address" if the bridge does not exist yet.
sudo pkill -f "listen-address=192.168.64.1"
sudo /opt/homebrew/sbin/dnsmasq --keep-in-foreground \
  --listen-address=192.168.64.1 --bind-interfaces --port=53 \
  --no-resolv --server=127.0.0.1#2054 &

# 3. buildkit node — RECREATE, never restart (see blocker 6)
docker rm -f buildx_buildkit_default
docker run -d --name buildx_buildkit_default --cap-add ALL \
  moby/buildkit:buildx-stable-1 --allow-insecure-entitlement=network.host

# 4. sanity check
docker run --rm ddev/ddev-utilities:latest nslookup registry-1.docker.io
```

The project also needs `router_http_port`/`router_https_port` above 1024 (blocker 4b) and the
healthcheck overrides (blocker 5). The global-cache bind mount needs no environment
variable — it detects socktainer on its own. A `ddev start` against an *already running*
project always fails on the db volume (blocker 1), so remove `ddev-appletest-web` and
`ddev-appletest-db` first.

Two more things the 2026-08-03 run needed:

- **`traefik_monitor_port` must not collide.** With OrbStack's router running it owns
  10999, and ddev-router dies with
  `bind(...): Address already in use (errno: 48)`. Set 11999 in
  `~/.ddev/global_config.yaml`, or stop the other router.
- **`ddev restart` after any socktainer restart.** Health status resets to `starting` for
  every container when the daemon comes back (the healthcheck manager re-adopts them and
  starts over) and stays there, so anything waiting for healthy hangs. The site keeps
  serving throughout; it is the reported status that is stale. `ddev restart` clears it —
  all three back to `healthy` in ~35s.
- A **surviving buildkit node does not need recreating** after a socktainer restart:
  `ddev debug rebuild` built both images in 9s against the node that predated the restart.
  Recreation is only needed once the node itself has exited (blocker 6).

Once started, reach the site at the router's container IP, not the published port:

```bash
RIP=$(docker inspect ddev-router --format '{{range .NetworkSettings.Networks}}{{.IPAddress}} {{end}}' | awk '{print $1}')
curl -sS -o /dev/null -w '%{http_code}\n' -H 'Host: appletest.ddev.site' http://$RIP:8080/
curl -sS -k -o /dev/null -w '%{http_code}\n' --resolve appletest.ddev.site:8443:$RIP https://appletest.ddev.site:8443/
```

---

## Blockers that need upstream fixes (socktainer / Apple Container)

### 1. Named volumes cannot be shared read-write — *the* structural blocker

Apple Container backs each named volume with an ext4 block image attached to one VM:

```bash
docker volume create voltest1
docker run -d --name volA -v voltest1:/data alpine sleep 300
docker run --rm -v voltest1:/data alpine ls /data
# Error Domain=VZErrorDomain Code=2 "The storage device attachment is invalid."
```

**Read-only attach is not exclusive, and this is the important nuance.** The rule is that a
volume is either attached read-write by exactly one container, *or* read-only by any number
of them:

| Held by | New `:ro` attach | New rw attach |
|---|---|---|
| nothing | works | works |
| one rw container | **fails** | **fails** |
| one or more `:ro` containers | **works** (tested with 3) | **fails** |

That leaves a usable pattern: seed a volume from a short-lived read-write container while
nothing else holds it, then have every consumer mount it `:ro`. See design option 7 below.

Host bind mounts (virtiofs) are shareable read-write. `docker volume create --driver local
-o type=none -o device=... -o o=bind` is ignored by socktainer — it still makes a block
volume.

Where this hits DDEV:

- `ddev-global-cache` — mounted simultaneously by web, db, router, ssh-agent and every
  `RunSimpleContainer` helper.
- `ddev-ssh-agent_socket_dir` — shared by ssh-agent and web.
- With mutagen on, web mounts `project_mutagen` **twice** (`/var/www` and
  `/tmp/project_mutagen`) — fails even within a single container.
- `GetExistingDBType` and snapshot helpers mount the db volume while `ddev-<p>-db` runs, so
  **`ddev start` on an already-running project always fails**; containers must be removed first.

### 2. No DNS on the built-in `default` network

Containers get `nameserver 192.168.64.1` (the vmnet gateway) but nothing listens there —
every lookup is `connection refused`. This breaks anything on `default`, including Apple's
own build VM (`container build` fails at `apk add` with "DNS: transient error").

Host-side workaround that fixes it (forwards to socktainer's own DNS, which also resolves
container names):

```bash
sudo /opt/homebrew/sbin/dnsmasq --keep-in-foreground \
  --listen-address=192.168.64.1 --bind-interfaces --port=53 \
  --no-resolv --server=127.0.0.1#2054      # 2054 = socktainer's DNS port
```

### 3. DNS returns unusable AAAA answers — currently the last blocker

On a socktainer network, `getent hosts ddev-appletest-web` and `nslookup -type=A` both
return the right address, but `nslookup -type=AAAA` returns an empty answer and
**curl (musl) and traefik (Go) both fail to resolve the name**:

```
curl: (6) Could not resolve host: ddev-appletest-web (DNS server returned answer with no data)
```

Result: traefik logs `502 ... "http://ddev-appletest-web:80"`, while `curl http://<web-ip>/`
from the same container returns 200.

**Status: root-caused and fixed.** The malformed EDNS0 response
([socktainer/socktainer#329](https://github.com/socktainer/socktainer/issues/329), filed
by someone else) is `buildAResponse`/`buildNodataResponse` echoing the query's OPT
record and appending the answer after it — see the verification-run section above for
the mechanism and the fix. This was the actual 502, and fixing it is what got the site
serving. A second, distinct DNS bug also exists: multi-homed hostnames get an arbitrary
network's address instead of the querying client's — fix ready
(`fix/dns-wrong-network-address`), not yet submitted, and *not* what caused the 502.

### 4. Published ports are accepted but reset — SOLVED: use the signed installer

`docker run -d -p 127.0.0.1:38080:80 nginx:alpine` → the `container` helper listens on
127.0.0.1:38080, but connections get `Recv failure: Connection reset by peer`. The same
nginx answers 200 on its container IP directly from the host.

**Cause: macOS Local Network (TCC) authorization is scoped to a binary path, and only the
signed installer's path was authorized.** Verified by installing
`container-1.2.0-installer-signed.pkg` — same version, same OS, same config — after which
the identical nginx repro returns 200, and DDEV serves at its real URLs.

| Install | nginx published-port test |
|---|---|
| Homebrew 1.2.0 (`/opt/homebrew/…`) | `Recv failure: Connection reset by peer` |
| Signed installer 1.2.0 (`/usr/local/…`) | 200 (3/3; second port also 200) |

`/Library/Preferences/com.apple.networkextension.plist` (readable with `sudo`, no Full Disk
Access) stores one record per authorized binary as a `(Path, SigningIdentifier)` pair, and
it is path-keyed rather than app-keyed — Chrome has a dozen records, one per
`code_sign_clone` path. Only `/usr/local/libexec/container/plugins/…/container-runtime-linux`
was recorded; the Homebrew binary had no record. Unauthorized, the helper binds the
listener, accepts the connection, then fails its backend connect with
`No route to host (errno: 65)` — `EHOSTUNREACH`, the signature of
[apple/container#2067](https://github.com/apple/container/issues/2067), which closed with
exactly this root cause. No prompt ever surfaces for the Homebrew copy, plausibly because
the helper runs as a launchd service with no UI session.

**Action for DDEV docs:** install Apple Container from the signed installer at
[apple/container releases](https://github.com/apple/container/releases), not
`brew install container`. This is invisible otherwise and looks exactly like a broken
provider.

Also still true: Apple Container makes container IPs routable from the host, so a DDEV
"Apple Container mode" pointing hostnames at the router's container IP remains an option —
but it is no longer *necessary*, so it should be weighed on its merits rather than as a
workaround.

### 4b. Ports below 1024 cannot be published at all

Apple Container's port forwarder runs unprivileged, so DDEV's *default* configuration is
rejected outright:

```text
Failed to start container: ... invalidArgument: "Permission denied while binding to
host port 443. Binding to ports below 1024 requires root privileges."
```

```bash
docker run -d -p 127.0.0.1:443:80  ddev/ddev-utilities sleep 60   # error
docker run -d -p 127.0.0.1:8443:80 ddev/ddev-utilities sleep 60   # works
```

This only shows up when 80/443 are actually free — with another router (OrbStack's) holding
them, DDEV picks ephemeral high ports and sidesteps it by accident. Fix:
`ddev config --router-http-port=8080 --router-https-port=8443`.

**This is not a new class of problem for DDEV.** Rootless Podman is already a supported
provider with the same restriction, and the docs already say so: *"Podman rootless — Rootless
by default … Can't use the default ports 80/443, so DDEV must be configured to use
unprivileged ports"* (`docs/content/users/install/docker-installation.md`), and the Buildkite
test machine setup configures `router-http-port=8080` / `router-https-port=8443` for exactly
this reason. So the handling is a documented configuration requirement, not new machinery.

The one macOS-specific wrinkle is that Linux offers an escape hatch —
`net.ipv4.ip_unprivileged_port_start=0`, which the DDEV docs recommend for rootless — and
macOS has no equivalent. On Apple Container, unprivileged router ports are mandatory rather
than merely recommended. Using the router's container IP instead of published ports (see
blocker 4) would avoid the question entirely.

### 5. Healthchecks never report with DDEV's timings

DDEV uses `interval 1s / timeout 70s / start_period 120s`. With those, socktainer reports
`starting` forever and `State.Health.Log` stays empty (verified past 4 minutes).
`--health-start-period 15s` works fine (`healthy`, log populated). Shortening to
`interval 2s / timeout 10s / start_period 10s` via a compose override made all three
DDEV containers healthy in ~10s. Health also **regressed from `healthy` back to `starting`**
on long-running containers, so the status is not stable.

**Status:** fix ready (`fix/healthcheck-timing`, fixes both the regression and the
stalled-probe freeze), not yet submitted upstream.

### 6. buildx cannot bootstrap its buildkit container

Two gaps:

- buildx creates the container, then `PUT /containers/<id>/archive` into it. Socktainer
  answers `404 Rootfs not found` for a *created but not started* container.
- `--privileged` is ignored (documented), so buildkit's `rbind` fails with
  `operation not permitted`.

Workaround — pre-create the node by hand (buildx adopts a running container with the right
name), with `--cap-add ALL` and on the `default` network so its DNS address is stable:

```bash
docker run -d --name buildx_buildkit_default --cap-add ALL \
  moby/buildkit:buildx-stable-1 --allow-insecure-entitlement=network.host
```

**The node must be recreated, never restarted.** Once it has exited, buildx will happily
restart it (`#2 starting container buildx_buildkit_default`) and every build then fails with
the same `rbind ... operation not permitted` as an uncapped node. `docker rm -f` plus a fresh
`docker run` of the identical command fixes it immediately — the very next build succeeded in
52s. The obvious explanation does not hold: capabilities survive a restart
(`CapEff: 000001ffffffffff` before and after, measured on an exec'd process, so not
necessarily the init process's set). Cause unidentified. This is the single most likely thing
to bite when picking the work back up, since the node exits on its own whenever
`container system` is restarted.

**Status:** fix for the archive-404 half ready (`fix/archive-404-prestart`), not yet
submitted upstream. The `--privileged` gap is documented, intentional behavior in
socktainer's README (Virtualization.framework has no capability model to honor it,
hence the `--cap-add ALL` workaround above) — not something to file upstream.

### 7. Copying a directory into a container is unsupported

`docker cp <dir>/. c:/path` → `Error response from daemon: Something went wrong.`
`docker cp <dir> c:/path/` → `cannot copy directory`. Single files work.
DDEV's `CopyIntoVolume` (mkcert CA, global commands, traefik config) is exactly this, and
it *hangs* rather than erroring when driven through the API.

**Status:** fix ready (`fix/cp-directory-hang`, bounds the guest-preparation exec so a
wedged guest shell can't hang the request forever), not yet submitted upstream.

### 8. vmnet attachments go dark when idle

On socktainer-created networks the DNS sidecar becomes unreachable (100% packet loss) after
a few minutes of no traffic, while still `running`. Containers created afterwards reach each
other fine. Restarting the sidecar fixes it but gives it a **new IP**, and already-running
containers keep the stale `nameserver` in `/etc/resolv.conf`. A container receiving traffic
every 30s stayed reachable for 14/14 probes, so this reads as an idle timeout.

### 9. `ddev start` never returns, even once everything is healthy — fixed

With all three containers healthy, `ddev start` does not exit. Two runs were killed at 400s
and 200s. The last debug line is always:

```
2026-08-02T13:23:04.240 Getting traefik error output
```

i.e. `GetRouterConfigErrors()` in `pkg/ddevapp/router.go:201`. At that point the process is
sleeping with **no child processes** and makes **no further Docker API requests** (socktainer
logs nothing after it). The three `exec` sequences it issues against `ddev-router` all show
`POST /exec/<id>/start` followed by `GET /exec/<id>/json`, so those reads completed — the
blocked call is not one of them. It is not a sudo/`/etc/hosts` prompt: `appletest.ddev.site`
resolves publicly to 127.0.0.1, so no hostname edit is needed.

**Status: fixed, and the earlier diagnosis here was wrong.** `GetRouterConfigErrors()`
calls `dockerutil.Exec()`, which attaches stdout/stderr but not stdin. socktainer
refuses to upgrade such a request (`shouldUpgrade` requires `attachStdin`) and answers
with a chunked body on a keep-alive connection, while the Go client — having asked to
upgrade — reads the socket until it closes. Nothing closes it. The hijacked-path
close bug that [socktainer#347](https://github.com/socktainer/socktainer/pull/347)
addresses is real but is *not* on DDEV's path; #347 alone leaves this hanging (measured).
See the verification-run section above. `dockerutil.Exec` hangs against every container
under released socktainer, not only ddev-router.
[ddev/ddev#8660](https://github.com/ddev/ddev/pull/8660) (stop requesting the hijacked
stream) is still worth having as the DDEV-side belt-and-braces.

### 10. vmnet degrades until the `default` network's host bridge disappears

After about an hour of moderate churn, `192.168.64.1` was gone from every host interface
while `container network inspect default` still reported it as the gateway and containers
still got `192.168.64.x` addresses. Consequences: the dnsmasq workaround (item 2) dies with
`Can't assign requested address` and cannot be restarted, and containers on `default` have a
gateway that does not exist. `container system stop && container system start` (the remedy
the socktainer README gives for vmnet degradation) restored it; socktainer, dnsmasq and the
buildkit node all have to be restarted afterwards, in that order, and dnsmasq only binds
once a container is running on `default`.

### 12. Compose's recreate-by-rename collides with the hostname registry

`ddev debug rebuild` builds fine but fails at the router:

```text
Container ddev-router Error response from daemon: Failed to create container:
exists: "hostname(s) already exist: ["ddev-router"]"
```

Compose recreates a container by renaming the old one out of the way
(`POST /containers/create?name=5b86b57d76cd_ddev-router`) and creating a new one under the
original name. Renaming does not move the container's *hostname*, and Apple Container
enforces hostname uniqueness globally, so the create is rejected. The old router keeps
running and the site keeps serving, but the command fails.

Same shape as blocker 1: any DDEV path that goes through compose's recreate rather than
removing the container first will hit it. `ddev restart` is unaffected because it removes
containers first. Fixing it socktainer-side means having rename move the hostname
registration (or having create ignore a hostname held by a container that no longer
carries that name).

### 11. Smaller compatibility gaps

- `docker ps -a --filter name=<x>` **ignores the filter** — `docker rm -f $(docker ps -aq
  --filter name=ddev-appletest)` removed *every container on the machine*.
  **Status:** two bugs, both now fixed locally. `fix/filter-name-ignored` corrected the
  matching (substring, like real Docker) but the filter never reached it: the parser
  accepted the CLI's `{"key":{"value":true}}` encoding only for `label`, silently
  dropping `name`, `status` and `id`. Both halves are needed; neither submitted upstream.
- ~~Only one network per container~~ **— this was wrong.** Multi-homing at *create* time
  works: `docker inspect ddev-appletest-web` reports
  `ddev-appletest_default=192.168.253.7 ddev_default=192.168.254.11`, two networks on one
  container. What is unsupported is *hot-attach* — `docker network connect` after creation
  is a documented no-op (Virtualization.framework has no NIC hotplug). Compose-time
  multi-homing is fine, which is also why the multi-homed DNS bug
  ([rfay/socktainer#7](https://github.com/rfay/socktainer/issues/7)) can exist at all.
- Engine version is reported as `v1.51` (the API version), so DDEV warns
  "installed Docker version v1.51 is not supported, please update to version 25.0 or newer".
  **Status:** fix ready (`fix/version-engine-version`), not yet submitted upstream — but it
  does not silence the warning, since socktainer's own version (`1.2.x`, or `0.0.0-dev`
  for a local build) also fails DDEV's `>= 25.0` check. DDEV needs a socktainer-aware
  version check; that is the actual fix for the warning.
- **Not an Apple Container issue at all — listed here by mistake.** `chown -R` failing with
  `Operation not permitted` on a bind-mounted host directory is ordinary bind-mount
  behavior on macOS across every Docker provider. It only showed up because the
  experimental branch swaps the `ddev-global-cache` *volume* for a *bind mount*; a named
  volume never hits it. DDEV's `start.sh` runs `sudo chown -R … /mnt/ddev-global-cache/`
  under `set -e`, which then kills the web container.
  [ddev/ddev#8659](https://github.com/ddev/ddev/pull/8659) is still worth having — the chown
  is redundant in the common case, since a privileged utility container already did it — but
  it is a consequence of the workaround, not a provider gap. The same applies to any future
  move toward bind-mounting parts of the global cache (design option 2 below): anything that
  chowns inside a bind mount will fail on every provider, not just this one.
- Container names with underscores are silently skipped for DNS registration. **Status:**
  investigated further (live daemon + variants) and could not reproduce; not part of the
  rollout submissions. Worth closing `rfay/socktainer#14` as not-reproducible, or asking
  for a sharper repro.

---

## Experimental DDEV changes (branch `20260802_rfay_apple_container_experiment`)

The global-cache bind mount turns on **automatically when the provider is socktainer**, so it cannot be
forgotten in a terminal that lacks an environment variable. `DDEV_BIND_GLOBAL_CACHE=true` or
`=false` forces it either way for testing against other providers. On every other provider,
and whenever the provider cannot be reached, detection returns false and behavior is
byte-identical to before.

- `pkg/dockerutil/providers.go` — `IsSocktainer()` (matches `Server.Platform.Name`, following
  the existing `IsPodman()` pattern), plus `UseBindGlobalCache()`, `GlobalCacheSource()` and
  `GlobalCacheMount()`. When it is on, the global cache is the host directory
  `~/.ddev/global-cache-bind` bind-mounted at `/mnt/ddev-global-cache`, rather than the
  `ddev-global-cache` volume. These live in `dockerutil` rather than
  `globalconfig` because detection needs a Docker call and `dockerutil` already imports
  `globalconfig`, not the other way round.
- `pkg/ddevapp/ddevapp.go` — use the helper for all `/mnt/ddev-global-cache` mounts; skip
  creating the volume and `MkdirAll` the host dir instead; new `copyIntoGlobalCache()` that
  copies on the host when the global cache is bind-mounted (avoids the directory-copy gap).
- `pkg/ddevapp/commands.go`, `pkg/ddevapp/traefik.go` — route their copies through
  `copyIntoGlobalCache()`.
- `pkg/ddevapp/app_compose_template.yaml`, `router_compose_template.yaml`,
  `config.go`, `router.go` — `GlobalCacheMount` / `BindGlobalCache` template vars; omit the
  `volumes:` section when it would be empty; put project services on `ddev_default` only
  when the global cache is bind-mounted.

- `pkg/ddevapp/performance_mode.go`, `pkg/ddevapp/mutagen.go` — force Mutagen off when the
  provider is socktainer, with a `util.WarningOnce` explaining why. Mutagen mounts
  `project_mutagen` at both `/var/www` and `/tmp/project_mutagen`, which is two attachments
  of one block-backed volume *in a single container*, so the web container cannot boot at
  all (`VZErrorDomain Code=2 … storage device attachment is invalid`). `no_bind_mounts`
  gets its own warning and is likewise not honored there, since it depends on Mutagen.
  Verified against a project whose only setting was the global `performance_mode: mutagen`
  default: the warning appears once and the web container gets
  `bind <project> -> /var/www/html` with zero `project_mutagen` mounts.

Project-side settings still needed by hand:

```yaml
# .ddev/config.yaml
omit_containers: [ddev-ssh-agent]   # its socket volume is shared
```

`performance_mode: none` is no longer needed — it is applied automatically on socktainer.

```yaml
# .ddev/docker-compose.healthcheck.yaml and ~/.ddev/router-compose.healthcheck.yaml
services:
  web:   # and db: / ddev-router:
    healthcheck:
      test: ["CMD-SHELL", "/healthcheck.sh"]
      interval: 2s
      timeout: 10s
      retries: 60
      start_period: 10s
```

```dockerfile
# .ddev/web-build/Dockerfile — chown of a bind mount fails on every provider, not just this one
RUN sed -i 's|^sudo chown -R "$(id -u):$(id -g)" /mnt/ddev-global-cache/ /var/lib/php$|sudo chown -R "$(id -u):$(id -g)" /mnt/ddev-global-cache/ /var/lib/php \|\| true|' /start.sh
```

## Design options for the shared-volume problem

`ddev-global-cache` conflates three kinds of data with different sharing needs. Splitting it
is what makes every other option tractable:

| Content | Real requirement | Today |
|---|---|---|
| `mkcert/`, `global-commands/` | read-only, sourced from host `~/.ddev` | copied into the volume |
| `traefik/` config + certs | router reads, DDEV writes | shared volume |
| `npm`, `yarn`, `corepack`, `n_prefix`, composer | write-heavy; value comes from sharing *across* projects | shared volume |
| `bashhistory`, `mysqlhistory` | tiny, per-container | shared volume |

1. **Push it upstream to socktainer — highest leverage.** socktainer chose ext4 block images
   to back named volumes; it already bind-mounts host directories over virtiofs, and those
   *are* shareable. Backing `docker volume create` with a host directory under
   `~/.socktainer/volumes/` (even opt-in, e.g. `-o backend=virtiofs`) fixes DDEV with no DDEV
   changes at all, and fixes Compose stacks generally.
2. **Bind-mount the read-only content instead of copying it in.** `mkcert/` and
   `global-commands/` already exist as host files under `~/.ddev`. Mounting
   `~/.ddev/commands:/mnt/ddev-global-cache/global-commands:ro` deletes two `CopyIntoVolume`
   calls, sidesteps the directory-copy gap (#7), and is faster on *every* provider.
3. **Give the router its own volume for `traefik/`.** It is the only consumer. Single-file
   `docker cp` works on socktainer, so per-file pushes also avoid the directory-copy hang.
4. **For the package caches, choose per-project or per-host.** A per-project cache volume
   (`ddev-cache-<project>`) keeps block-device performance but loses cross-project reuse and
   multiplies disk. A host bind mount keeps sharing at virtiofs cost. Worth measuring first:
   DDEV avoids bind mounts because Docker Desktop's gRPC-FUSE is slow, and that assumption
   may simply not hold for Apple Container's virtiofs.
5. **Fix `GetExistingDBType` regardless of provider.** It mounts the db volume only to read a
   version file, which is why `ddev start` fails against an already-running project. Reading
   it via `exec` when the db container is up, or from a container label written at creation,
   is faster everywhere and removes one concurrent-attach case. **Shipped:**
   [ddev/ddev#8658](https://github.com/ddev/ddev/pull/8658).
6. **Make the global-cache `chown` non-fatal in `start.sh`.** Any bind mount fails it — this
   is normal macOS bind-mount behavior on every provider, not something Apple Container
   introduces — and `set -e` then kills the web container. It is a no-op whenever ownership
   already matches, which it does, since DDEV runs as the host uid. Belongs in
   `containers/ddev-webserver/ddev-webserver-base-scripts/start.sh`. **Shipped:**
   [ddev/ddev#8659](https://github.com/ddev/ddev/pull/8659). This is a prerequisite for *any*
   bind-mount-based approach to the global cache, not a socktainer-specific patch.

7. **Seed with a transient writer, then mount read-only.** Because N containers can hold the
   same volume `:ro` at once (blocker 1), the read-mostly part of the cache needs no bind
   mount and no upstream change. A short-lived writer attaches read-write, seeds the content,
   and exits; the project containers then all mount `:ro`. Verified end to end:

   ```text
   1. transient writer seeds, exits          v1
   2. two long-lived :ro readers see it      v1
   3. re-seed WHILE readers are up           500  (blocked)
   4. write from inside a :ro reader         Read-only file system
   5. re-seed after readers are gone         v2
   ```

   Step 3 is the only real constraint, and it lines up with DDEV's lifecycle: re-seeding
   requires every reader to be down, which is exactly what `ddev start` / `ddev restart`
   already does, since the project containers are recreated at that point anyway. Nothing
   needs to write to this content mid-session.

   Step 4 is what decides the split. Anything written *continuously* by a running container —
   the npm/Composer caches, `n_prefix`, `corepack`, shell and mysql history — cannot live in
   the read-only volume and has to move out. Note this is not only a DDEV-side change:
   `ddev-webserver`'s `start.sh` currently does `mkdir -p` and `chown -R` under
   `/mnt/ddev-global-cache/`, which fails on a read-only mount.

   `traefik/` does not need to be in the shared volume at all — the router is its only
   consumer, so it can hold a router-exclusive read-write volume, with DDEV pushing updates
   the way it already does, by copying single files into the running router.

8. **A cache service container — but it has to serve a protocol, not a filesystem.** A
   long-lived writer container that owns the volume while others mount it `:ro` **does not
   work**; the exclusivity is on the block device, and `:ro` readers coexist only with each
   other:

   ```text
   writer holds RW, is it alive?         seed
   reader :ro  while writer holds RW ->  500
   reader rw   while writer holds RW ->  500
   after writer removed, :ro         ->  seed
   ```

   The idea does work if the owner exposes the cache over the *network* instead of the
   filesystem: one container attaches the volume read-write exclusively, and everything else
   talks to it over HTTP. A single caching forward proxy (squid, or nginx `proxy_cache`)
   pointed at by `HTTP_PROXY`/`HTTPS_PROXY` would cover npm, Composer, apt and pip at once,
   and DDEV already installs a trusted mkcert root CA in its containers, which is what HTTPS
   interception needs. Per-ecosystem alternatives (Verdaccio for npm, a Composer mirror) are
   narrower but avoid interception.

   The limit is that this only covers *downloads*. `n_prefix` and `corepack` are install
   prefixes read at runtime, not download caches, and shell/mysql history is small writable
   state — neither can be proxied, so both still need a real filesystem (small per-project
   volumes, or the host).

Items 2, 5 and 6 are provider-independent improvements that happen to unblock this path, and
could go in as a normal PR ahead of anything Apple-Container-specific. Option 7 is the most
promising Apple-Container-specific direction, and unlike the global-cache bind mount
currently implemented for socktainer, it does not trade away volume performance.
Options 7 and 8 compose: seed-then-`:ro` for the read-only content, a proxy for the download
caches, small per-project volumes for writable state.

The ssh-agent socket directory is the one case not to bind-mount — unix sockets over virtiofs
are unreliable. Omitting `ddev-ssh-agent` or using the host's own agent are the realistic
options there.

## State this session left behind

- The three new socktainer fixes live **uncommitted** in the working tree of
  `~/workspace/socktainer` on branch `tmp/rollout-verify` (the seven queued fixes merged
  on top of `main`). Files touched:
  `Sources/socktainer/Routes/Containers/ExecRoutes.swift` (close-on-EOF correction to
  #347, plus the `shouldUpgrade` gating fix),
  `Sources/socktainer/DNS/SocktainerDNSServer.swift` (EDNS0 response truncation),
  `Sources/socktainer/Utilities/DockerFilterUtility.swift` (dict-form filter parsing),
  and new tests in `Tests/socktainerTests/{DNS,Utilities}/`. Rebuild with
  `make release`; run with `~/workspace/socktainer/.build/release/socktainer`.
- **The patched local build is running** (`~/tmp/socktainer.log`), the appletest project is
  up and healthy, and the site serves 200 at the router's container IP. The global docker
  context is still `orbstack` — the verification above ran with `DOCKER_CONTEXT=socktainer`
  in the environment, which both `docker` and `ddev` honor, so nothing global was switched.
- Note `brew services start socktainer` is *not* a usable "restore the normal daemon" step:
  the service runs with `HOME=/opt/homebrew/var/run/socktainer`, so it binds
  `/opt/homebrew/var/run/socktainer/.socktainer/container.sock` and leaves the
  `socktainer` docker context (which points at `~/.socktainer/container.sock`) talking to
  a dead socket. Run the daemon by hand, as this and the previous session did.
- A root `dnsmasq` is bound to `192.168.64.1:53` (item 2). Stop it with
  `sudo pkill -f "listen-address=192.168.64.1"`. Without it, no container has DNS.
- `traefik_monitor_port` is set to **11999** (10999 is OrbStack's). Revert it to 10999 when
  done with socktainer if that matters for other projects.
- The global `~/.ddev/router-compose.healthcheck.yaml` override is **in place** (it is
  needed for ddev-router to report healthy); a copy of the pre-session state is at
  `~/tmp/router-compose.healthcheck.yaml.bak`. It also affects OrbStack projects.
- Test project at `~/tmp/appletest`, left started; `keepalive` and
  `buildx_buildkit_default` left running.
