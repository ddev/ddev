# DDEV on Apple Container + socktainer — investigation notes

Working notes for [#7372](https://github.com/ddev/ddev/issues/7372), started 2026-08-02 and
last verified 2026-10-11. This documents an exploration, not a feature: the DDEV changes here
only take effect when the Docker provider is socktainer, and this file is expected to be
deleted before any of it merges.

Environment as last verified: Apple `container` 1.5.0 (signed installer, **not** Homebrew — see
below), `socktainer` built from
[`tmp/combined-verify-5`](https://github.com/rfay/socktainer/tree/tmp/combined-verify-5)
(upstream `main` @ 9dd88f7, which is v1.5.2 plus four commits, plus five fork branches), docker
CLI 29.9.0, docker context `socktainer`, macOS 27.0.1 (26A434), `ddev` from branch
[`20260802_rfay_apple_container_experiment`](https://github.com/rfay/ddev/tree/20260802_rfay_apple_container_experiment).

## Where this stands

**It works end to end on five fork fixes, down from nine.** Two DDEV projects run side by side
through one router — image builds, database, published ports, `ddev exec`/`mysql`/`snapshot`,
`ddev restart`, a second project's `ddev start`, `ddev poweroff` — with no dnsmasq, no keepalive
container and no hand-made buildkit node. Every remaining fork fix is now one commit on current
upstream `main`, ready to offer as a PR.

socktainer moved fast between 2026-09-16 and 2026-10-11: five releases (v1.4.0 through v1.5.2)
and about 30 merged PRs. Upstream fixed four of the nine fork fixes on its own (EDNS0, both
filter fixes, directory copy) and the buildx bootstrap failure. The maintainer has invited
issues and PRs, including AI-generated ones.

**On stock upstream socktainer, `ddev start` still hangs** — in `GetRouterConfigErrors()` →
`dockerutil.Exec`, the non-interactive exec bug. That is the one hard blocker. After it, the
DNS and healthcheck fixes are what make a second project and `ddev restart` work.

**What still fails with all fixes applied:** anything that makes Compose recreate a container
while the old one exists — `ddev snapshot restore`, and `ddev start` over a running project
whose config changed. See "Hostname uniqueness" below. `ddev restart` is the workaround for
the second; there is none for the first.

Three DDEV-side hardenings split out of this work are merged:
[#8657](https://github.com/ddev/ddev/pull/8657) (bound host ports fallback),
[#8658](https://github.com/ddev/ddev/pull/8658) (`GetExistingDBType` via exec) and
[#8659](https://github.com/ddev/ddev/pull/8659) (non-fatal global-cache chown).

### What still has to be done by hand

- `omit_containers: [ddev-ssh-agent]` — its socket volume is shared, and unix sockets over
  virtiofs are unreliable, so bind-mounting is not an answer either.
- `router_http_port`/`router_https_port` above 1024, distinct per project.
- A `traefik_monitor_port` that does not collide with another provider's router.
- `ddev restart` rather than `ddev start` over a running project with changed config.

Mutagen needs no per-project setting: it is forced off automatically on this provider.

## socktainer fixes carried in the fork

Each is one commit on upstream `main` @ 9dd88f7, merged together on
[`tmp/combined-verify-5`](https://github.com/rfay/socktainer/tree/tmp/combined-verify-5). That
branch passes socktainer's full test suite (1105 tests) and `make fmt`.

| Fix | Branch | Issue | Needed for |
|---|---|---|---|
| Exec upgraded only when stdin is attached; channel not closed on output EOF | [`fix/exec-hijack-close`](https://github.com/rfay/socktainer/tree/fix/exec-hijack-close) | [#8](https://github.com/rfay/socktainer/issues/8), upstream [socktainer#346](https://github.com/socktainer/socktainer/issues/346) | `ddev start` at all |
| DNS answers a multi-homed host with another network's address | [`fix/dns-multihomed-network-scoped`](https://github.com/rfay/socktainer/tree/fix/dns-multihomed-network-scoped) | [#7](https://github.com/rfay/socktainer/issues/7) | a second project (504 otherwise) |
| Health probes run as root, so `/tmp/healthy` is root-owned | [`fix/healthcheck-probe-user`](https://github.com/rfay/socktainer/tree/fix/healthcheck-probe-user) | [#12](https://github.com/rfay/socktainer/issues/12) | a second project, `ddev restart` |
| A stalled probe freezes the health loop | [`fix/healthcheck-probe-timeout`](https://github.com/rfay/socktainer/tree/fix/healthcheck-probe-timeout) | [#12](https://github.com/rfay/socktainer/issues/12) | latent |
| `HostConfig.PortBindings` always nil on inspect | [`fix/port-bindings-inspect`](https://github.com/rfay/socktainer/tree/fix/port-bindings-inspect) | none filed | correctness; DDEV's #8657 covers it |
| `Server.Version` reports the API version | [`fix/version-engine-version`](https://github.com/rfay/socktainer/tree/fix/version-engine-version) | [#13](https://github.com/rfay/socktainer/issues/13) | cosmetic now that #436 dropped the "v" |

`tmp/combined-verify-5` merges `fix/healthcheck-probe-user-and-timeout`, which holds both
healthcheck commits. `fix/version-engine-version` deliberately overrides a testcontainers-java
workaround in upstream's `VersionRoute.swift`, so it needs that reason answered before it goes
upstream.

The DNS and healthcheck branches were re-ported from scratch, not rebased, because upstream
had rewritten the code they touch. The DNS port also fixes a bug in the original fork commit,
which left stale entries on a multi-homed container's other networks after stop or delete.

socktainer's tests do run here, contrary to earlier notes: add
`-Xswiftc -plugin-path -Xswiftc /Library/Developer/CommandLineTools/usr/lib/swift/host/plugins/testing`
to `swift build --build-tests` or `swift test`.

### Fixed upstream since 2026-09-16, dropped from the fork

| Was | Fixed by | Closed |
|---|---|---|
| EDNS0 OPT record echoed in DNS replies | [socktainer#348](https://github.com/socktainer/socktainer/pull/348) | [#15](https://github.com/rfay/socktainer/issues/15) |
| `--filter name=` ignored; dict-form filters dropped | [socktainer#404](https://github.com/socktainer/socktainer/pull/404), regex matching like moby | [#9](https://github.com/rfay/socktainer/issues/9) |
| buildx bootstrap PUT into a never-started container fails | [socktainer#395](https://github.com/socktainer/socktainer/pull/395), [#405](https://github.com/socktainer/socktainer/pull/405) | [#17](https://github.com/rfay/socktainer/issues/17) |
| Archive GET aborts on a dangling symlink; raw ext4 stat mode | [socktainer#372](https://github.com/socktainer/socktainer/pull/372), [#376](https://github.com/socktainer/socktainer/pull/376) | [#16](https://github.com/rfay/socktainer/issues/16) |
| `docker cp` of a directory hangs | verified gone on stock upstream | [#11](https://github.com/rfay/socktainer/issues/11) |
| `ApiVersion` "v" prefix failing DDEV's version check | [socktainer#436](https://github.com/socktainer/socktainer/pull/436) | half of [#13](https://github.com/rfay/socktainer/issues/13) |
| Archive 404 on created-but-not-started containers | [socktainer#372](https://github.com/socktainer/socktainer/pull/372), [#379](https://github.com/socktainer/socktainer/pull/379) | [#10](https://github.com/rfay/socktainer/issues/10) |
| `--privileged` granted no capabilities | [socktainer#364](https://github.com/socktainer/socktainer/pull/364) | [#1](https://github.com/rfay/socktainer/issues/1) |

## Open blockers

### Hostname uniqueness collides with Compose's recreate — now the main gate

```text
Container ddev-appletest-web Error response from daemon: Failed to create container:
exists: "hostname(s) already exist: ["appletest-web", "appletest-web"]"
```

Compose creates the replacement container before removing the old one. Apple Container
rejects any hostname already used by **any** container, whatever its state
(`ContainerAPIService/Server/Containers/ContainersService.swift`). The name appears twice
because the container has two network attachments sharing one hostname. Upstream's rename
support ([socktainer#418](https://github.com/socktainer/socktainer/pull/418)) covers only
never-started containers and doesn't help, because the create itself fails.

Still present on container 1.5.0 and upstream `main`. It breaks `ddev snapshot restore`
outright: restore removes the db container and runs `compose up`, which recreates web as a
dependent while the old web still exists. `ddev restart` is unaffected, because `Stop()` runs
`compose down` first. The running project keeps serving through the failure.

The likely socktainer-side fix: keep the requested hostname as the container's own hostname,
but stop passing it as the network attachment hostname. That name feeds Apple Container's own
DNS, which socktainer doesn't use since it runs its own. No issue filed yet.

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

**Read-only attach is not exclusive**: a volume is either held read-write by exactly one
container, or read-only by any number.

| Held by | New `:ro` attach | New rw attach |
|---|---|---|
| nothing | works | works |
| one rw container | **fails** | **fails** |
| one or more `:ro` containers | **works** (tested with 3) | **fails** |

Host bind mounts (virtiofs) are shareable read-write. `docker volume create --driver local -o
type=none -o device=… -o o=bind` is ignored by socktainer — it still makes a block volume.

Where it hits DDEV: `ddev-global-cache` (web, db, router, ssh-agent and every
`RunSimpleContainer` helper), `ddev-ssh-agent_socket_dir` (ssh-agent and web), and Mutagen,
which mounts `project_mutagen` twice inside the web container alone. The branch bind-mounts
the global cache instead; see "DDEV changes on the branch".

[lunguini/gocker](https://github.com/lunguini/gocker) dodges this with a `shared` mode that runs
every container inside one VM — Docker Desktop's architecture rebuilt from scratch, giving up
the per-container VMs that are the point of Apple Container. It confirms the constraint is the
cost of that model rather than something an architecture sidesteps for free.

### Ports below 1024 cannot be published

Apple Container's port forwarder runs unprivileged, so `ddev config
--router-http-port=8080 --router-https-port=8443`. Same restriction as rootless Podman, except
that macOS has no `ip_unprivileged_port_start` equivalent.
[apple/container#1985](https://github.com/apple/container/issues/1985) narrows it to explicit
low bind addresses (`-p 127.0.0.1:80:80`); DDEV publishes without one, so nothing changes here
yet.

### Smaller gaps

- **Hot-attach of networks is unsupported.** Multi-homing at create time works; `docker
  network connect` afterwards is a no-op, since Virtualization.framework has no NIC hotplug.
- **The healthy marker's group is 0, not the container's 20**, even with probes running as
  the configured user. Harmless to DDEV, since the owner is right.
- **A buildkit node that has exited must be recreated, never restarted** — a restarted one
  fails every build with `rbind … operation not permitted`. Not re-checked on 1.5.0.
- **Upstream's DNS forwards non-container names to the vmnet gateway** (#426). That depends
  on macOS's own resolver answering on `192.168.64.1`; see the next section.

## Resolved this round: no DNS on the vmnet gateway was this machine

The long-standing "no DNS on the built-in `default` network", and the dnsmasq that forwarded
`192.168.64.1` to socktainer, were an artifact of this machine. A root Homebrew dnsmasq service
(`sudo brew services`, installed 2025-09-18, not visible in a plain `brew services list`) held
port 53. While it ran, macOS's own vmnet DNS proxy never answered on `192.168.64.1`. With that
service stopped and `container system` restarted, the gateway resolves on its own, for both
`default` and the project networks. The service is now stopped.

This matters more since upstream #426, which forwards every non-container lookup to the gateway.
The old dnsmasq workaround would now loop, since it forwarded the gateway back to socktainer.
On a machine where something else holds port 53, named networks lose external DNS too.

The `keepalive` container and the dnsmasq cold-start recipe are no longer needed. DNS works on
both networks with `keepalive` removed, though not yet tested with the `default` network fully
idle, since the buildkit node runs on it.

## Recipes

### Teardown and clean rebuild

**Always `ddev poweroff` before switching Docker contexts**, in either direction. A router left
running under the previous context keeps 8080/8443/11999 bound.

```bash
# 1. tear everything down
ddev poweroff
pkill -f "\.build/release/socktainer$"
/usr/local/bin/container rm -f $(/usr/local/bin/container ls -aq)   # every instance
/usr/local/bin/container system stop
rm -f ~/.socktainer/container.sock

# 2. build the branch under test (in its worktree)
cd ~/tmp/socktainer-cv5          # worktree of tmp/combined-verify-5
swift build -c release           # ~380s from empty .build, ~150s incremental

# 3. bring both systems back up
/usr/local/bin/container system start            # the signed installer explicitly
nohup ./.build/release/socktainer >> ~/tmp/socktainer.log 2>&1 &
docker context use socktainer                    # socktainer creates the context itself

# 4. confirm
/usr/local/bin/container system status | grep -E 'installRoot|server.version'
```

Restart socktainer with `ddev poweroff` first. Health status resets to `starting` for every
container the daemon re-adopts, so anything waiting for healthy can hang.

Homebrew's `container` formula (1.5.0 too, by coincidence) sits ahead of `/usr/local/bin` on
`PATH`, so always call `/usr/local/bin/container` for `system start`/`status`. Do not
`brew install socktainer` from homebrew-core here: its maintainer requires Homebrew's own
`container`
([socktainer/socktainer#392](https://github.com/socktainer/socktainer/issues/392)), and
socktainer's auto-start bootstrap
([socktainer#374](https://github.com/socktainer/socktainer/pull/374)) runs `container system
start` with whichever binary is first on `PATH`. Starting `/usr/local/bin/container` first
makes that a no-op. `brew services start socktainer` also binds a different socket, because
the service runs with a different `HOME`.

To upgrade `container`: `gh release download <ver> -R apple/container -p
'*installer-signed.pkg'`, check it with `pkgutil --check-signature`, then
`sudo installer -pkg … -target /` with the apiserver stopped.

To reach a project at the router's container IP rather than a published port:

```bash
RIP=$(docker inspect ddev-router --format '{{range .NetworkSettings.Networks}}{{.IPAddress}} {{end}}' | awk '{print $1}')
curl -sS -k -o /dev/null -w '%{http_code}\n' --resolve appletest.ddev.site:8443:$RIP https://appletest.ddev.site:8443/
```

## DDEV changes on the branch

The global-cache bind mount turns on **automatically when the provider is socktainer**.
`DDEV_BIND_GLOBAL_CACHE` forces it either way for testing. On every other provider, and
whenever the provider cannot be reached, detection returns false and behavior is unchanged.

- `pkg/dockerutil/providers.go` — `IsSocktainer()` (matches `Server.Platform.Name`, like
  `IsPodman()`) and `IsAppleContainer()`, plus `UseBindGlobalCache()`, `GlobalCacheSource()`
  and `GlobalCacheMount()`. With the bind mount on, the global cache is
  `~/.ddev/global-cache-bind` mounted at `/mnt/ddev-global-cache`.
- `pkg/ddevapp/ddevapp.go` — route every `/mnt/ddev-global-cache` mount through the helpers;
  create the host dir instead of the volume; `copyIntoGlobalCache()` copies on the host;
  `mergeCopyDir()` copies into existing directories so Traefik's watch survives. start-chown
  skips the db volume when the db container is already running.
- `pkg/ddevapp/traefik.go` — `NotifyRouterOfTraefikConfigChange()` touches the config from
  inside the router, since host writes raise no inotify event there.
- `pkg/ddevapp/performance_mode.go`, `mutagen.go` — force Mutagen off with a `WarningOnce`;
  `no_bind_mounts` warns separately, since it depends on Mutagen.
- Compose templates, `config.go`, `router.go` — `GlobalCacheMount`/`BindGlobalCache` template
  vars, and no `ddev-global-cache` volume declaration when bind-mounted.
- `environment.go`, `version.go`, `utility-port-diagnose.go`, `addons.go` — report the
  provider as `apple-container`, and give add-on installs the virtiofs propagation delay.

Removed 2026-10-11 as dead code: an exec-based db-version read that upstream #8658 had made
unreachable, and a `networks: ["ddev_default"]` restriction on db and web that
`fixupComposeYaml` always undid.

## Design options for the shared-volume problem

`ddev-global-cache` conflates data with different sharing needs, and splitting it is what
makes the other options tractable:

| Content | Real requirement |
|---|---|
| `mkcert/`, `global-commands/` | read-only, already on the host under `~/.ddev` |
| `traefik/` config + certs | router reads, DDEV writes |
| npm/yarn/corepack/composer caches | write-heavy, value comes from sharing across projects |
| `bashhistory`, `mysqlhistory` | tiny, per-container |

1. **Push it upstream to socktainer — highest leverage.** Backing `docker volume create` with
   a host directory (even opt-in, `-o backend=virtiofs`) fixes DDEV with no DDEV changes, and
   fixes Compose stacks generally.
2. **Bind-mount the read-only content instead of copying it in** — faster on every provider.
3. **Give the router its own volume for `traefik/`** — it is the only consumer.
4. **For the package caches, choose per-project or per-host.** Measure first: DDEV avoids bind
   mounts because of Docker Desktop's gRPC-FUSE, and that may not hold for virtiofs.
5. **Seed with a transient writer, then mount read-only.** Verified end to end: a writer
   seeds and exits, two `:ro` readers see it, re-seeding is blocked while readers are up.
   Anything written continuously by a running container can't live there.
6. **A cache service container serving a protocol** — a caching proxy behind
   `HTTP_PROXY`/`HTTPS_PROXY` covers npm, Composer, apt and pip, using DDEV's mkcert CA for
   interception. Covers downloads only.

Options 5 and 6 compose. Per-project cache volumes were considered on 2026-09-16 and shelved:
the constraint is per volume, and web, db and the router all attach the cache read-write
within one project, so per-project naming doesn't help. Splitting per writer would, at a
cost paid on every provider.

## Current machine state (2026-10-11)

- Apple `container` **1.5.0** from the signed installer at `/usr/local`
  (`container-apiserver` running). Homebrew's `container` is also 1.5.0.
- socktainer running by hand from `~/tmp/socktainer-cv5/.build/release/socktainer`, built
  from `tmp/combined-verify-5` without `make`, so `docker version` shows `unspecified`.
  The previous stock-upstream build is at `~/workspace/socktainer/.build/release/socktainer`
  (branch `tmp/upstream-main-test`).
- **The root Homebrew dnsmasq service is stopped** (`sudo brew services stop dnsmasq`), and no
  dnsmasq is running. Start it again with `sudo brew services start dnsmasq` only if
  `*.example.site` resolution matters; vmnet DNS then needs it stopped and a `container system`
  restart.
- No `keepalive` container. `buildx_buildkit_default` was bootstrapped by buildx itself.
- `appletest` and `appletest2` (`~/tmp/`) running, both serving 200; `appletest3` untested.
  All three need `omit_containers: [ddev-ssh-agent]`.
- `traefik_monitor_port` is **11999** in `~/.ddev/global_config.yaml` (10999 is OrbStack's).
- The global `~/.ddev/router-compose.healthcheck.yaml` override stays removed; do not restore
  it.

### Push checklist (not run automatically — see the repo's push policy)

`rfay/socktainer`, from `~/workspace/socktainer`:

- New branches: `fix/dns-multihomed-network-scoped`, `fix/healthcheck-probe-user`,
  `fix/healthcheck-probe-timeout`, `fix/healthcheck-probe-user-and-timeout`,
  `tmp/combined-verify-5`.
- **Force-push** (rebased onto upstream `main`): `fix/exec-hijack-close`,
  `fix/version-engine-version`, `fix/port-bindings-inspect`.
- Local only, no need to push: `backup/version-v-prefix-strip` (superseded by #436),
  `tmp/upstream-main-test`.

`rfay/ddev`: this session's commits on `20260802_rfay_apple_container_experiment`.
