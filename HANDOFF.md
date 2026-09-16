# DDEV on Apple Container + socktainer — investigation notes

Working notes for [#7372](https://github.com/ddev/ddev/issues/7372), started 2026-08-02 and
last verified 2026-09-16. This documents an exploration, not a feature: the DDEV changes here
only take effect when the Docker provider is socktainer, and this file is expected to be
deleted before any of it merges.

Environment as last verified: Apple `container` 1.4.1 (signed installer, **not** Homebrew — see
below), `socktainer` built from
[`tmp/combined-verify-4`](https://github.com/rfay/socktainer/tree/tmp/combined-verify-4), cut
2026-09-16 from current upstream `main` (replaces `tmp/combined-verify-3`), docker CLI 29.7.2,
docker context `socktainer`, macOS 27 (26A5421a), `ddev` from branch
[`20260802_rfay_apple_container_experiment`](https://github.com/rfay/ddev/tree/20260802_rfay_apple_container_experiment).

## Where this stands

**It works end to end, and it cannot ship.** Two DDEV projects run side by side on Apple
Container — image builds, database, router, published ports, `ddev ssh`/`exec`/`mysql` — but
only against a socktainer built from the fork with unmerged fixes. Nothing changes for DDEV
until upstream socktainer carries them.

The gate is socktainer's maturity, not DDEV-side work. In the three weeks to 2026-09-16, one of
the ten fixes tracked here was fully subsumed by upstream's own (better) design —
[`fix/archive-404-prestart`](https://github.com/rfay/socktainer/tree/fix/archive-404-prestart)
is dropped from the fork; see "Resolved" below — leaving nine. **Still no PRs are being filed**
from the fork: searching `socktainer/socktainer` for any PR referencing `rfay` returns nothing.
The nine remaining fixes span DNS, exec hijacking, archive internals and healthcheck timing, and
each would need sustained review defense; the issues at
[rfay/socktainer](https://github.com/rfay/socktainer/issues) carry the reproductions instead,
with the branches as reference implementations. The natural time to revisit is a socktainer
release that closes several of them at once — the tagged release is still v1.2.1 (2026-08-01)
despite ~20 PRs merged to `main` since, so nothing has actually shipped in three weeks even where
upstream has fixed things.

Three DDEV-side hardenings split out of this work are merged and are worth having on their
own: [#8657](https://github.com/ddev/ddev/pull/8657) (bound host ports fallback),
[#8658](https://github.com/ddev/ddev/pull/8658) (`GetExistingDBType` via exec) and
[#8659](https://github.com/ddev/ddev/pull/8659) (non-fatal global-cache chown).
[#8660](https://github.com/ddev/ddev/pull/8660) (stop hijacking the exec stream) was closed,
so that hang is covered only by socktainer's `fix/exec-hijack-close`.

### What still has to be done by hand

- `omit_containers: [ddev-ssh-agent]` — its socket volume is shared, and unix sockets over
  virtiofs are unreliable, so bind-mounting is not an answer either.
- `router_http_port`/`router_https_port` above 1024, distinct per project.
- A `traefik_monitor_port` that does not collide with another provider's router.
- The cold-start recipe below, after any `container system` restart.
- `ddev restart` rather than `ddev start` over an already-running project.
- Pre-create `buildx_buildkit_default` by hand before the first `ddev start` — see "buildx's
  buildkit node" below. Without it, `ddev start` fails outright on `tmp/combined-verify-4`.

Mutagen needs no per-project setting: it is forced off automatically on this provider.
Healthcheck overrides are no longer needed and now actively cause harm — see the resolved
list.

## socktainer fixes carried in the fork

Eleven fixes came out of this investigation, each on its own branch in
[`rfay/socktainer`](https://github.com/rfay/socktainer). Nine are combined and verified
together on
[`tmp/combined-verify-4`](https://github.com/rfay/socktainer/tree/tmp/combined-verify-4), cut
from upstream `main` on 2026-09-16 (replaces `tmp/combined-verify-3`, cut 2026-08-28). Two of
the original eleven are dropped, both because upstream fixed the same bug independently, not
from this fork: `fix/privileged-cap-all`
([socktainer#364](https://github.com/socktainer/socktainer/pull/364)) and, new this round,
`fix/archive-404-prestart` — see "Resolved" below. None of the remaining nine is merged
upstream; searching `socktainer/socktainer` for a PR referencing `rfay` returns nothing, so "no
PRs are being filed" is still accurate three weeks on.

### Blocks `ddev start` from working at all

| Fix | Branch | Issue |
|---|---|---|
| Exec was upgraded only when stdin was attached, so `dockerutil.Exec` (stdout/stderr only) never got a reply that closed — `ddev start` hung forever at "Getting traefik error output". Also closes the channel on output EOF | [`fix/exec-hijack-close`](https://github.com/rfay/socktainer/tree/fix/exec-hijack-close) | [#8](https://github.com/rfay/socktainer/issues/8) — upstream tried [socktainer#347](https://github.com/socktainer/socktainer/pull/347) for the same underlying [socktainer#346](https://github.com/socktainer/socktainer/issues/346), closed unmerged (breaks buildx's `buildctl dial-stdio`, which legitimately outlives a fixed bound) |
| `/version` reported the Docker API version instead of socktainer's own, tripping DDEV's minimum-version check | [`fix/version-engine-version`](https://github.com/rfay/socktainer/tree/fix/version-engine-version) | [#13](https://github.com/rfay/socktainer/issues/13) — verified still current: `Package.swift` hardcodes `DOCKER_ENGINE_API_MAX_VERSION` to `"v1.51"` on upstream `main` today, "v" prefix and all |
| Malformed EDNS0 responses break Go/musl resolvers — the actual cause of the router's 502s | [`fix/dns-edns0-truncation`](https://github.com/rfay/socktainer/tree/fix/dns-edns0-truncation) | [#15](https://github.com/rfay/socktainer/issues/15), upstream [#329](https://github.com/socktainer/socktainer/issues/329) — upstream's own fix, [socktainer#348](https://github.com/socktainer/socktainer/pull/348), is open and unmerged (`REVIEW_REQUIRED`) as of 2026-09-16 |

### Blocks specific DDEV scenarios

| Fix | Branch | Issue |
|---|---|---|
| `HostConfig.PortBindings` always nil on inspect, so a second project could never start | [`fix/port-bindings-inspect`](https://github.com/rfay/socktainer/tree/fix/port-bindings-inspect) | none filed |
| `docker cp` of a directory hangs forever — what DDEV's `CopyIntoVolume` does. Side-stepped for the global cache by the bind mount, not fixed | [`fix/cp-directory-hang`](https://github.com/rfay/socktainer/tree/fix/cp-directory-hang) | [#11](https://github.com/rfay/socktainer/issues/11) |
| DNS gives multi-homed hostnames the wrong network's address | [`fix/dns-wrong-network-address`](https://github.com/rfay/socktainer/tree/fix/dns-wrong-network-address) | [#7](https://github.com/rfay/socktainer/issues/7) |

### Real bugs, not currently blocking DDEV

| Fix | Branch | Issue |
|---|---|---|
| Healthcheck status regression and stalled-probe freeze | [`fix/healthcheck-timing`](https://github.com/rfay/socktainer/tree/fix/healthcheck-timing) | [#12](https://github.com/rfay/socktainer/issues/12) |
| `--filter name=`/`status=`/`id=` matching semantics wrong — DDEV never hits it, `FindContainerByName` re-checks the name | [`fix/filter-name-ignored`](https://github.com/rfay/socktainer/tree/fix/filter-name-ignored) | [#9](https://github.com/rfay/socktainer/issues/9) |
| Dict-form filter values dropped for every key but `label` — the other half of the same bug | [`fix/dict-filter-parsing`](https://github.com/rfay/socktainer/tree/fix/dict-filter-parsing) | [#9](https://github.com/rfay/socktainer/issues/9) |

[rfay/socktainer#16](https://github.com/rfay/socktainer/issues/16) has no fix branch: archive
GET aborts on a dangling symlink (`/etc/mtab → /proc/mounts`), and PathStat reports the raw
ext4 mode where the Docker CLI expects a Go `os.FileMode`. buildx works without it.

Upstream keeps a Docker Engine API endpoint-coverage table at
[socktainer#14](https://github.com/socktainer/socktainer/issues/14) — useful for a first
orientation on what's implemented at all (e.g. it independently confirms the
`/networks/{id}/connect` hot-attach gap noted below), but **it isn't a live source**: the body
says "Last updated April 4th, 2026" and hasn't moved since, the last comment on it is from
2025-09-22, and the highest PR it cites (#191) is far behind everything this investigation
tracks (#348, #372, #374, #379, #389, #391). Treat it as a one-time map, not current status.

`tmp/combined-verify-4` rebases cleanly onto current upstream `main` for eight of the nine fixes;
`fix/dns-edns0-truncation` and `fix/dns-wrong-network-address` both touch
`SocktainerDNSServer.swift` and needed a hand-merge (git `rerere` replayed the same resolution
used building `combined-verify-3`) — verified coherent, both fixes' logic present, by reading
the merged diff directly.

## Open blockers

### Named volumes cannot be shared read-write — the structural one

Apple Container backs each named volume with an ext4 block image attached to one VM, and
Virtualization.framework will not attach one image to a second VM while the first holds it.
Apple closed [apple/container#1133](https://github.com/apple/container/issues/1133) as
working-as-designed: one VM per container is the model.

```bash
docker volume create voltest1
docker run -d --name volA -v voltest1:/data alpine sleep 300
docker run --rm -v voltest1:/data alpine ls /data
# Error Domain=VZErrorDomain Code=2 "The storage device attachment is invalid."
```

**Read-only attach is not exclusive**, which is the nuance that leaves a way through: a volume
is either held read-write by exactly one container, or read-only by any number.

| Held by | New `:ro` attach | New rw attach |
|---|---|---|
| nothing | works | works |
| one rw container | **fails** | **fails** |
| one or more `:ro` containers | **works** (tested with 3) | **fails** |

Host bind mounts (virtiofs) are shareable read-write. `docker volume create --driver local -o
type=none -o device=… -o o=bind` is ignored by socktainer — it still makes a block volume.

Where it hits DDEV: `ddev-global-cache` (web, db, router, ssh-agent and every
`RunSimpleContainer` helper), `ddev-ssh-agent_socket_dir` (ssh-agent and web), and Mutagen,
which mounts `project_mutagen` twice inside the web container alone. `RestoreSnapshot` was
checked and is safe — it removes the db container before touching the volume. Every
`RunSimpleContainer` call site mounting the db volume was swept; `db.go` and `start-chown`
were the only two, and both are fixed.

**Worth checking:** [lunguini/gocker](https://github.com/lunguini/gocker), a separate Go
implementation of the same idea as socktainer (Docker-compatible CLI/API daemon over Apple
Container), has an `isolation` setting — `full` / `hybrid` / `shared` — controlling whether
every container gets its own microVM or several share one. `shared`/`hybrid` would sidestep
this entire structural blocker outright, since a volume mounted RW by two containers in the
same VM is a non-issue. Found by general search, not a link from socktainer's own issues/PRs —
unconfirmed whether it's actively developed or how complete its Docker API coverage is; worth a
look before assuming socktainer is the only path here.

### Hostname uniqueness collides with compose's recreate

```text
Container ddev-appletest-db Error response from daemon: Failed to create container:
exists: "hostname(s) already exist: ["appletest-db", "appletest-db"]"
```

Compose creates the new container before removing the old one. Apple Container enforces
hostname uniqueness across **every** container irrespective of state
(`ContainerAPIService/Server/Containers/ContainersService.swift:310`), so the create is
rejected; socktainer cannot move the old one aside either, since
`POST /containers/{id}/rename` is `NotImplemented`. The hostname appears twice because the
container has two network attachments sharing one hostname.

The failure is clean — the running project keeps serving — and **`ddev restart` is unaffected**,
because `Stop()` runs `compose down` first and frees the hostnames. Unchanged through
`container` 1.3.0. A real fix needs Apple Container to scope uniqueness to running containers,
or socktainer to stop using the container name as the attachment hostname.

`ddev restart` recreating the db/web containers every time is not a bug: `Restart()` is
`Stop()` + `Start()`, so `compose up` has nothing to converge against, on any provider. It is
just slower here, because each fresh db container pays a hypervisor block-attach cost.

### Ports below 1024 cannot be published

Apple Container's port forwarder runs unprivileged:

```text
Failed to start container: … invalidArgument: "Permission denied while binding to
host port 443. Binding to ports below 1024 requires root privileges."
```

Fix: `ddev config --router-http-port=8080 --router-https-port=8443`. This only shows up when
80/443 are actually free — with another router holding them, DDEV picks ephemeral high ports
and sidesteps it by accident.

Not a new class of problem: rootless Podman is a supported provider with the same restriction,
and the docs already tell users to configure unprivileged ports. The macOS-specific wrinkle is
that Linux offers `net.ipv4.ip_unprivileged_port_start=0` and macOS has no equivalent, so here
it is mandatory rather than recommended.

Refined by [apple/container#1985](https://github.com/apple/container/issues/1985) (opened
2026-09-12): it's specifically an *explicit* low bind address that fails —
`-p 127.0.0.1:80:80` still needs root — while `-p 80:80` or `-p 0.0.0.0:80:80` now succeeds
unprivileged. DDEV always publishes without a bind address, so this doesn't change anything
here; noted in case a future DDEV change starts binding to a specific host IP.

### No DNS on the built-in `default` network

Containers get `nameserver 192.168.64.1` (the vmnet gateway) but nothing listens there, so
every lookup is `connection refused`. This breaks anything on `default`, including Apple's own
build VM. The cold-start recipe's dnsmasq forwards that address to socktainer's own DNS.

### vmnet goes dark when idle, and degrades over hours

On socktainer-created networks the DNS sidecar becomes unreachable (100% packet loss) after a
few minutes of no traffic while still `running`; a container receiving traffic every 30s
stayed reachable 14/14. Restarting the sidecar gives it a new IP, which already-running
containers never pick up.

Worse, after about an hour of churn `192.168.64.1` disappeared from every host interface while
`container network inspect default` still reported it as the gateway. The dnsmasq workaround
then dies with `Can't assign requested address` and cannot restart.
`container system stop && container system start` restores it; socktainer, dnsmasq and the
buildkit node all need restarting afterwards, in that order, and dnsmasq only binds once a
container is running on `default`.

Now tracked upstream, none fixed: [apple/container#1884](https://github.com/apple/container/issues/1884)
(open since 2026-07-02) matches the hours-of-churn degradation directly — a hung container can
drop a custom network's gateway until it's killed. [apple/container#2135](https://github.com/apple/container/issues/2135)
(opened 2026-08-30) is a narrower DNS symptom on `default`: some external hostnames SERVFAIL from
inside a container while the host resolves them fine. [apple/container#2275](https://github.com/apple/container/issues/2275)
(opened 2026-09-15, macOS 27.0) is a new one worth watching: the apiserver can hang forever on
startup when `com.apple.pfd` is unresponsive, so every `container` command hangs with it.

### buildx's buildkit node

On 1.3.0 with `fix/archive-404-prestart`, buildx bootstrapped its own node and both project
images built — the manual pre-create was a fallback, not a prerequisite, on that build. **On
`tmp/combined-verify-4` it's a prerequisite again**, for a different reason: buildx's own
bootstrap now fails outright. It creates the node, then PUTs files into `/etc` on the
created-but-not-started container, which hits upstream's *new* pre-start-injection path
(`stagePreStartInjection()`, replacing the fork's dropped `fix/archive-404-prestart` — see
"Resolved") and throws `ArchiveUtilityError.archiveReadFailed(... ArchiveError error 9
[unableToCloseArchive] ...)`. `docker-compose build` then fails with a bare "Something went
wrong." Filed as [rfay/socktainer#17](https://github.com/rfay/socktainer/issues/17). Pre-creating
the node first sidesteps it entirely — the container is already running by the time buildx PUTs
into it, so the PUT goes through the ordinary running-container path instead of the new
never-started one. Confirmed: `ddev start` fails without the pre-create, succeeds with it, on
both `appletest` and `appletest2`. Always pre-create it:

```bash
docker run -d --name buildx_buildkit_default --cap-add ALL \
  moby/buildkit:buildx-stable-1 --allow-insecure-entitlement=network.host
```

`--cap-add ALL` is required because `--privileged` grants no capabilities — documented,
intentional socktainer behavior, since Virtualization.framework has no capability model.

**A node that has exited must be recreated, never restarted.** buildx will happily restart it
and every build then fails with `rbind … operation not permitted`. Capabilities do survive the
restart (`CapEff: 000001ffffffffff` measured either side), so the cause is unidentified. A node
that never exited is fine across a socktainer restart.

### Smaller gaps

- **Hot-attach of networks is unsupported.** Multi-homing at *create* time works;
  `docker network connect` afterwards is a documented no-op, since Virtualization.framework
  has no NIC hotplug.
- **`docker ps -a --filter name=` is ignored**, so `docker rm -f $(docker ps -aq --filter
  name=ddev-appletest)` removes *every container on the machine*. Both halves fixed in the
  fork; be careful running that idiom against stock socktainer.
- **`chown -R` fails on a bind mount** with `Operation not permitted`. Ordinary macOS
  bind-mount behavior on every provider, surfaced here only because the branch swaps the
  global-cache volume for a bind mount. Covered by
  [#8659](https://github.com/ddev/ddev/pull/8659).

## Recipes

### Teardown and clean rebuild

**Always `ddev poweroff` before switching Docker contexts**, in either direction. DDEV's router
and its host ports belong to whichever context created them, and a router left running under
the previous context keeps 8080/8443/11999 bound, so the next context's router cannot bind
them.

```bash
# 1. tear everything down
ddev poweroff
container rm -f $(container ls -aq)              # every instance, not just DDEV's
pkill -f "\.build/release/socktainer$"
container system stop
docker context rm socktainer                     # recreated in step 3
rm -f ~/.socktainer/container.sock               # stale socket from the killed daemon

# 2. clean-room build of the branch under test
cd ~/workspace/socktainer
git checkout tmp/combined-verify-4
rm -rf .build
swift build -c release                           # ~450s from empty .build

# 3. bring both systems back up
/usr/local/bin/container system start            # the signed installer explicitly
cd ~/workspace/socktainer
nohup ./.build/release/socktainer >> ~/tmp/socktainer.log 2>&1 &
disown
docker context create socktainer \
  --docker "host=unix:///Users/$(whoami)/.socktainer/container.sock" \
  --description "Socktainer — Docker API over Apple Container"
docker context use socktainer

# 4. confirm the signed installer is what is running
/usr/local/bin/container system status | grep installRoot        # expect /usr/local/
```

Homebrew's `container` formula tracks upstream and will update itself out from under the
signed installer, and `/opt/homebrew/bin` sits ahead of `/usr/local/bin` on `PATH` — so always
invoke `/usr/local/bin/container` explicitly for anything touching `system start`/`system
status`, or the check in step 4 will look right by coincidence while every command runs the
wrong binary. As of 2026-09-16 this is Homebrew's stated position, not a transient drift risk:
[socktainer/socktainer#392](https://github.com/socktainer/socktainer/issues/392) asked
homebrew-core to make its `socktainer` formula's `depends_on "container"` conditional so it
wouldn't shadow the signed installer, and the maintainer refused outright — "if a user wants to
use socktainer on Homebrew Core, they MUST NOT use Apple's pkg," since a self-contained prefix is
one of Homebrew's own rules. **Do not `brew install socktainer` from homebrew-core at all on
this machine.** It's sharper than passive drift, too: socktainer's `main` (not yet in the tagged
v1.2.1 release) gained an auto-start bootstrap in
[socktainer#374](https://github.com/socktainer/socktainer/pull/374) — on launch it runs
`container system start` against whatever `container` binary `discoverContainerCLIPath()` finds
on `PATH`, *before* its own version-compatibility check — so a machine with both installs
present could have socktainer boot the wrong one ahead of the check that exists to catch exactly
that. It's a no-op here only because step 3 above always starts `/usr/local/bin/container`
before socktainer ever runs, so `ensureRunning()` sees it already reachable and does nothing;
this order still matters even building from source, since the bootstrap is code in `main`, not a
Homebrew packaging artifact. #392 is open with an agreed fix (stop auto-starting; fail fast with
an explicit error instead) but no PR yet.

`docker context create` may print `context "socktainer" already exists` right after the `rm`;
harmless, `docker context inspect` confirms the fresh socket either way.

### Cold start

The environment does not survive idling: the `default` network's host bridge is torn down once
nothing runs on it, which kills DNS. Run these before `ddev start`, and again after any
`container system` restart.

```bash
# 1. keep the default network (and therefore 192.168.64.1) alive
docker run -d --name keepalive ddev/ddev-utilities:latest sleep 100000

# 2. resolver on the gateway, forwarding to socktainer's DNS. Must come AFTER step 1;
#    it fails with "Can't assign requested address" if the bridge does not exist yet.
sudo pkill -f "listen-address=192.168.64.1"
sudo /opt/homebrew/sbin/dnsmasq --keep-in-foreground \
  --listen-address=192.168.64.1 --bind-interfaces --port=53 \
  --no-resolv --server=127.0.0.1#2054

# 3. sanity check
docker run --rm ddev/ddev-utilities:latest nslookup registry-1.docker.io
```

Then `ddev restart` after any socktainer daemon restart: health status resets to `starting`
for every container when the daemon re-adopts them and stays there, so anything waiting for
healthy hangs. The site keeps serving throughout — only the reported status is stale.

To reach a project at the router's container IP rather than a published port:

```bash
RIP=$(docker inspect ddev-router --format '{{range .NetworkSettings.Networks}}{{.IPAddress}} {{end}}' | awk '{print $1}')
curl -sS -k -o /dev/null -w '%{http_code}\n' --resolve appletest.ddev.site:8443:$RIP https://appletest.ddev.site:8443/
```

## DDEV changes on the branch

The global-cache bind mount turns on **automatically when the provider is socktainer**, so it
cannot be forgotten in a terminal without an environment variable. `DDEV_BIND_GLOBAL_CACHE`
forces it either way for testing. On every other provider, and whenever the provider cannot be
reached, detection returns false and behavior is unchanged.

- `pkg/dockerutil/providers.go` — `IsSocktainer()` (matches `Server.Platform.Name`, following
  the existing `IsPodman()` pattern) and `IsAppleContainer()`, plus `UseBindGlobalCache()`,
  `GlobalCacheSource()` and `GlobalCacheMount()`. When on, the global cache is
  `~/.ddev/global-cache-bind` bind-mounted at `/mnt/ddev-global-cache` instead of the
  `ddev-global-cache` volume. These live in `dockerutil` because detection needs a Docker call
  and `dockerutil` already imports `globalconfig`, not the other way round.
- `pkg/ddevapp/ddevapp.go` — route every `/mnt/ddev-global-cache` mount through the helpers;
  skip creating the volume and `MkdirAll` the host dir instead; `copyIntoGlobalCache()` copies
  on the host when bind-mounted, avoiding the directory-copy gap. `mergeCopyDir()` copies into
  the directories that are already there rather than recreating them, which is what keeps
  Traefik's watch alive.
- `pkg/ddevapp/traefik.go` — `NotifyRouterOfTraefikConfigChange()` touches the config from
  inside the router so Traefik re-reads it; `listGlobalCacheFiles()`/`removeGlobalCacheFiles()`
  sync stale configs against whichever backing is in use.
- `pkg/ddevapp/performance_mode.go`, `mutagen.go` — force Mutagen off with a `WarningOnce`
  explaining why. `no_bind_mounts` gets its own warning, since it depends on Mutagen.
- Compose templates, `config.go`, `router.go` — `GlobalCacheMount`/`BindGlobalCache` template
  vars, omit an empty `volumes:` section, put project services on `ddev_default` only when the
  global cache is bind-mounted.
- `pkg/ddevapp/db.go` — read the db version by exec'ing into the running db container instead
  of mounting its volume a second time.

## Design options for the shared-volume problem

`ddev-global-cache` conflates three kinds of data with different sharing needs, and splitting
it is what makes every other option tractable:

| Content | Real requirement |
|---|---|
| `mkcert/`, `global-commands/` | read-only, already on the host under `~/.ddev` |
| `traefik/` config + certs | router reads, DDEV writes |
| npm/yarn/corepack/composer caches | write-heavy, value comes from sharing across projects |
| `bashhistory`, `mysqlhistory` | tiny, per-container |

1. **Push it upstream to socktainer — highest leverage.** socktainer already bind-mounts host
   directories over virtiofs, and those are shareable. Backing `docker volume create` with a
   host directory (even opt-in, `-o backend=virtiofs`) fixes DDEV with no DDEV changes at all,
   and fixes Compose stacks generally.
2. **Bind-mount the read-only content instead of copying it in** — deletes two
   `CopyIntoVolume` calls and is faster on every provider.
3. **Give the router its own volume for `traefik/`** — it is the only consumer, and single-file
   `docker cp` works here.
4. **For the package caches, choose per-project or per-host.** Worth measuring first: DDEV
   avoids bind mounts because Docker Desktop's gRPC-FUSE is slow, and that assumption may not
   hold for virtiofs.
5. **Seed with a transient writer, then mount read-only.** The most promising
   Apple-Container-specific direction, and unlike the bind mount in the branch today it does
   not trade away volume performance. Verified end to end: a transient writer seeds and exits,
   two `:ro` readers see it, re-seeding while readers are up is blocked (500), and re-seeding
   after they are gone works. The blocked case matches DDEV's lifecycle, since project
   containers are recreated at that point anyway. What decides the split is that anything
   written *continuously* by a running container — the caches, `n_prefix`, `corepack`,
   histories — cannot live in the read-only volume. `ddev-webserver`'s `start.sh` would also
   need to stop `mkdir -p`/`chown -R`-ing under a read-only mount.
6. **A cache service container, serving a protocol rather than a filesystem.** A long-lived
   writer holding the volume while others mount `:ro` does **not** work — `:ro` readers coexist
   only with each other. It does work if the owner exposes the cache over the network: one
   caching forward proxy behind `HTTP_PROXY`/`HTTPS_PROXY` covers npm, Composer, apt and pip at
   once, and DDEV already installs a trusted mkcert root CA, which is what HTTPS interception
   needs. It only covers downloads, so install prefixes and writable state still need a real
   filesystem.

Options 5 and 6 compose: seed-then-`:ro` for read-only content, a proxy for download caches,
small per-project volumes for writable state. The ssh-agent socket directory is the one case
not to bind-mount — unix sockets over virtiofs are unreliable.

## Appendix: per-project cache volumes, considered and shelved (2026-09-16)

Raised as "give each project its own `ddev-<project>-cache` volume instead of one shared
`ddev-global-cache`, which would fix the Apple Container problem outright." Traced through the
actual mount graph before writing this off, since it's a reasonable-sounding shortcut past the
whole design-options table above.

**It doesn't fix the blocker it targets.** The Apple Container constraint is per-volume — one
RW attacher, any number of RO attachers, ever — not "shared across projects." `db`, `web`, and
the single global `ddev-router` all mount `ddev-global-cache` `:rw` simultaneously
(`app_compose_template.yaml:36,162`, `router_compose_template.yaml:46`), so the conflicting-RW
problem already exists **inside one project**, before a second project is in the picture.
Renaming the volume per-project doesn't change how many containers attach it RW at once; `web`
and `db` in the same project still fight over it, and the one router serving every project still
needs a way into each one.

Closing it for real would mean splitting per **writer**, not per project — a
`ddev-<project>-web-cache` (web's own npm/yarn/composer/corepack/bashhistory, sole writer) and a
`ddev-<project>-db-cache` (db's own mysqlhistory, sole writer), plus solving the router's RW need
separately (options 3/5 above: the router's own traefik-config path, host-bind-mounted or
seed-then-`:ro`, so the router itself never needs `:rw`). That's real design and template work
across every provider, not just this one, since compose templates aren't provider-conditional
today. It also gives up the thing the current single volume is *for*: a fresh project reusing
every other project's already-downloaded npm/yarn/composer cache instead of cold-starting its
own. That's a cost paid on Docker Desktop, Podman, and OrbStack too, for a benefit that only
matters on Apple Container.

**Not worth it for what this is.** This whole file documents an exploration gated on socktainer's
maturity, not a DDEV feature in flight — see "Where this stands" above. Restructuring how every
project caches dependencies, on every provider, to accommodate one experimental provider that
isn't close to shippable is the wrong trade. The bind-mount workaround already in place
(`UseBindGlobalCache()`) costs one conditional and stays out of the way of every other provider;
a cache-volume redesign would not. Left here rather than in the design-options table above
because it was proposed and rejected as a shortcut, not folded into the options being weighed.

## Resolved

Each of these was a blocker at some point; none needs re-investigating.

- **DNS returned unusable AAAA answers**, so traefik 502'd — the reply echoed the query's
  EDNS0 OPT record and appended the answer after it, which strict parsers read as no data.
  `getent` worked throughout because it does not send EDNS0. Fixed in
  `fix/dns-edns0-truncation`.
- **`ddev start` never returned**, hanging at "Getting traefik error output" — socktainer
  upgraded an exec only when stdin was attached, and answered a chunked body on a keep-alive
  connection to a client that had already hijacked the socket. Fixed in
  `fix/exec-hijack-close`, which also closes the channel on output EOF. Bounding the channel
  on a timer instead (the approach in the closed upstream
  [#347](https://github.com/socktainer/socktainer/pull/347)) breaks every image build, since
  buildx's `buildctl dial-stdio` legitimately outlives any bound.
- **Healthchecks never reported with DDEV's timings** — the health loop slept out the whole
  `start_period` before its first probe, so the first result always landed just after DDEV's
  identical wait expired. `ddev start` went from a 5m18s failure to 28s. Docker probes during
  `start_period` and merely declines to count failures. The healthcheck overrides that were
  tried made it worse: `timeout: 10s` undercuts the deliberate 59s sleep in `healthcheck.sh`
  and turns every later probe into a failure. Do not override.
- **Health probes ran as root**, so `ddev restart` could not remove a root-owned `/tmp/healthy`
  from a container running as 501 — `/tmp` is sticky, so file mode is irrelevant. Fixed in
  `fix/healthcheck-timing`.
- **DDEV's minimum-version check rejected socktainer** — `/version` reported `ApiVersion` as
  `"v1.51"`, a human-readable build label reused as the wire value, and the leading "v" alone
  fails `GreaterThanOrEqualTo("v1.51", "1.44")`. Fixed in `fix/version-engine-version`; no
  DDEV-side change was needed.
- **No second project could start** — `HostConfig.PortBindings` was nil on inspect, so
  `GetBoundHostPorts()` could not tell the running router's own ports from a foreign conflict.
  Fixed in `fix/port-bindings-inspect` and, on the DDEV side, in
  [#8657](https://github.com/ddev/ddev/pull/8657).
- **Two DDEV paths mounted the db volume a second time**, fatal against a running db
  container: `getDBVersionFromVolume()` and `start-chown`. Both fixed on the branch and in
  [#8658](https://github.com/ddev/ddev/pull/8658).
- **Published ports were accepted but reset.** Two distinct causes, same symptom. First, macOS
  Local Network authorization is keyed on binary *path*, and only the signed installer's path
  was authorized — hence "use the signed installer, not Homebrew"
  ([apple/container#2067](https://github.com/apple/container/issues/2067)). Second, on macOS 27
  the permission for `container-runtime-linux` never surfaced
  ([apple/container#2029](https://github.com/apple/container/issues/2029)); a previous session
  had granted it by hand for one CIDR, which made `default`-network tests pass while every
  DDEV project subnet reset. Both resolved: #2029 is fixed as of macOS 27 build 26A5421a, and
  `container-runtime-linux` now appears and is enabled in System Settings → Privacy & Security
  → Local Network.
- **Traefik stopped seeing config changes** once the global cache became a bind mount, so a
  second project 404'd. Two causes, both fixed on the branch: `copyIntoGlobalCache()` recreated
  the watched `traefik/config` directory on every push, leaving Traefik watching a dead inode;
  and a write from the host, or from the throwaway container the stop path uses, raises no
  inotify event inside the router. Routes now come and go with the router left running.
- **Container names with underscores skipped for DNS registration** — investigated against a
  live daemon and could not reproduce ([rfay/socktainer#14](https://github.com/rfay/socktainer/issues/14),
  closed).
- **`PUT`/`GET`/`HEAD /containers/{id}/archive` 404'd on a created-but-not-started
  container**, blocking buildx's bootstrap and `docker cp` into a fresh container. Was carried
  as `fix/archive-404-prestart` (clone the rootfs bundle early via a local reproduction of the
  runtime's own bundle-materialization step). Upstream fixed it independently, with a different
  and better design merged 2026-08-30/31: GET/HEAD resolve the path straight out of
  `runtime-configuration.json`'s `containerRootFilesystem.source` (the shared image snapshot)
  when `rootfs.ext4` doesn't exist yet
  ([socktainer#372](https://github.com/socktainer/socktainer/pull/372), fixing
  [socktainer#370](https://github.com/socktainer/socktainer/issues/370)); PUT stages the archive
  and injects it as a bind mount at container start when `startedDate == nil`
  ([socktainer#379](https://github.com/socktainer/socktainer/pull/379), fixing
  [socktainer#363](https://github.com/socktainer/socktainer/issues/363)). Verified by reading
  current `upstream/main` directly — no bundle-clone step remains, no gap on HEAD. Dropped from
  the fork; [rfay/socktainer#10](https://github.com/rfay/socktainer/issues/10) closed
  2026-09-16 with this explanation. Not yet in a tagged release (still v1.2.1).

## Current machine state (2026-09-16)

- Apple `container` **1.4.1** from the signed installer at `/usr/local`, upgraded this session
  (was 1.3.0) via the notarized `container-1.4.1-installer-signed.pkg` from the
  [apple/container 1.4.1 release](https://github.com/apple/container/releases/tag/1.4.1).
  Homebrew's `container` is also 1.4.1, coincidentally matching right now — that's luck, not a
  guarantee, and the two can still drift apart; see the Homebrew guidance under "Teardown and
  clean rebuild." `container-apiserver` running.
- socktainer running by hand from `~/workspace/socktainer/.build/release/socktainer`, built
  from `tmp/combined-verify-4` (replaces `tmp/combined-verify-3`), no flags. Its own
  compatibility check warns `detected 1.4.1 but this socktainer binary requires 1.2.2` — the
  pin in `AppleContainerVersionCheck` hasn't moved to track 1.4.1, but it's a printed warning,
  not fatal, and everything tested worked regardless. socktainer now auto-creates the
  `socktainer` docker context itself on startup (`docker context create` is no longer a manual
  step; harmless if run anyway — it'll just report the context already exists). The branch is
  committed but **not pushed** — see the push checklist below.
- `brew services start socktainer` is not a usable substitute: the service runs with
  `HOME=/opt/homebrew/var/run/socktainer`, so it binds a different socket and leaves the
  `socktainer` docker context talking to a dead one. Run it by hand. Homebrew's own
  `socktainer` formula (not the `socktainer/homebrew-tap`) is now a much stronger reason to
  avoid Homebrew here at all — see the `depends_on "container"` discussion above.
- Docker context is `socktainer`. Switch away only after `ddev poweroff`.
- Verified end to end this session: `ddev start` on `appletest` (fresh images, cold buildkit —
  1m30s) and `appletest2` (warm buildkit — 31s) both succeeded, both served HTTP 200
  simultaneously through the shared router, `ddev exec` and `ddev mysql` both worked.
  `appletest3` (`~/tmp/appletest3`) left stopped, not re-tested. All three need
  `omit_containers: [ddev-ssh-agent]`.
- **`buildx_buildkit_default` had to be pre-created by hand this time** — letting buildx
  bootstrap it itself now fails (see "buildx's buildkit node" and
  [rfay/socktainer#17](https://github.com/rfay/socktainer/issues/17)), a regression from
  dropping `fix/archive-404-prestart` in favor of upstream's own, buggier replacement.
- `keepalive` and the root `dnsmasq` on `192.168.64.1:53` are up per the cold-start recipe. Stop
  dnsmasq with `sudo pkill -f "listen-address=192.168.64.1"`.
- `traefik_monitor_port` is **11999** in `~/.ddev/global_config.yaml` (10999 is OrbStack's).
  Revert when done here if that matters elsewhere.
- The global `~/.ddev/router-compose.healthcheck.yaml` override remains **removed** (backup at
  `~/tmp/router-compose.healthcheck.yaml.bak`); do not restore it.
- `make staticrequired` passes on the DDEV branch (only `HANDOFF.md` changed this session; no Go
  code touched). No Go tests have been run under socktainer — the suite assumes a Docker
  provider. socktainer's own `swift test` cannot run in this environment: the tests compile, but
  loading `Testing.framework` needs full Xcode.

### Push checklist (not run automatically — see the repo's own push policy)

- `rfay/socktainer`: `git push rfay tmp/combined-verify-4` — the new integration branch replacing
  `tmp/combined-verify-3` (still on the remote, safe to leave or delete later).
- `rfay/ddev` (or wherever this branch's remote points): this session's `HANDOFF.md` commit(s) on
  `20260802_rfay_apple_container_experiment`.
