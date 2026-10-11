# DDEV on Apple Container + socktainer — investigation notes

Working notes for [#7372](https://github.com/ddev/ddev/issues/7372), started 2026-08-02 and
last verified 2026-10-11. This documents an exploration, not a feature: the DDEV changes here
only take effect when the Docker provider is socktainer, and this file is expected to be
deleted before any of it merges.

Environment as last verified: Apple `container` 1.5.0 (signed installer, **not** Homebrew — see
below), `socktainer` built from
[`tmp/combined-verify-5`](https://github.com/rfay/socktainer/tree/tmp/combined-verify-5) @
2a599ee (upstream `main` @ 9dd88f7 plus five fixes, listed below), docker CLI 29.9.0, docker
context `socktainer`, macOS 27.0.1 (26A434), `ddev` from branch
[`20260802_rfay_apple_container_experiment`](https://github.com/rfay/ddev/tree/20260802_rfay_apple_container_experiment).

## Resume here

Work is paused partway through taking the fork's socktainer fixes upstream, **one issue and
PR at a time**. socktainer's maintainer has invited issues and PRs, including AI-generated
ones. Each upstream issue, PR and comment opens with this disclaimer:

> (This is AI-generated, working through issues with DDEV socktainer support in
> ddev/ddev#8656 - thanks for your generosity in allowing it)

**Before filing anything, search upstream's open *and* closed issues and PRs by topic, not by
title or author.** The exec fix was filed as #442/#443 two days after someone else's #438/#439
covered the same bug; ours were closed as duplicates. For example:
`gh search issues --repo socktainer/socktainer "<topic words>"` and
`gh pr list -R socktainer/socktainer --state all --search "<topic words>"`.

Order of remaining upstream work:

1. **Watch [socktainer#439](https://github.com/socktainer/socktainer/pull/439)** (exec fix,
   someone else's; our test report is posted on it) and DDEV's
   [#8928](https://github.com/ddev/ddev/pull/8928). Nothing to do unless review asks.
2. **Watch [socktainer#444](https://github.com/socktainer/socktainer/issues/444) /
   [#445](https://github.com/socktainer/socktainer/pull/445)** (DNS answers a multi-homed host
   with another network's address; ours, filed 2026-10-11). Branch
   `fix/dns-multihomed-network-scoped` @ 6ed8ec6, pushed. The issue's Compose repro is in
   `~/tmp/dnsrepro/` (`repro.sh`); it fails on stock 9dd88f7 every run, passes on cv5 and on
   Docker Engine. Fork issue [rfay/socktainer#7](https://github.com/rfay/socktainer/issues/7)
   points at both. Drafts as filed: `~/tmp/sk-upstream/dns-issue.md`, `dns-pr.md`.
3. **Health probes run as root** — next to file. `fix/healthcheck-probe-user` (c054611). Same
   steps as the DNS item: search upstream by topic; confirm a test fails without the fix (if the
   new tests need new API, add one that uses only existing API, as #445 did); write a
   DDEV-independent repro (e.g. an image with `USER 1000` and a healthcheck that writes a file,
   then `ls -l` it); live-test on stock and on cv5; issue with a step-by-step description of what
   happens, then PR. Fork issue [rfay/socktainer#12](https://github.com/rfay/socktainer/issues/12).
4. **A stalled health probe freezes the loop** — `fix/healthcheck-probe-timeout` (2892512),
   independent of 3.
5. **`HostConfig.PortBindings` nil on inspect** — `fix/port-bindings-inspect` (42b6e46). No fork
   issue exists; DDEV no longer depends on it (#8657).
6. **`Server.Version` reports the API version** — `fix/version-engine-version` (395d2f0). Hold:
   it overrides a deliberate testcontainers-java workaround in upstream's `VersionRoute.swift`,
   so ask upstream first, as a question, not a PR.
7. **Hostname uniqueness** — no fix exists; file as an issue with the proposal in "Open
   blockers" below. Upstream's closed #385 (Compose recreate and rename) is related background.

Per-item checklist, from socktainer's own `AGENTS.md`: Conventional Commits title; DCO
`Signed-off-by` on every commit (`git commit --signoff`); `make fmt` leaves no diff; tests pass;
a bug fix carries a unit test that fails without the fix; PR body follows
`.github/PULL_REQUEST_TEMPLATE.md`; no labels unless asked. Commits here are also GPG-signed
through 1Password, which needs the user present to approve — an agent's `git commit` fails
with "1Password: failed to fill whole buffer" otherwise. Never bypass signing.

## Where this stands

**It works end to end on upstream socktainer plus five fixes, down from nine.** Two DDEV projects
run side by side through one router — image builds, database, published ports, `ddev
exec`/`mysql`/`snapshot`, `ddev restart`, a second project's `ddev start`, `ddev poweroff` —
with no dnsmasq, no keepalive container and no hand-made buildkit node.

**On stock upstream socktainer, `ddev start` no longer has to hang.** The exec hang is fixed on
both sides, neither merged yet: upstream by [socktainer#439](https://github.com/socktainer/socktainer/pull/439),
and in DDEV by [#8928](https://github.com/ddev/ddev/pull/8928), which half-closes the exec
stream as the Docker CLI does. With only #8928, stock socktainer starts a first project, but the
router can still 504 (DNS) and a second project or `ddev restart` fails (healthcheck as root).

**What still fails with all fixes applied:** anything that makes Compose recreate a container
while the old one exists — `ddev snapshot restore`, and `ddev start` over a running project
whose config changed. See "Hostname uniqueness" below. `ddev restart` is the workaround for
the second; there is none for the first.

Merged DDEV-side hardenings from this work: [#8657](https://github.com/ddev/ddev/pull/8657)
(bound host ports fallback), [#8658](https://github.com/ddev/ddev/pull/8658)
(`GetExistingDBType` via exec), [#8659](https://github.com/ddev/ddev/pull/8659) (non-fatal
global-cache chown). Open: [#8928](https://github.com/ddev/ddev/pull/8928) (exec half-close,
branch `20261011_rfay_exec_closewrite`, worktree
`~/workspace/ddev-worktrees/20261011_rfay_exec_closewrite`). Draft:
[#8656](https://github.com/ddev/ddev/pull/8656), this branch.

### What still has to be done by hand

- `omit_containers: [ddev-ssh-agent]` in each project — see "ssh-agent" below.
- `router_http_port`/`router_https_port` above 1024 (8080/8443 here).
- A `traefik_monitor_port` other than OrbStack's 10999 (11999 here), when another provider's
  DDEV router might also be running.
- `ddev restart` rather than `ddev start` over a running project with changed config.

Mutagen needs no per-project setting: it is forced off automatically on this provider.

## socktainer fixes in `tmp/combined-verify-5`

Each is one commit on upstream `main` @ 9dd88f7, signed and DCO signed-off.
`tmp/combined-verify-5` @ 2a599ee merges all of them; it passes socktainer's full test suite
(1120 tests) and `make fmt`.

| Fix | Branch / source | Issues | Needed for DDEV |
|---|---|---|---|
| Exec without stdin never ends unless the client half-closes | upstream PR [#439](https://github.com/socktainer/socktainer/pull/439) (c338b88, fetched as local `pr-439`) | upstream [#438](https://github.com/socktainer/socktainer/issues/438); fork [#8](https://github.com/rfay/socktainer/issues/8) | `ddev start` at all, unless DDEV has #8928 |
| DNS answers a multi-homed host with another network's address | upstream PR [#445](https://github.com/socktainer/socktainer/pull/445), [`fix/dns-multihomed-network-scoped`](https://github.com/rfay/socktainer/tree/fix/dns-multihomed-network-scoped) @ 6ed8ec6 (cv5 has b355bf9, same code, one test fewer) | upstream [#444](https://github.com/socktainer/socktainer/issues/444); fork [#7](https://github.com/rfay/socktainer/issues/7) | a second project, and sometimes the first (504) |
| Health probes run as root, so `/tmp/healthy` is root-owned | [`fix/healthcheck-probe-user`](https://github.com/rfay/socktainer/tree/fix/healthcheck-probe-user) | fork [#12](https://github.com/rfay/socktainer/issues/12) | a second project, `ddev restart` |
| A stalled probe freezes the health loop | [`fix/healthcheck-probe-timeout`](https://github.com/rfay/socktainer/tree/fix/healthcheck-probe-timeout) | fork [#12](https://github.com/rfay/socktainer/issues/12) | latent |
| `HostConfig.PortBindings` always nil on inspect | [`fix/port-bindings-inspect`](https://github.com/rfay/socktainer/tree/fix/port-bindings-inspect) | none | no; DDEV's #8657 covers it |
| `Server.Version` reports the API version | [`fix/version-engine-version`](https://github.com/rfay/socktainer/tree/fix/version-engine-version) | fork [#13](https://github.com/rfay/socktainer/issues/13) | cosmetic |

`tmp/combined-verify-5` merges `fix/healthcheck-probe-user-and-timeout`, which holds both
healthcheck commits.

The DNS and healthcheck branches were re-ported from scratch rather than rebased, because
upstream had rewritten the code they touch. The DNS port also fixes a bug in the original fork
commit, which left stale entries on a multi-homed container's other networks after stop or
delete.

### The exec fix, for the record

socktainer sent a stdout/stderr-only exec that asked for `Upgrade: tcp` down its HTTP streaming
fallback, because the hijack decision required `attachStdin`. That fallback also answers `101`,
but after the output ends Vapor keeps the connection alive, so a client reading to EOF waits
until it half-closes. The Docker CLI always half-closes after attaching
([`hijack.go`](https://github.com/docker/cli/blob/v29.0.0/cli/command/container/hijack.go#L159-L189)),
so `docker exec` worked; Go clients such as DDEV's `dockerutil.Exec` hung. A `socat` capture
showed the CLI's and DDEV's requests and responses byte-identical apart from that half-close.

Our own fix, PR #443 (branch `fix/exec-hijack-stdout-only`, 087134f), was closed as a duplicate
of #439. Both builds passed the same suite, including 15 MB of output arriving intact; #439 is
about twice as fast and has fuller tests. The older fork branch `fix/exec-hijack-close` is
superseded.

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

A volume is either held read-write by exactly one container, or read-only by any number:

| Held by | New `:ro` attach | New rw attach |
|---|---|---|
| nothing | works | works |
| one rw container | **fails** | **fails** |
| one or more `:ro` containers | **works** (tested with 3) | **fails** |

Host bind mounts (virtiofs) are shareable read-write. `docker volume create --driver local -o
type=none -o device=… -o o=bind` is ignored by socktainer — it still makes a block volume.

Where it hits DDEV: `ddev-global-cache` (web, db, router and every `RunSimpleContainer`
helper), `ddev-ssh-agent_socket_dir`, and Mutagen, which mounts `project_mutagen` twice inside
the web container alone. The branch bind-mounts the global cache instead; see "DDEV changes on
the branch".

[lunguini/gocker](https://github.com/lunguini/gocker) dodges this with a `shared` mode that runs
every container inside one VM — Docker Desktop's architecture rebuilt from scratch, giving up
the per-container VMs that are the point of Apple Container.

### ssh-agent cannot work as designed

Not a volume-mode problem. Web would only need the socket volume read-only, but `:ro` still
can't attach while ssh-agent holds it read-write. And even if it could, a unix socket belongs
to the kernel that created it: every container is its own VM, so web could see the socket file
but never connect to the agent in another VM. A virtiofs bind mount has the same limit. It
needs a different transport — the agent listening on TCP on `ddev_default` with `socat` in web
bridging `/home/.ssh-agent/socket`, or the agent running inside web — which changes DDEV's
ssh-agent design on every provider. Deferred until the blockers above are settled.

### Ports below 1024 cannot be published

Apple Container's port forwarder runs unprivileged, so use `ddev config
--router-http-port=8080 --router-https-port=8443`. Same restriction as rootless Podman, except
that macOS has no `ip_unprivileged_port_start` equivalent.
[apple/container#1985](https://github.com/apple/container/issues/1985) narrows it to explicit
low bind addresses (`-p 127.0.0.1:80:80`); DDEV publishes without one.

### Smaller gaps

- **Hot-attach of networks is unsupported.** Multi-homing at create time works; `docker
  network connect` afterwards is a no-op, since Virtualization.framework has no NIC hotplug.
- **The healthy marker's group is 0, not the container's 20**, even with probes running as
  the configured user. Harmless to DDEV, since the owner is right.
- **A buildkit node that has exited must be recreated, never restarted** — a restarted one
  fails every build with `rbind … operation not permitted`. Not re-checked on 1.5.0.
- **Upstream's DNS forwards non-container names to the vmnet gateway** (#426), which depends
  on macOS's own resolver answering on `192.168.64.1` — see the next section.

## Resolved: no DNS on the vmnet gateway was this machine

The long-standing "no DNS on the built-in `default` network", and the dnsmasq that forwarded
`192.168.64.1` to socktainer, were an artifact of this machine. A root Homebrew dnsmasq service
(`sudo brew services`, installed 2025-09-18, not visible in a plain `brew services list`) held
port 53. While it ran, macOS's own vmnet DNS proxy never answered on `192.168.64.1`. With that
service stopped and `container system` restarted, the gateway resolves on its own, for both
`default` and the project networks.

Since upstream #426 forwards every non-container lookup to the gateway, the old dnsmasq
workaround would now loop. On a machine where something else holds port 53, named networks
lose external DNS too. The `keepalive` container is no longer needed either.

## Recipes

### Teardown and clean rebuild

**Always `ddev poweroff` before switching Docker contexts or socktainer binaries.** A router left
running keeps 8080/8443/11999 bound, and a daemon restart resets every re-adopted container's
health to `starting`.

```bash
# 1. tear everything down
ddev poweroff
pkill -f "\.build/release/socktainer$"
/usr/local/bin/container rm -f $(/usr/local/bin/container ls -aq)   # every instance
/usr/local/bin/container system stop
rm -f ~/.socktainer/container.sock

# 2. build the branch under test, in its worktree
cd ~/tmp/socktainer-cv5          # worktree of tmp/combined-verify-5
swift build -c release           # ~380s from empty .build, ~150-330s incremental

# 3. bring both systems back up
/usr/local/bin/container system start            # the signed installer explicitly
nohup ./.build/release/socktainer >> ~/tmp/socktainer.log 2>&1 &
docker context use socktainer                    # socktainer creates the context itself

# 4. confirm
/usr/local/bin/container system status | grep -E 'installRoot|server.version'
```

To test a different socktainer build, swap only the daemon: `ddev poweroff`, `pkill` as above,
start the other build's `.build/release/socktainer` with `nohup`, test, then swap back and
`ddev start` the projects again. One daemon at a time: they share the socket and DNS port.
Builds on disk: `~/tmp/socktainer-cv5` (the working stack), `~/workspace/socktainer` on
`tmp/upstream-main-test` (stock upstream 9dd88f7), `~/tmp/socktainer-pr439` (#439 alone).

socktainer's tests run here with
`swift test -Xswiftc -plugin-path -Xswiftc /Library/Developer/CommandLineTools/usr/lib/swift/host/plugins/testing`;
`make test` has no clean way to pass the flag. Also run `make fmt` and check `git status`.

Homebrew's `container` formula sits ahead of `/usr/local/bin` on `PATH`, so always call
`/usr/local/bin/container` for `system start`/`status`. Do not `brew install socktainer` from
homebrew-core here: it requires Homebrew's own `container`
([socktainer/socktainer#392](https://github.com/socktainer/socktainer/issues/392)), and
socktainer's auto-start bootstrap
([socktainer#374](https://github.com/socktainer/socktainer/pull/374)) runs `container system
start` with whichever binary is first on `PATH`. `brew services start socktainer` also binds a
different socket, because the service runs with a different `HOME`.

To upgrade `container`: `gh release download <ver> -R apple/container -p
'*installer-signed.pkg'`, check it with `pkgutil --check-signature`, then
`sudo installer -pkg … -target /` with the apiserver stopped.

### Test tools

Kept in `~/tmp/socktainer-tools/`:

- `execrepro-main.go` — non-interactive exec through the moby Go client, as DDEV does it.
  Prints byte counts, exit code and timing; `CLOSEWRITE=1` half-closes like the CLI. It builds
  against DDEV's vendored client, so copy it to `.gotmp/execrepro/main.go` in a DDEV checkout
  and `go run ./.gotmp/execrepro <container> '<cmd>'` with
  `DOCKER_HOST=unix://$HOME/.socktainer/container.sock`.
- `exec-suite.sh <label>` — the exec suite run against #439 and #443: small and 15 MB outputs,
  exit codes, piped stdin, a 20s exec, `-d`. Needs the repro in place as above.
- `bxtest.Dockerfile` — a trivial build for `docker buildx build --no-cache`; remove
  `buildx_buildkit_*` containers and their `_state` volumes first to test a cold bootstrap.
- `wt-status.sh` — every socktainer worktree's branch against `rfay` and `upstream/main`.

To capture raw API traffic, put a logging proxy in front of the socket:
`socat -v UNIX-LISTEN:$HOME/tmp/proxy.sock,fork UNIX-CONNECT:$HOME/.socktainer/container.sock 2> ~/tmp/socat.log`,
then point `DOCKER_HOST` at the proxy.

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

The exec half-close is deliberately **not** on this branch; it is #8928 on its own branch.
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
- socktainer running by hand from `~/tmp/socktainer-cv5/.build/release/socktainer`
  (`tmp/combined-verify-5` @ 2a599ee), built without `make`, so `docker version` shows
  `unspecified`. Log: `~/tmp/socktainer.log`.
- `appletest` and `appletest2` (`~/tmp/`) running, both serving 200 over HTTP and HTTPS;
  `appletest3` untested. All need `omit_containers: [ddev-ssh-agent]`.
- **The root Homebrew dnsmasq service is stopped** (`sudo brew services stop dnsmasq`); no
  dnsmasq is running. Restart it only if `*.example.site` resolution matters; vmnet DNS then
  needs it stopped again and a `container system` restart.
- No `keepalive` container. buildx bootstraps its own nodes.
- `traefik_monitor_port` is **11999** in `~/.ddev/global_config.yaml` (10999 is OrbStack's).
- The global `~/.ddev/router-compose.healthcheck.yaml` override stays removed.
- The built `ddev` in this worktree has no exec half-close, so it exercises socktainer's exec
  fix directly.

### Repositories and worktrees

socktainer, main checkout `~/workspace/socktainer` (remotes `upstream`, `rfay`):

| Worktree | Branch | State |
|---|---|---|
| `~/tmp/socktainer-cv5` | `tmp/combined-verify-5` @ 2a599ee | the running stack; **needs force-push** |
| `~/workspace/socktainer-worktrees/port-bindings-inspect` | `fix/port-bindings-inspect` @ 42b6e46 | amended for sign-off + fmt; **needs force-push** |
| `~/tmp/socktainer-dnsfix` | `fix/dns-multihomed-network-scoped` @ 6ed8ec6 | pushed; upstream PR #445 |
| `~/tmp/socktainer-hcfix` | `fix/healthcheck-probe-user-and-timeout` | pushed |
| `~/tmp/socktainer-hcfix-timeout` | `fix/healthcheck-probe-timeout` | pushed |
| `~/workspace/socktainer-worktrees/version-engine-version` | `fix/version-engine-version` | pushed |
| `~/tmp/socktainer-pr439` | `pr-439` (upstream PR #439) | local only, test build |
| `~/tmp/socktainer-execfix` | `fix/exec-hijack-stdout-only` | pushed; PR #443 closed, **can delete** |
| `~/workspace/socktainer-worktrees/exec-hijack-close` | `fix/exec-hijack-close` | superseded by #439, **can delete** |
| `~/workspace/socktainer` | `tmp/upstream-main-test` | stock upstream build |

`fix/healthcheck-probe-user` has no worktree. Local-only backups:
`backup/combined-verify-5-fork-exec` (the previous cv5, with the fork's exec fix) and
`backup/version-v-prefix-strip`.

DDEV worktrees: this branch is in
`~/workspace/ddev-worktrees/20260802_rfay_apple_container_experiment`, and the branch for PR
8928 is in `~/workspace/ddev-worktrees/20261011_rfay_exec_closewrite`.

### Push checklist (not run automatically — see the repo's push policy)

```bash
cd ~/workspace/socktainer
git push --force-with-lease rfay tmp/combined-verify-5 fix/port-bindings-inspect
git -C ~/workspace/ddev-worktrees/20260802_rfay_apple_container_experiment push rfay 20260802_rfay_apple_container_experiment
```
