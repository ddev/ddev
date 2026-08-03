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

`ddev start` on a plain PHP project reaches **web + db + ddev-router all healthy**
(`ready in 10.5s` for each group). Traefik is reachable from the host at the router's
container IP and routes correctly. The remaining gap is the last hop:
traefik → web returns **502** because name resolution against socktainer's DNS fails for
musl/Go clients (see "DNS returns unusable AAAA" below).

So: not working end-to-end yet, but much further than "impossible".

## Upstream engagement status (2026-08-03)

Fixes for seven of the eleven blockers below are written, tested, and DCO-signed-off
against socktainer, tracked in the
[`rfay/socktainer` fork's rollout plan](https://github.com/rfay/socktainer/blob/fix/exec-hijack-close/UPSTREAM_ROLLOUT.md),
ranked by DDEV relevance vs. diff risk. Submitting upstream one at a time rather than
bundling. Status so far:

- **Filed upstream:** [socktainer/socktainer#346](https://github.com/socktainer/socktainer/issues/346)
  (exec-hijack hang, blocker 9) / [socktainer/socktainer#347](https://github.com/socktainer/socktainer/pull/347)
  (fix, open, DCO passing, awaiting review)
- **Already filed by someone else, corroborated independently:**
  [socktainer/socktainer#329](https://github.com/socktainer/socktainer/issues/329)
  (malformed EDNS0, blocker 3)
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

**Status:** root cause is a malformed EDNS0 response, filed upstream (not by us) as
[socktainer/socktainer#329](https://github.com/socktainer/socktainer/issues/329) — the
highest-impact fix on this list, since it breaks any Go/musl client's resolution on a
socktainer network, not just DDEV's. A second, distinct DNS bug turned up investigating
this one: multi-homed hostnames get an arbitrary network's address instead of the
querying client's — fix ready (`fix/dns-wrong-network-address`), not yet submitted.

### 4. Published ports are accepted but reset

`docker run -d -p 127.0.0.1:38080:80 nginx:alpine` → the `container` helper listens on
127.0.0.1:38080, but connections get `Recv failure: Connection reset by peer`. The same
nginx answers 200 on its container IP directly from the host. Same for DDEV's router ports.

Silver lining: Apple Container makes container IPs routable from the host, so a DDEV
"Apple Container mode" could skip port publishing entirely and point hostnames at the
router's container IP.

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

### 9. `ddev start` never returns, even once everything is healthy — unresolved

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

**Status: root cause identified and fixed, both sides.** `GetRouterConfigErrors()` calls
`dockerutil.Exec()`, which hangs because it waits for EOF on a hijacked exec-start
connection that socktainer never closes after the process exits — confirmed at the raw
HTTP level (output arrives correctly, the connection just never tears down). Filed as
[socktainer/socktainer#346](https://github.com/socktainer/socktainer/issues/346), fixed
by [socktainer/socktainer#347](https://github.com/socktainer/socktainer/pull/347) (open).
DDEV also doesn't need the hijacked stream at all —
[ddev/ddev#8660](https://github.com/ddev/ddev/pull/8660) stops requesting it, so this
stops hanging regardless of which side merges first.

### 10. vmnet degrades until the `default` network's host bridge disappears

After about an hour of moderate churn, `192.168.64.1` was gone from every host interface
while `container network inspect default` still reported it as the gateway and containers
still got `192.168.64.x` addresses. Consequences: the dnsmasq workaround (item 2) dies with
`Can't assign requested address` and cannot be restarted, and containers on `default` have a
gateway that does not exist. `container system stop && container system start` (the remedy
the socktainer README gives for vmnet degradation) restored it; socktainer, dnsmasq and the
buildkit node all have to be restarted afterwards, in that order, and dnsmasq only binds
once a container is running on `default`.

### 11. Smaller compatibility gaps

- `docker ps -a --filter name=<x>` **ignores the filter** — `docker rm -f $(docker ps -aq
  --filter name=ddev-appletest)` removed *every container on the machine*.
  **Status:** fix ready (`fix/filter-name-ignored`, matches by substring like real Docker),
  not yet submitted upstream.
- Only one network per container: `docker network connect` is a documented no-op, so DDEV's
  web container (`["default", "ddev_default"]`) actually lands on one of them. Documented,
  intentional limitation in socktainer's README (Virtualization.framework has no NIC
  hotplug) — no rollout fix.
- Engine version is reported as `v1.51` (the API version), so DDEV warns
  "installed Docker version v1.51 is not supported, please update to version 25.0 or newer".
  **Status:** fix ready (`fix/version-engine-version`), not yet submitted upstream.
- `chown -R` on a virtiofs bind mount fails with `Operation not permitted`; DDEV's
  `start.sh` runs `sudo chown -R … /mnt/ddev-global-cache/` under `set -e`, which kills the
  web container. **Status:** fixed DDEV-side, open as
  [ddev/ddev#8659](https://github.com/ddev/ddev/pull/8659) (the chown is redundant in the
  common case anyway, since a privileged utility container already did it) — no socktainer
  change needed.
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

Project-side settings still needed by hand:

```yaml
# .ddev/config.yaml
performance_mode: none              # mutagen mounts one volume twice
omit_containers: [ddev-ssh-agent]   # its socket volume is shared
```

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
# .ddev/web-build/Dockerfile — chown of the virtiofs bind mount always fails
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
6. **Make the global-cache `chown` non-fatal in `start.sh`.** On virtiofs it always fails and
   `set -e` kills the web container. It is a no-op whenever ownership already matches, which
   it does, since DDEV runs as the host uid. Belongs in
   `containers/ddev-webserver/ddev-webserver-base-scripts/start.sh`. **Shipped:**
   [ddev/ddev#8659](https://github.com/ddev/ddev/pull/8659).

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

- `socktainer` was restarted by hand and now logs to `~/tmp/socktainer.log`
  (was previously running in a terminal).
- A root `dnsmasq` is bound to `192.168.64.1:53` (item 2). Stop it with
  `sudo pkill -f "listen-address=192.168.64.1"`. Without it, no container has DNS.
- `traefik_monitor_port` was temporarily changed to 11999 (OrbStack's router owns 10999)
  and has been **reverted to 10999**.
- The global `~/.ddev/router-compose.healthcheck.yaml` override was moved to
  `~/tmp/router-compose.healthcheck.yaml.bak` so it doesn't affect OrbStack projects.
- Test project at `~/tmp/appletest`; test networks removed; `ddev-buildnet` and
  `buildx_buildkit_default` left in place.
