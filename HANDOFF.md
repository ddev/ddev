# DDEV on Apple Container + socktainer — investigation notes

Working notes for [#7372](https://github.com/ddev/ddev/issues/7372), started 2026-08-02 and
last verified 2026-10-31. This documents an exploration, not a feature: the DDEV changes here
only take effect when the Docker provider is socktainer, and this file is expected to be
deleted before any of it merges.

Environment as last verified: Apple `container` 1.6.1 (signed installer, **not** Homebrew — see
below), `socktainer` built from upstream `main` as of 2026-10-31 with no fork fixes, docker
CLI 29.9.2, docker context `socktainer`, macOS 27.0.1 (26A434), `ddev` from branch
[`20260802_rfay_apple_container_experiment`](https://github.com/rfay/ddev/tree/20260802_rfay_apple_container_experiment),
rebased on `upstream/main` as of 2026-10-31.

## Resume here

Paused 2026-10-31. **Stock upstream socktainer runs DDEV end to end** on this branch, which
is now much smaller: upstream DDEV's split of `ddev-global-cache` (for
[#8895](https://github.com/ddev/ddev/issues/8895)) removed most of what the branch had to
work around. Two new socktainer bugs turned up while bringing it back, both with DDEV-side
workarounds on the branch and DDEV-independent repros in `handoff/`, neither filed yet.

socktainer's maintainer has invited issues and PRs, including AI-generated ones, **one issue
and PR at a time**. Each upstream issue, PR and comment opens with this disclaimer:

> (This is AI-generated, working through issues with DDEV socktainer support in
> ddev/ddev#8656 - thanks for your generosity in allowing it)

**Before filing anything, search upstream's open *and* closed issues and PRs by topic, not by
title or author.** The exec fix was filed as #442/#443 two days after someone else's #438/#439
covered the same bug; ours were closed as duplicates. For example:
`gh search issues --repo socktainer/socktainer "<topic words>"` and
`gh pr list -R socktainer/socktainer --state all --search "<topic words>"`. Both items below
were searched on 2026-10-31 and found nothing.

Next to file, in this order:

1. **The image list ignores the `label` filter** — filed 2026-10-31 as
   [socktainer#491](https://github.com/socktainer/socktainer/issues/491), fix in PR
   [#492](https://github.com/socktainer/socktainer/pull/492) (branch
   `fix/image-list-label-filter`), waiting on review. `before`, `since` and `until` look
   ignored the same way, untested and left out of #492. `GET /images/json`
   applies only `dangling` and `reference` (`ImageListRoute.applyFilters`), so
   `docker images --filter label=x=y` lists every image. Repro
   `handoff/imgfilterrepro/repro.sh`: socktainer lists all 14 images, OrbStack none.
   Through DDEV's `FindImagesByLabels`, `ddev delete` deleted every image on the machine,
   including `ddev-utilities`, `socktainer-dns:embedded` and apple container's own
   `vminit:0.53.0`, and every `ddev start` tried to delete `socktainer-dns:embedded` as a
   dangling project image ("Unable to delete sha256:5545… referenced in multiple
   repositories"). The branch now re-checks the labels client-side.
2. **`docker cp` into a running container drops files under a volume mount.** Directories
   arrive, files don't; the same copy onto the rootfs is complete. Repro
   `handoff/cpvolrepro/repro.sh`. Cause: `putArchiveViaCopyIn` creates directories with an
   in-guest `sh` exec, which sees the mount, but sends files through apple container's
   `copyIn`, whose `vminitd.copy` writes to `root` + path — the container's rootfs as the VM
   sees it, outside the container's mount namespace — so files land under the mountpoint,
   hidden. Possibly an apple/containerization issue rather than socktainer's; decide where to
   file after reading `Vminitd.copy`. It left DDEV's `ddev-router-traefik` volume empty, so
   Traefik fell back to its defaults and failed binding `:80`. The branch extracts with
   `tar` inside the container instead on this provider.
3. **Hostname uniqueness** — still the main remaining DDEV gate; no fix exists. File as an
   issue with the proposal in "Open blockers" below, with a DDEV-independent Compose repro (a
   service recreated while its old container still exists). Upstream's closed #385 (Compose
   recreate and rename) is related background.
4. **socktainer accepts any API version in the path.** `/v1.99/version` answers 200, where
   Docker Engine returns 400. Low priority; file only if it matters to a client.
5. **The exec-path crash seen once** (see "Smaller gaps") — file only if it recurs.

Done since 2026-10-12: [socktainer#456](https://github.com/socktainer/socktainer/pull/456)
(probe timeout, issue #455) and [#458](https://github.com/socktainer/socktainer/pull/458)
(`HostConfig.PortBindings`, issue #457) merged;
[#459](https://github.com/socktainer/socktainer/issues/459) (`Server.Version`) closed by
agreement — socktainer's version tracks Apple Container's, and we no longer propose
reporting a Docker Engine version. Earlier: [#439](https://github.com/socktainer/socktainer/pull/439)
(exec, someone else's), [#445](https://github.com/socktainer/socktainer/pull/445) (DNS) and
[#447](https://github.com/socktainer/socktainer/pull/447) (probe user).

Per-item steps: search upstream by topic; confirm a test fails without the fix (if the new
test needs new API, show it fails with the bug put back, or add one using only existing API);
write a DDEV-independent repro in `handoff/`; live-test on stock and on the fix and compare
with Docker Engine (`DOCKER_CONTEXT=orbstack`); issue with a step-by-step description of what
happens, linking the PR/commit that introduced the behavior; then PR.

Per-item checklist, from socktainer's own `AGENTS.md`: Conventional Commits title; DCO
`Signed-off-by` on every commit (`git commit --signoff`); `make fmt` leaves no diff; tests pass;
a bug fix carries a unit test that fails without the fix; PR body follows
`.github/PULL_REQUEST_TEMPLATE.md`; no labels unless asked. Commits here are also GPG-signed
through 1Password, which needs the user present to approve — an agent's `git commit` fails
with "1Password: failed to fill whole buffer" otherwise. Never bypass signing.

## Where this stands

**It works end to end on stock upstream socktainer `main` (2026-10-31) with apple container
1.6.1.** Verified 2026-10-31: `appletest` and `appletest2` run side by side through one
router, both answering 200 over HTTP (8080) and HTTPS (8443); `ddev exec`, `ddev mysql`,
`ddev snapshot`, `ddev restart`, `ddev start` over a running project with unchanged config,
`ddev stop`, and `ddev delete` of a throwaway project (router files, volumes and only its own
images removed); `handoff/exec-suite.sh` passes; three `ddev poweroff`/`ddev start` cycles
of both projects came back clean.

**What still fails:** anything that makes Compose recreate a container while the old one
exists — `ddev snapshot restore` (re-verified 2026-10-31: "hostname(s) already exist:
["appletest-web", "appletest-web"]"), and `ddev start` over a running project whose config
changed. See "Hostname uniqueness" below. `ddev restart` recovers from both.

Merged DDEV-side hardenings from this work: [#8657](https://github.com/ddev/ddev/pull/8657),
[#8658](https://github.com/ddev/ddev/pull/8658), [#8659](https://github.com/ddev/ddev/pull/8659),
[#8928](https://github.com/ddev/ddev/pull/8928), and upstream's split of the global cache
for [#8895](https://github.com/ddev/ddev/issues/8895):
[#8945](https://github.com/ddev/ddev/pull/8945) (router's own `ddev-router-traefik`),
[#9024](https://github.com/ddev/ddev/pull/9024) (per-project web and db cache volumes),
[#8956](https://github.com/ddev/ddev/pull/8956) (global commands copied into each container).
Draft: [#8656](https://github.com/ddev/ddev/pull/8656), this branch.

### What still has to be done by hand

- `omit_containers: [ddev-ssh-agent]` in each project — see "ssh-agent" below.
- `ddev config global --router-http-port=8080 --router-https-port=8443`. The **global**
  setting matters: with 80/443 there, the router fails with "listen tcp :80: bind:
  permission denied" even when the project sets 8080/8443.
- A `traefik_monitor_port` other than OrbStack's 10999 (11999 here), when another provider's
  DDEV router might also be running.
- `ddev restart` rather than `ddev start` over a running project with changed config.

Mutagen needs no per-project setting: it is forced off automatically on this provider.

## DDEV changes on the branch

After upstream's #8895 work, only `ddev-global-cache` still needs a stand-in. It now holds
just `mkcert/`, but every project's web mounts it read-write and db mounts it read-only,
which apple container can't share (see "Named volumes" below). The router volume and the
per-project cache volumes each have one writer and are used as upstream ships them.

- `pkg/dockerutil/providers.go` — `IsSocktainer()`, `IsAppleContainer()`,
  `UseBindGlobalCache()`, `GlobalCacheSource()`, `GlobalCacheMount()`. On this provider
  `/mnt/ddev-global-cache` is a bind mount of `~/.ddev/global-cache-bind`;
  `DDEV_BIND_GLOBAL_CACHE` forces it either way for testing.
- `app_compose_template.yaml`, `config.go` — web and db mount `{{ .GlobalCacheMount }}`, and
  the `ddev-global-cache` volume isn't declared when it's bind-mounted.
- `pkg/ddevapp/ddevapp.go` — create the host directory instead of the volume; copy the
  mkcert CA there on the host (`copyIntoGlobalCache`); the start-chown helper skips the db
  volume and each project cache volume whose container is already running
  (`isServiceRunning`), since a second writer is refused; `Stop()` cleans the router's files
  through `runInRouterTraefikVolume` (in `router.go`), an exec in the running router rather
  than a helper container beside it, and cleans legacy history through `GlobalCacheMount()`.
- `pkg/dockerutil/containers.go` — `CopyIntoContainer` extracts with `tar` through
  `ExecWithStdin` on this provider, working around the volume copy bug above.
- `pkg/dockerutil/images.go` — `FindImagesByLabels` re-checks labels on what the API
  returns, working around the image filter bug above. Applies on every provider.
- `performance_mode.go`, `mutagen.go` — force Mutagen off with a `WarningOnce`;
  `no_bind_mounts` warns separately, since it depends on Mutagen.
- `environment.go`, `version.go`, `utility-port-diagnose.go`, `addons.go` — report the
  provider as `apple-container`, and give add-on installs the virtiofs propagation delay.

Dropped in the 2026-10-31 rebase, as upstream made them unnecessary: the router's bind-mounted
Traefik directory, `NotifyRouterOfTraefikConfigChange()`, `mergeCopyDir()` and the
host-side list/remove helpers, and the global-commands copy into the cache.

### Reviewing the 2026-10-31 rebase

The rebase onto `e907890db` resolved conflicts in ways that would have broken every
provider: `app_compose_template.yaml` was left with an unbalanced `{{ if not
.BindGlobalCache }}` (no `{{ end }}`, wrapping `web-cache` and the rest of the volumes), and
web and db had lost `{{ .GlobalCacheMount }}`, so the bind mount was never used. The router
template gated the declaration of `ddev-router-traefik` on the bind mount while the service
still mounted it, and `traefik.go` still pushed config into the bind-mounted directory, which
the router no longer reads. All fixed as described above.

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

Still present on container 1.6.1 and upstream `main`. It breaks `ddev snapshot restore`
outright: restore removes the db container and runs `compose up`, which recreates web as a
dependent while the old web still exists. `ddev restart` is unaffected, because `Stop()` runs
`compose down` first. The running project keeps serving through the failure.

The likely socktainer-side fix: keep the requested hostname as the container's own hostname,
but stop passing it as the network attachment hostname. That name feeds Apple Container's own
DNS, which socktainer doesn't use since it runs its own. No issue filed yet. Upstream's open

socktainer#487 (a mode resolving names with apple/container's DNS) touches the same area

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

Where it still hits DDEV after #8895: `ddev-global-cache` (only `mkcert/` now; rw in every
web, ro in db), any helper container that mounts a volume a running container holds (handled
on the branch as described above), `ddev-ssh-agent_socket_dir`, and Mutagen, which mounts
`project_mutagen` twice inside the web container alone. If web mounted `ddev-global-cache`
read-only, with DDEV seeding it while no project runs, the bind mount could go too — but
add-ons may write their own directories there today.

### ssh-agent cannot work as designed

Not a volume-mode problem. Web would only need the socket volume read-only, but `:ro` still
can't attach while ssh-agent holds it read-write. And even if it could, a unix socket belongs
to the kernel that created it: every container is its own VM, so web could see the socket file
but never connect to the agent in another VM. A virtiofs bind mount has the same limit. It
needs a different transport — the agent listening on TCP on `ddev_default` with `socat` in web
bridging `/home/.ssh-agent/socket`, or the agent running inside web — which changes DDEV's
ssh-agent design on every provider.

### Ports below 1024 cannot be published

Apple Container's port forwarder runs unprivileged, so use router ports 8080/8443 (see "What
still has to be done by hand"). Same restriction as rootless Podman, except that macOS has no
`ip_unprivileged_port_start` equivalent.
[apple/container#1985](https://github.com/apple/container/issues/1985) narrows it to explicit
low bind addresses (`-p 127.0.0.1:80:80`); DDEV publishes without one.

### Smaller gaps

- **Hot-attach of networks is unsupported.** Multi-homing at create time works; `docker
  network connect` afterwards is a no-op, since Virtualization.framework has no NIC hotplug.
- **The healthy marker's group is 0, not the container's 20**, even with probes running as
  the configured user. Harmless to DDEV, since the owner is right.
- **A buildkit node that has exited must be recreated, never restarted** — a restarted one
  fails every build with `rbind … operation not permitted`. Not re-checked on 1.6.1; the
  stale `buildx_buildkit_*` containers were removed on 2026-10-31 and buildx bootstrapped
  fresh ones.
- **One daemon crash, not reproduced.** Stock upstream `main` (2026-10-12) died on a
  `ddev poweroff`: a SwiftNIO precondition in `Posix.close` (most likely `EBADF`), from
  `channel.close` in `DockerTCPProtocolUpgrader.upgrade`'s error path, right after two exec
  starts. Report: `~/Library/Logs/DiagnosticReports/socktainer-2026-10-12-101205.ips`.
- **`docker images --format` is ignored** on socktainer; the repro output shows unformatted
  names. Cosmetic.
- **Upstream's DNS forwards non-container names to the vmnet gateway** (#426), which depends
  on macOS's own resolver answering on `192.168.64.1` — see the next section.

## Resolved: no DNS on the vmnet gateway was this machine

The long-standing "no DNS on the built-in `default` network" was an artifact of this machine:
a root Homebrew dnsmasq service (`sudo brew services`, not visible in a plain `brew services
list`) held port 53, so macOS's vmnet DNS proxy never answered on `192.168.64.1`. With it
stopped and `container system` restarted, the gateway resolves on its own. Since upstream

socktainer#426 forwards every non-container lookup to the gateway, a dnsmasq workaround would now loop

## Recipes

### Teardown and clean rebuild

**Always `ddev poweroff` before switching Docker contexts or socktainer binaries.** A router left
running keeps 8080/8443/11999 bound, and a daemon restart resets every re-adopted container's
health to `starting`.

```bash
# 1. tear everything down
ddev poweroff
pkill -f "\.build/release/socktainer$"
pgrep -fl "release/socktainer$"                  # must print nothing
/usr/local/bin/container system stop
rm -f ~/.socktainer/container.sock

# 2. build upstream main
cd ~/workspace/socktainer        # tmp/upstream-main-test, tracks upstream/main
git fetch upstream && git merge --ff-only upstream/main
swift build -c release           # ~260s incremental after a dependency bump

# 3. bring both systems back up
/usr/local/bin/container system start            # the signed installer explicitly
nohup ./.build/release/socktainer >> ~/tmp/socktainer.log 2>&1 &
docker context use socktainer                    # socktainer creates the context itself

# 4. confirm
/usr/local/bin/container system status | grep -E 'installRoot|server.version'
docker version --format '{{.Server.Platform.Name}} {{.Server.APIVersion}}'
```

Check that only one daemon runs: on 2026-10-31 a daemon started on 2026-10-12 was still
alive. One daemon at a time: they share the socket and DNS port.

socktainer's tests run here with
`swift test -Xswiftc -plugin-path -Xswiftc /Library/Developer/CommandLineTools/usr/lib/swift/host/plugins/testing`;
`make test` has no clean way to pass the flag. Also run `make fmt` and check `git status`.

Homebrew's `container` formula sits ahead of `/usr/local/bin` on `PATH`, so always call
`/usr/local/bin/container` for `system start`/`status`. Do not `brew install socktainer` from
homebrew-core here: it requires Homebrew's own `container`
([socktainer/socktainer#392](https://github.com/socktainer/socktainer/issues/392)), and
socktainer's auto-start bootstrap runs `container system start` with whichever binary is
first on `PATH`.

To upgrade `container`: `gh release download <ver> -R apple/container -p
'*installer-signed.pkg'`, check it with `pkgutil --check-signature`, then
`sudo installer -pkg … -target /` with the apiserver stopped.

### Test tools

Kept in [`handoff/`](handoff/) on this branch; run them from the repo root. All the repros
take `DOCKER_CONTEXT=orbstack` to compare with Docker Engine.

- `handoff/imgfilterrepro/repro.sh` — `docker images --filter label=…` matching nothing.
- `handoff/cpvolrepro/repro.sh` — `docker cp` of a directory into a running container, at a
  volume mount and on the rootfs.
- `handoff/testdata/execrepro/` — non-interactive exec through the moby Go client, as DDEV does
  it: `go run ./handoff/testdata/execrepro <container> '<cmd>'` with
  `DOCKER_HOST=unix://$HOME/.socktainer/container.sock`; `CLOSEWRITE=1` half-closes like the
  CLI. Under `testdata` so `./...` skips it.
- `handoff/exec-suite.sh <label>` — small and 15 MB outputs, exit codes, piped stdin, a 20s
  exec, `-d`.
- `handoff/dnsrepro/repro.sh`, `handoff/hcrepro/repro.sh`, `handoff/hctimeout/repro.sh`,
  `handoff/pbrepro/repro.sh` — repros for the merged socktainer#444, #446, #455 and #457.
- `handoff/bxtest.Dockerfile` — a trivial build for `docker buildx build --no-cache`.
- `handoff/wt-status.sh` — every socktainer worktree's branch against `rfay` and
  `upstream/main`.

To capture raw API traffic, put a logging proxy in front of the socket:
`socat -v UNIX-LISTEN:$HOME/tmp/proxy.sock,fork UNIX-CONNECT:$HOME/.socktainer/container.sock 2> ~/tmp/socat.log`,
then point `DOCKER_HOST` at the proxy.

## Current machine state (2026-10-31)

- Apple `container` **1.6.1** from the signed installer at `/usr/local`
  (`container-apiserver` running). Homebrew's `container` is also 1.6.1.
- socktainer running by hand from `~/tmp/socktainer-imgfilter/.build/release/socktainer`
  (upstream `main` `8ea2350` plus #492). Stock upstream `main` is built at
  `~/workspace/socktainer/.build/release/socktainer`. Log: `~/tmp/socktainer.log`.
- Docker context **`socktainer`**.
- Global router ports 8080/8443, `traefik_monitor_port` 11999.
- `appletest` and `appletest2` (`~/tmp/`) powered off, both served 200 when last up; both have
  `omit_containers: [ddev-ssh-agent]`. `appletest3` untested; `appledel` was a throwaway,
  deleted (its `.ddev` remains in `~/tmp/appledel`).
- The old `ddev-global-cache` block volume still exists on socktainer; nothing mounts it.
  `ensureRouterTraefikVolume` seeded `ddev-router-traefik` from it once.
- The root Homebrew dnsmasq service is stopped.

### Repositories and worktrees

socktainer, main checkout `~/workspace/socktainer` (remotes `upstream`, `rfay`), on
`tmp/upstream-main-test` (local only, fast-forwarded to upstream `main`). Worktree
`~/tmp/socktainer-imgfilter` holds `fix/image-list-label-filter`, pushed to `rfay` for
upstream PR #492. The worktrees `~/tmp/socktainer-hcfix-timeout` (#456) and `~/tmp/socktainer-portbindings` (#458) are
merged upstream and can be removed, as can
`~/workspace/socktainer-worktrees/version-engine-version` (#459 closed). Drafts and posted
text for every upstream issue, PR and fork comment are in `~/tmp/sk-upstream/`.

DDEV worktree: this branch is in
`~/workspace/ddev-worktrees/20260802_rfay_apple_container_experiment`.
