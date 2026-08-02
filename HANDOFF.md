# DDEV on Apple Container + socktainer — investigation notes (2026-08-02)

Working notes for [#7372](https://github.com/ddev/ddev/issues/7372). This documents an
exploratory session, not a finished feature: the code changes here are experiments behind an
environment variable, and this file is expected to be deleted before any of it merges.

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

Bind mounts (virtiofs) are shareable read-write. `docker volume create --driver local
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

### 4. Published ports are accepted but reset

`docker run -d -p 127.0.0.1:38080:80 nginx:alpine` → the `container` helper listens on
127.0.0.1:38080, but connections get `Recv failure: Connection reset by peer`. The same
nginx answers 200 on its container IP directly from the host. Same for DDEV's router ports.

Silver lining: Apple Container makes container IPs routable from the host, so a DDEV
"Apple Container mode" could skip port publishing entirely and point hostnames at the
router's container IP.

### 5. Healthchecks never report with DDEV's timings

DDEV uses `interval 1s / timeout 70s / start_period 120s`. With those, socktainer reports
`starting` forever and `State.Health.Log` stays empty (verified past 4 minutes).
`--health-start-period 15s` works fine (`healthy`, log populated). Shortening to
`interval 2s / timeout 10s / start_period 10s` via a compose override made all three
DDEV containers healthy in ~10s. Health also **regressed from `healthy` back to `starting`**
on long-running containers, so the status is not stable.

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

### 7. Copying a directory into a container is unsupported

`docker cp <dir>/. c:/path` → `Error response from daemon: Something went wrong.`
`docker cp <dir> c:/path/` → `cannot copy directory`. Single files work.
DDEV's `CopyIntoVolume` (mkcert CA, global commands, traefik config) is exactly this, and
it *hangs* rather than erroring when driven through the API.

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
resolves publicly to 127.0.0.1, so no hostname edit is needed. **Cause not identified.**

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
- Only one network per container: `docker network connect` is a documented no-op, so DDEV's
  web container (`["default", "ddev_default"]`) actually lands on one of them.
- Engine version is reported as `v1.51` (the API version), so DDEV warns
  "installed Docker version v1.51 is not supported, please update to version 25.0 or newer".
- `chown -R` on a virtiofs bind mount fails with `Operation not permitted`; DDEV's
  `start.sh` runs `sudo chown -R … /mnt/ddev-global-cache/` under `set -e`, which kills the
  web container.
- Container names with underscores are silently skipped for DNS registration.

---

## Experimental DDEV changes (branch `20260802_rfay_apple_container_experiment`)

All gated on `DDEV_BIND_GLOBAL_CACHE=true`; default behavior is unchanged.

- `pkg/globalconfig/global_config.go` — `UseBindGlobalCache()`, `GlobalCacheSource()`,
  `GlobalCacheMount()`. In bind mode the global cache is `~/.ddev/global-cache-bind`
  on the host instead of the `ddev-global-cache` volume.
- `pkg/ddevapp/ddevapp.go` — use the helper for all `/mnt/ddev-global-cache` mounts; skip
  creating the volume and `MkdirAll` the host dir instead; new `copyIntoGlobalCache()` that
  copies on the host in bind mode (avoids the directory-copy gap).
- `pkg/ddevapp/commands.go`, `pkg/ddevapp/traefik.go` — route their copies through
  `copyIntoGlobalCache()`.
- `pkg/ddevapp/app_compose_template.yaml`, `router_compose_template.yaml`,
  `config.go`, `router.go` — `GlobalCacheMount` / `BindGlobalCache` template vars; omit the
  `volumes:` section when it would be empty; put project services on `ddev_default` only
  in bind mode.

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
   is faster everywhere and removes one concurrent-attach case.
6. **Make the global-cache `chown` non-fatal in `start.sh`.** On virtiofs it always fails and
   `set -e` kills the web container. It is a no-op whenever ownership already matches, which
   it does, since DDEV runs as the host uid. Belongs in
   `containers/ddev-webserver/ddev-webserver-base-scripts/start.sh`.

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
promising Apple-Container-specific direction, and unlike the bind-mount approach currently
implemented behind `DDEV_BIND_GLOBAL_CACHE`, it does not trade away volume performance.
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
