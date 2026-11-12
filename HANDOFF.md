# DDEV on Apple Container + socktainer — investigation notes

Working notes for [#7372](https://github.com/ddev/ddev/issues/7372), started 2026-08-02 and
last verified 2026-11-09, release status updated 2026-11-11; CI bring-up started 2026-11-01. This documents an exploration, not a feature: the DDEV changes here
only take effect when the Docker provider is socktainer, and this file is expected to be
deleted before any of it merges.

Environment as last verified: Apple `container` 1.6.1 (signed installer, **not** Homebrew — see
below), `socktainer` built from upstream `main` as of 2026-11-09 with no fork fixes (the same
commit as the v1.6.1 release, `71f2415`), docker CLI 29.9.2, docker context `socktainer`,
macOS 27.0.1 (26A434), `ddev` from branch
[`20260802_rfay_apple_container_experiment`](https://github.com/rfay/ddev/tree/20260802_rfay_apple_container_experiment),
rebased on `upstream/main` as of 2026-11-12, after [#9115](https://github.com/ddev/ddev/pull/9115).

## Resume here

### General changes to move to their own PR (2026-11-12)

These change behavior on every provider, not just Apple Container. Moving them to a
separate PR from `upstream/main` makes this PR smaller and Apple-Container-only to review,
and gets the fixes to other providers sooner. The branch drops them on its next rebase
after they merge. Everything else in the diff is gated on `IsAppleContainer()`/`IsSocktainer()`,
adds Apple Container next to an existing rootless-Podman check, or is socktainer CI, docs
or `handoff/`.

| Change | Files | Commit on this branch | Why it's general |
| --- | --- | --- | --- |
| Captures read the pipe from the start | `pkg/util/capture.go`, `pkg/util/utils_test.go` (`TestCaptureUserOutLarge`) | `fix(test): read captured output as it is written` | A captured write past the pipe buffer deadlocked on any platform; it hung build 24 for 5h |
| `TestTimezoneConfig` reads the zone before `TZ=""` | `pkg/ddevapp/config_test.go` | `test: read host zone before clearing TZ` | Fails on any provider when nothing in the process loaded `time.Local` first |
| Pull platform is `linux/<arch>` | `pkg/dockerutil/images.go` (`BuildPlatformToPull`), `images_test.go`, `containers_test.go` (`TestMain`) | `fix(apple-container): pull linux images on socktainer` (part) | Replaces `serverVersion.Os` everywhere. Correct on every provider, since DDEV images are Linux-only, but a behavior change reviewers should see on its own. The alternative is gating it on `IsSocktainer()` here |
| `slices.Concat` in `GetOmittedContainers` | `pkg/ddevapp/ddevapp.go` | `feat: experimental Apple Container/socktainer support` (part) | Upstream's `append` onto `OmitContainersGlobal` can write into the global list's spare capacity |
| Local Network prompt step for socktainer testbots | `docs/content/developers/buildkite-testmachine-setup.md` | `feat: experimental Apple Container/socktainer support` (part) | Upstream already documents socktainer testbot setup |

All five are on branch `20261112_rfay_capture_tz_platform_fixes` (worktree
`../20261112_rfay_capture_tz_platform_fixes`), one commit each, from `upstream/main`, in
[#9108](https://github.com/ddev/ddev/pull/9108). `TestTimezoneConfig` and the `pkg/dockerutil` image tests passed there on
OrbStack. The router-port test fix is gone from this branch: upstream fixed the same test in
[#9105](https://github.com/ddev/ddev/pull/9105), with 9443, and the 2026-11-12 rebase took
theirs.

Stacking: rebase this branch onto the split branch (`git rebase
20261112_rfay_capture_tz_platform_fixes`). The capture commit drops as already applied, the
tree is unchanged, and the one conflict is `GetOmittedContainers`, where this branch's
version wins; `rerere` has recorded it. After the split PR is squash-merged, drop its
commits with `git rebase --onto upstream/main 20261112_rfay_capture_tz_platform_fixes`,
which needs no patch matching. If the split branch changes in review, restack with
`git rebase --onto <new split tip> <old split tip>`. On the restacked branch, the
`test: read host zone before clearing TZ` commit keeps only #508 skip removals, so squash it
into `fix(apple-container): socktainer 1.6.1, serial builds`.

### socktainer v1.6.1 (released 2026-11-09)

[v1.6.1](https://github.com/socktainer/socktainer/releases/tag/v1.6.1) is upstream `main` as
of 2026-11-09 (`71f2415`); nothing has merged since. It has every fix this work was waiting
on: #492 (image `label` filter), #494 (`State.ExitCode`), #496 (`*.sock` bind mounts), #506
(#479, peer hostname rewrite), #507 (`created` state), #509 (#508, hostname
uniqueness) and #511 (#467, `docker cp` under mounts). It also builds against apple container 1.6.1 and
containerization 0.53.0 (#485), not 1.7.1. Homebrew has socktainer 1.6.1, installed here and
first on `PATH`.

Still open upstream as of 2026-11-11: issues #484, #497, #498, #499, #502, #504, #505,
the newer issues #510, #514, #517 and #518, and the pull requests #487, #513
(ours, for #505), #515 (ours, for #514) and #516 (apple container 1.7.1 and containerization 0.54.0).

Done on the branch 2026-11-11 (uncommitted), run against Homebrew's socktainer 1.6.1:

- `SocktainerMinVersion` is 1.6.1; `ddev utility dockercheck` reports `socktainer version: 1.6.1`.
- Removed the #508 hostname-uniqueness skips (11 in `pkg/ddevapp` and `pkg/testcommon`, 6 in
  `cmd/ddev/cmd`) and the four #493 exit-code skips; six test files now match upstream.
  `TestPopulateGlobalCustomCommandFiles` and `TestCmdDescribe` keep a skip, now naming only
  #514 and the `ExposedPorts` gap.
- Removed the #479 `DDEV_PRIMARY_URL` exception in `TestEnvironmentVariables`.
- Removed the `*.sock` check in `AppleContainerSSHAgentSocket`; `TestAppleContainerSSHAgent`
  now uses a socket named `agent.sock`. The user docs drop the socat relay and the 1.6.0
  hostname limitation, and require socktainer 1.6.1.
- **`CopyIntoContainer` keeps its in-container `tar`**, deliberately: socktainer's archive
  copy-in still ignores gzip (#484) and drops owner and mtime (#514), which `tar` keeps.
  Revisit when #515 is released.
- The testbots run socktainer 1.6.1 (per rfay, 2026-11-11).

Tests on Homebrew socktainer 1.6.1 (2026-11-11, with OrbStack stopped; logs in
`~/tmp/sk161-tests2/`): all 21 un-skipped tests pass, plus `TestEnvironmentVariables` and
`TestAppleContainerSSHAgent` (now with an `agent.sock` socket), except two that needed fixes:

- `TestTimezoneConfig` failed every run: **an upstream test bug, not socktainer's**. It
  calls `t.Setenv("TZ", "")` before anything has used `time.Local`, and Go loads
  `time.Local` lazily, so it becomes UTC and the test expects `timezone=UTC` while the
  container correctly says `MST`. It passes elsewhere only when something earlier in the
  process touched `time.Local`. Fixed in `config_test.go` by reading the host zone before
  clearing `TZ`; passes after the fix. Worth its own upstream PR.
- `TestStartProfileImages` failed 3 of 5 runs: **a new socktainer bug**, filed 2026-11-11 as
  [socktainer#518](https://github.com/socktainer/socktainer/issues/518); PR to follow once #513 or #515 lands.
  Two `docker load`s at once can leave one tag on its old image: 4 of 16 tags stale with
  concurrent loads, 0 of 16 serially (`handoff/inlinerecreaterepro/concurrent-load.sh`,
  `SERIAL=1` for the control). Compose's bake build loads every built image at once, so
  after `ddev start` rebuilds, a service can keep its old image; Compose then sees no
  change and doesn't recreate it (`handoff/inlinerecreaterepro/repro.sh`). Users hit it
  too, since every `ddev start` builds web and db together. Cause: socktainer makes a new
  containerization `ImageStore` per request (`ClientImageService.swift:609`, also 704 and
  790), and `ReferenceManager.create` does an unlocked read-modify-write of `state.json`;
  each actor serializes only its own calls, so the later save puts back the other
  reference. Fix upstream: share one `ImageStore` (or serialize the writes). DDEV
  workaround: `composeBuild` builds one service per `Build` call on Apple Container
  (`compose_build.go`); `WithMaxConcurrency` doesn't reach the bake path. With it, the
  test passed 3 of 3 runs; builds are slower than parallel ones.
- Also seen, unfiled and harmless to DDEV so far: inspect's `.Image` is the reference rather
  than the image ID, and `docker images -q` reports an ID that differs from inspect's and
  changes on every load of the same image.

Buildkite [build 24](https://buildkite.com/ddev/macos-socktainer/builds/24) (2026-11-11,
tb-macos-arm64-5) ran 7+ hours and failed three ways:

- `cmd/ddev/cmd` hit the 6h test timeout: `TestTLSDiagnoseSmoke` blocked 308 minutes in a
  `write` to its `CaptureUserOut` pipe after about 570 bytes. `util.CaptureUserOut`,
  `CaptureUserErr` and `CaptureStdOut` started reading only at `restore()`, so any write
  past the pipe buffer deadlocks. macOS shrinks new pipes to about 512 bytes when pipe
  memory runs short, which fits: the test passes alone and passed in build 21. Unconfirmed
  on the testbot. Fixed in `pkg/util/capture.go` (reads from the start), with
  `TestCaptureUserOutLarge`, which hangs on the old code. The pipeline now passes
  `TEST_TIMEOUT=3h`, so a hang costs 3h, not 6h.
- `TestWriteConfigKeepsProjectRouterPorts` (upstream #9075) set the project's HTTPS port to
  8443, which the pipeline sets globally, so `WriteConfig` left it out. Fails on any machine
  with global 8443, here too. Upstream fixed it in [#9105](https://github.com/ddev/ddev/pull/9105).
- `TestUtilityDiagnoseCmd/BasicProject`: "No internet access from containers", 13s after the
  same check passed in another subtest. Intermittent; not investigated.

### State (2026-11-09)

- **Review pass of 2026-11-09 (committed in `fix(apple-container): version check, auth ssh hints, docs`):** `CheckDockerVersion` no longer returns an
  error when socktainer reports no `socktainer` component, since that would have made
  every `ddev start` warn; the version is just not compared. `ddev utility dockercheck`
  prints the socktainer version. `ddev auth ssh` with `ssh_agent_upstream` already set
  but unusable (a `*.sock` socket) now points at the warning instead of telling the user
  to set `--ssh-agent-upstream=host` again. `perf.sh` no longer omits `ddev-ssh-agent`
  globally, since DDEV omits it itself. The user docs no longer tell people to omit
  `ddev-ssh-agent` per project or list the mkcert CA as a limitation; they document
  `--ssh-agent-upstream=host`, the `*.sock` relay, `docker-buildx`, and
  `host.docker.internal` listening on all interfaces. The PR description was
  replaced 2026-11-11 from `~/tmp/pr-8656-body.md` (1.6.1, #517, #518; the 2026-11-09 draft
  is `~/tmp/pr-8656-body.2026-11-09.md`).
- **A `brew upgrade` stops Apple Container.** On 2026-11-09 it upgraded Homebrew's
  `container` to 1.7.1 and booted out the running apiserver mid-test; restarting the signed
  1.6.1 install then hit vmnet 1001 until a reboot. Homebrew's `container` (now 1.7.1) is
  ahead of `/usr/local/bin` on PATH.
- **socktainer `main` fixes the main blocker.** #509 (hostname uniqueness), #506 (#479),
  #507 (`created` state) and #511 (#467, `docker cp` under mounts) all merged 2026-11-09
  and are in v1.6.1. On `main`: `ddev snapshot restore` and `ddev start`
  over changed config work, and `DDEV_PRIMARY_URL` survives a router recreate.
- **The #508 skips on `main`, run with the skips removed:** `TestStartProfileImages`,
  `TestDdevRestoreSnapshot`, `TestProjectPortOverride`, `TestUseRouterSubstitutePort`,
  `TestCorepackEnable`, `TestTimezoneConfig`, `TestConfigDefaultContainerTimeout`,
  `TestMailpit`, `TestRestartServices`, `TestExtraPortExpose`, `TestGetLocalHTTPResponse`,
  `TestLaunchCommand`, `TestNpmNpxYarnCommands`, `TestCmdStartOptionalProfiles`,
  `TestCmdSnapshotRestore`, `TestCmdSnapshotRestoreFile`, `TestCmdRestartProfilesAndServices`
  pass, as do `TestGlobalPortOverride` (now un-skipped) and `TestEnvironmentVariables`.
  Three won't pass on a release with #509 alone:
    - `TestGetWebContainerDirectURLsWithGenericWebserver` needs router ports 80/443; it now
      skips for privileged ports.
    - `TestPopulateGlobalCustomCommandFiles` fails its mtime check intermittently (2 of 3
      runs). Cause: `docker cp` into a running container drops the archive's mtime and
      ownership. Every file gets the time of the copy and `0:0` (tested with a tar of six
      files dated 12:00:01–06 owned 33:33). The test passes only when the copy lands in
      the same rounded second the files were created. Since #511, `putArchiveViaCopyIn`
      writes each file with `sh -c 'cat > f && chmod mode f'`; before it, Apple
      Container's `copyIn(source:destination:mode:)` took no owner or time either, so
      ownership was likely lost before #511 too; mtime unknown. Docker Engine keeps both.
      DDEV impact is small: `CopyIntoContainer` extracts with `tar` in the container.
      A never-started container loses both too; a stopped one keeps them. Filed
      2026-11-09 as [socktainer#514](https://github.com/socktainer/socktainer/issues/514),
      repro `handoff/cpmetarepro/repro.sh`. Fix for the running case in PR
      [socktainer#515](https://github.com/socktainer/socktainer/pull/515) (branch
      `fix/archive-owner-mtime`, worktree `~/workspace/socktainer-worktrees/archive-owner-mtime`);
      the test passed 4 of 4 runs on it. The never-started case is still open.
    - `TestCmdDescribe`: `ddev describe` omits `web:5492`, a Compose `expose:` port. Cause:
      inspect builds `Config.ExposedPorts` from `configuration.publishedPorts` only
      (`ContainerInspectRoute`). The request's `ExposedPorts` and the image's `EXPOSE` are
      read at create only for `PublishAllPorts` and never stored, since Apple Container's
      configuration has no field for them. A fix would persist them in a label, as the
      healthcheck already is. DDEV reads unpublished ports only from there. Not filed;
      searched upstream 2026-11-09.
- **socktainer fix for #505** (aliases on reconnect): PR
  [socktainer#513](https://github.com/socktainer/socktainer/pull/513), opened 2026-11-09
  from `rfay:fix/network-connect-aliases` (worktree
  `~/workspace/socktainer-worktrees/network-connect-aliases`). Push it with an explicit
  refspec: `push.default` is `tracking`, so a branch made from `upstream/main` pushes to
  `rfay/main` otherwise. Unit tests pass (1314 total; the 6 new ones fail with the fix
  removed). Live: the #505 repro's reconnect alias resolves, a later reconnect replaces it,
  it follows the container's new IP after a restart, and removal drops it.
  `TestTraefikSharedTLDCerts` passes with its skip removed. `TestRouterNotRebuiltOnHostnameChange`
  still fails, reading the router's aliases from inspect, which this leaves to
  [#487](https://github.com/socktainer/socktainer/pull/487) (it touches the same files).
- **Skips:** `SkipUnlessDefaultEnvironment` no longer lists Apple Container, but on the
  socktainer pipeline its Mutagen check still skips those 17 tests, since Mutagen is the
  macOS default. Every Apple Container skip now names an issue except healthcheck output
  (unfiled) and the structural ones (ssh-agent, privileged ports).
- **Try apple container 1.7.1** (released 2026-11-06) locally and on the testbots once
  socktainer supports it; v1.6.1 builds against 1.6.1, and open PR #516 bumps it to 1.7.1.
- **Remove `HANDOFF.md` and `handoff/`** before review. Move what matters into the PR
  description or #7372.
- **Decide the release stance:** ship marked experimental with the hand steps documented;
  v1.6.1 removed the wait on #509.
- **Buildkite:** the CI pieces merged in [#9115](https://github.com/ddev/ddev/pull/9115), so
  `main` now has `macos-socktainer.yml`. Widen `macos-socktainer`'s branch filter, and
  decide whether it's a required check.

### Buildkite CI (2026-11-01)

The test suite now has a socktainer pipeline, following the plan in
[this PR comment](https://github.com/ddev/ddev/pull/8656#issuecomment-6562617661).

- Pipeline [macos-socktainer](https://buildkite.com/ddev/macos-socktainer) runs
  `.buildkite/macos-socktainer.yml` with `DOCKER_TYPE=socktainer` on agents tagged
  `socktainer=true` (tb-macos-arm64-5/6/7; each `buildkite-agent.cfg` has a
  `.bak-20261101` backup). It builds fork PRs only from branches matching
  `*20260802_rfay_apple_container_experiment`, and doesn't build branches. Its GitHub
  webhook is on ddev/ddev. The pipeline, `lib-provider.sh`, `test.sh`, `sanetestbot.sh`,
  perf and testbot-setup changes merged in [#9115](https://github.com/ddev/ddev/pull/9115)
  and are no longer in this branch's diff.
- The testbots have Homebrew's apple container 1.6.1 and socktainer 1.6.1, neither run as a
  service; [#9037](https://github.com/ddev/ddev/pull/9037) (testbot disk cleanup) prepares
  them. 1.6.1 honors the image `label` filter (#492); upstream DDEV also re-checks labels
  client-side (#9040), which kept `testbot_maintenance.sh` safe on 1.6.0.
- `lib-provider.sh`'s `socktainer` case runs `container system start
  --disable-kernel-install`, starts `socktainer --no-auto-start` in the background (log
  `~/tmp/socktainer.log`), waits for it, and removes leftover containers.
  `provider_shutdown` runs `pkill -x socktainer` and `container system stop`.
- `test.sh` sets router ports 8080/8443 globally. It used to omit `ddev-ssh-agent` globally
  too, which is why CI never hit the shared socket volume that broke `ddev start` of any
  project with default config (found 2026-11-03 on `~/workspace/d11`); DDEV now omits it on
  Apple Container itself.
- Skipped on `IsAppleContainer()`: see "Test skips to rethink" below.

**Build 1** (tb-macos-arm64-6) got through bring-up, the build and `testbot_maintenance.sh`,
then `sanetestbot.sh` hit its 60s timeout: `docker run -v $HOME:/tmp/junker99 …` never
returns from `start` on the testbots (over ssh, too), though it does on rfay's machine. With
no mount, or a small `~/tmp` directory mounted, the container starts in about a second.
`sanetestbot.sh` now mounts `$HOME/tmp` for socktainer. Why mounting all of `$HOME` hangs
there is unknown (size, or a TCC-protected folder under `~/Library`, are guesses).

**Build 11** (2026-11-03, `aea267838`) was the first full test run; builds 7–10 were
`[skip ci]` commits, so don't use `[skip ci]` on this branch. 22 tests failed:

- Fixed in DDEV: the mkcert CA copy (item 7 below), `host.docker.internal` (empty on Apple
  Container, so Xdebug never reached the host; now `host-gateway`, which socktainer resolves
  to the network gateway where the Mac answers), `TestPoweroffOnNewVersion` (expected
  `ddev-ssh-agent`), and `TestUtilityCheckCustomConfigCmd`'s stale-dbimage subtest
  (socktainer's builder is buildx's `docker-container` driver, so `docker build` needs
  `--load`).
- Skipped with the cause named: hostname uniqueness (8 tests), socktainer#497 (2),
  #498 (1), #493 (2), single-writer volume (`TestCmdAddonGetWithDotEnv`), `Os: macOS`
  (`TestFindServiceImages`), and #499 (`TestCmdExec`'s working-directory check only).
- `TestDdevXdebugEnabled` panicked (`send on closed channel`: its accept goroutine sends on
  `acceptListenDone` after the test closed it), which aborted the rest of `pkg/ddevapp`. That
  is an upstream test bug; the `host-gateway` fix should keep it from triggering here.

**Build 12** (tb-macos-arm64-7) failed every host↔container connection: macOS had never
been allowed to give `container-runtime-linux` Local Network access there. Allowed by
hand over Screen Sharing; the step is now in `buildkite-testmachine-setup.md`.

**Build 13** was the first to run all of `pkg/ddevapp` (the Xdebug panic is fixed in the
test). Of its 18 failures:

- Fixed: `TestDdevXdebugEnabled` listened on `127.0.0.1`, but VMs reach the Mac at the
  vmnet gateway, so it listens on all interfaces here; `TestNamedVolumesHaveOneWriter`'s
  stale-entry check (nothing shares `ddev-ssh-agent_socket_dir` once ddev-ssh-agent is
  omitted).
- Skipped for hostname uniqueness: `TestGetWebContainerDirectURLsWithGenericWebserver`,
  `TestExtraPortExpose`, `TestMailpit`, `TestProjectPortOverride`,
  `TestUseRouterSubstitutePort`, `TestCorepackEnable`, `TestRestartServices`,
  `TestStartProfileImages`.
- Skipped, **a DDEV bug, not Apple Container's**: `TestGlobalPortOverride`. `Start()` copies
  the resolved router port into `app.RouterHTTPPort`, and `WriteConfig` drops it only when it
  is 80 (`config.go`), so with the global port at 8080 any later `WriteConfig` pins 8080
  into the test site's config, overriding the test's global 8555. Podman rootless shares
  the 8080 setting.
- Fixed: `TestWriteableFilesDirectory` — virtiofs caches attributes for ~1 s, so a host
  append to a bind-mounted file shows in the container after ~1.1 s; the test now polls.
  `TestStopRemoveDataRemovesStaleUtilityContainers` — socktainer returns nothing for the
  container-list `volume=` filter, so `FindContainersByVolume` matches `Mounts` itself
  (which also un-skips `TestRemoveVolumeInUseNamesContainer`).
- Fixed in DDEV, socktainer#497 (`nocopy` ignored): the web image's `/var/www/html` was
  copied into the empty Mutagen volume and synced into the project directory on first
  start. [ddev#9061](https://github.com/ddev/ddev/pull/9061) ships the image with an empty
  `/var/www/html` (replacing the Podman-only cleanup), so its three skips are gone:
  `TestWebserverMissingIndexExplanation`, `TestConfigGitignore`,
  `TestComposerCmdCreateConfigInstall` all pass on socktainer (2026-11-06). Any other image
  content mounted under a `nocopy` volume would still leak until #497 is fixed.
- Skipped, **new socktainer gaps** (filed 2026-11-04):
    - [socktainer#504](https://github.com/socktainer/socktainer/issues/504): `group_add` ignored (`HostConfig.GroupAdd` decoded, never used; apple/container's
      `ProcessConfiguration.supplementalGroups` exists). The web user loses group 0, so
      `npm install -g` and anything else writing the image's group-0 directories gets
      EACCES: `TestNpmGlobalInstall`.
    - [socktainer#505](https://github.com/socktainer/socktainer/issues/505): network aliases: inspect reports `Aliases: null`, and a disconnect/connect with new
      aliases succeeds but the new alias never resolves. So the router's in-place alias
      update does nothing, and a project started after the router can't reach its own
      hostname from inside: `TestRouterNotRebuiltOnHostnameChange`,
      `TestTraefikSharedTLDCerts`.
    - [socktainer#479](https://github.com/socktainer/socktainer/issues/479): at
      container create, `rewritePeerHostnames` turns `://name:port` and `@name:port` in
      env into the IP of whatever holds `name` in socktainer's DNS table. The router
      holds each project hostname as an alias, so from a project's second start on,
      web's `DDEV_PRIMARY_URL` and `DRUSH_OPTIONS_URI` become
      `https://<router IP>:8443` (`ddev poweroff; ddev start; ddev restart` shows it).
      `TestEnvironmentVariables` ignores that one value on Apple Container; it failed
      only after `TestCustomCerts`, whose `PowerOff()` recreates the router with the
      alias. Users hit it too (`drush uli` links). Fix PR
      [socktainer#506](https://github.com/socktainer/socktainer/pull/506) makes the peer
      rewrite opt-in by label, as the loopback rewrite already is; merged 2026-11-09,
      released in v1.6.1. Verified on `main`: after `ddev poweroff; ddev start; ddev restart`,
      `DDEV_PRIMARY_URL` stays `https://appletest.ddev.site:8443`.

### Test skips to rethink (audited 2026-11-03)

A skip on `IsAppleContainer()` hides whatever else that test would have caught on this
runtime, as the ssh-agent skips plus `test.sh`'s global omit did. In order of concern:

1. **`SkipUnlessDefaultEnvironment`** (`pkg/testcommon/testcommon.go`, 17 tests) no longer
   names Apple Container (2026-11-09), but still skips there through its Mutagen check. To
   run those tests on this runtime, the pipeline would need a run with Mutagen off.
2. **ssh-agent tests** — `TestSSHAuth`, `TestSshAuthConfigOverride`, `TestSSHAgentUpstream`,
   `TestSSHAgentUpstreamHost`, `TestCmdAuthSSH`, `TestCmdAuthSSHStdin`. They check through
   the `ddev-ssh-agent` container, which no longer runs here; `TestAppleContainerSSHAgent`
   covers the replacement. `TestCmdAuthSSH` could instead assert `ddev auth ssh`'s Apple
   Container message.
3. **Hostname uniqueness** — `TestDdevRestoreSnapshot`, `TestCmdSnapshotRestore`,
   `TestCmdSnapshotRestoreFile`, `TestGetLocalHTTPResponse` (changes router ports), and from
   build 11 `TestPopulateGlobalCustomCommandFiles`, `TestTimezoneConfig`,
   `TestConfigDefaultContainerTimeout`, `TestLaunchCommand`, `TestNpmNpxYarnCommands`,
   `TestCmdDescribe`, `TestCmdRestartProfilesAndServices`, `TestCmdStartOptionalProfiles`.
   Fixed in v1.6.1 (#509); skips removed 2026-11-11, except `TestPopulateGlobalCustomCommandFiles`
   (#514) and `TestCmdDescribe` (`ExposedPorts`).
4. **socktainer gaps, remove when fixed** (the `State.ExitCode` skips, #493, went 2026-11-11):
   healthcheck output
   (`containers_test.go`, 2); `Os: macOS`, now
   [socktainer#517](https://github.com/socktainer/socktainer/issues/517), no longer skipped: DDEV pulls `linux/<arch>`;
   network names, #498 (`TestNetworkDuplicates`, `TestComposerCmdCreateRequireRemoveConfigVersion`);
   exec working directory, #499 (part of `TestCmdExec`).
5. **Single-writer volumes** — `TestCmdAddonGetWithDotEnv` (its busybox add-on mounts a volume a
   running container holds). Structural.
6. **Privileged ports** — `TestNetworkAliases` and
   `TestNetworkAliasesDifferentPorts`. Permanent; also
   means CI never runs the router on 80/443.

### socktainer upstream

Paused 2026-10-31. **Stock upstream socktainer runs DDEV end to end** on this branch, which
is now much smaller: upstream DDEV's split of `ddev-global-cache` (for
[#8895](https://github.com/ddev/ddev/issues/8895)) removed most of what the branch had to
work around, and since #9045 no DDEV container mounts `ddev-global-cache` for long. Two new socktainer bugs turned up while bringing it back, both with DDEV-side
workarounds on the branch and DDEV-independent repros in `handoff/`, neither filed yet.

socktainer's maintainer has invited issues and PRs, including AI-generated ones. Keep only a
few open at once so as not to overload review; two at a time has been fine. Each upstream
issue, PR and comment opens with this disclaimer:

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
   `fix/image-list-label-filter`), merged 2026-11-03, released in v1.6.1. `before`, `since` and `until` look
   ignored the same way, untested and left out of #492. `GET /images/json`
   applies only `dangling` and `reference` (`ImageListRoute.applyFilters`), so
   `docker images --filter label=x=y` lists every image. Released in v1.6.1. Repro
   `handoff/imgfilterrepro/repro.sh`: socktainer lists all 14 images, OrbStack none.
   Through DDEV's `FindImagesByLabels`, `ddev delete` deleted every image on the machine,
   including `ddev-utilities`, `socktainer-dns:embedded` and apple container's own
   `vminit:0.53.0`, and every `ddev start` tried to delete `socktainer-dns:embedded` as a
   dangling project image ("Unable to delete sha256:5545… referenced in multiple
   repositories"). Upstream DDEV re-checks the labels client-side since #9040.
2. **`docker cp` into a running container drops files under a volume mount.** Already
   reported upstream by someone else as
   [socktainer#467](https://github.com/socktainer/socktainer/issues/467); fixed on `main` by
   someone else's [#511](https://github.com/socktainer/socktainer/pull/511) (merged
   2026-11-09, in v1.6.1), and `handoff/cpvolrepro/repro.sh` passes there. DDEV still
   can't drop its workaround: `CopyIntoContainer` sends a **gzipped** tar, which socktainer
   silently ignores ([socktainer#484](https://github.com/socktainer/socktainer/issues/484),
   someone else's, open). Tested 2026-11-09 with the workaround disabled: the router's
   Traefik config never arrived and Traefik failed binding `:80`. Either #484 gets fixed or
   DDEV sends an uncompressed tar (Docker accepts both). Directories
   arrive, files don't; the same copy onto the rootfs is complete. Repro
   `handoff/cpvolrepro/repro.sh`. Cause: `putArchiveViaCopyIn` creates directories with an
   in-guest `sh` exec, which sees the mount, but sends files through apple container's
   `copyIn`, whose `vminitd.copy` writes to `root` + path — the container's rootfs as the VM
   sees it, outside the container's mount namespace — so files land under the mountpoint,
   hidden. Possibly an apple/containerization issue rather than socktainer's; decide where to
   file after reading `Vminitd.copy`. It left DDEV's `ddev-router-traefik` volume empty, so
   Traefik fell back to its defaults and failed binding `:80`. The branch extracts with
   `tar` inside the container instead on this provider.
3. **Hostname uniqueness** — filed 2026-11-06 as
   [socktainer#508](https://github.com/socktainer/socktainer/issues/508), fixed by PR
   [#509](https://github.com/socktainer/socktainer/pull/509), merged 2026-11-09,
   released in v1.6.1. Repro: `~/tmp/hostname-recreate-repro/repro.sh`.
4. **`docker inspect` reports `State.ExitCode` 0 for every stopped container** — filed
   2026-11-01 as [socktainer#493](https://github.com/socktainer/socktainer/issues/493), fix in PR
   [#494](https://github.com/socktainer/socktainer/pull/494) (branch `fix/inspect-exit-code`,
   worktree `~/tmp/socktainer-exitcode`), merged 2026-11-03, released in v1.6.1.
   `docker wait` is right. DDEV's `RunSimpleContainer` polls inspect, so failing helpers
   looked successful. Repro `handoff/exitcoderepro/repro.sh`.
5. **Gaps found by the first CI run (2026-11-01), not filed yet**, each with a DDEV test skip
   that names it: healthcheck `Output` is empty in inspect's `State.Health.Log` (still so on
   `main` 2026-11-09); the container list ignores the `volume=` filter (DDEV no longer uses
   it); `/version` reports `Os: macOS` while `/info` reports `OSType: linux`, filed 2026-11-11 as
   [socktainer#517](https://github.com/socktainer/socktainer/issues/517). The
   uppercase network names are part of #498. A never-started container now inspects as
   `created` on `main`, through someone else's
   [#507](https://github.com/socktainer/socktainer/pull/507).
6. **socktainer accepts any API version in the path.** `/v1.99/version` answers 200, where
   Docker Engine returns 400. Low priority; file only if it matters to a client.
7. **Copying a directory into a never-started container fails** — found 2026-11-02; someone
   else filed it 2026-11-04 as
   [socktainer#502](https://github.com/socktainer/socktainer/issues/502) (open). Upstream #9045 copies the mkcert CA into web between `compose create` and `start`;
   socktainer refuses the archive's directory entry ("cannot copy a directory into … before
   it starts"), so web generated its own CA and HTTPS between containers trusted the wrong
   root. Files alone go through, so since 2026-11-03 `caRootArchive` leaves out the
   directory entry on Apple Container (the image has the directory). Verified on `d11`: the
   CA lands as the web user, and HTTPS from web to the router answers 200.
8. **The exec-path crash seen once** (see "Smaller gaps") — file only if it recurs.
9. **Bind mounts of host sockets named `*.sock` are dropped** — filed 2026-11-03 as
   [socktainer#495](https://github.com/socktainer/socktainer/issues/495). The runtime relays
   a socket bind mount into the guest, but `ContainerCreateRoute` skips sources by name, so
   1Password's `…/agent.sock` never arrives while the same socket named `agent` works. Repro
   `handoff/sockbindrepro/repro.sh`. Fix in PR
   [#496](https://github.com/socktainer/socktainer/pull/496) (branch `fix/socket-bind-mounts`,
   worktree `~/tmp/socktainer-sockbind`); three rounds of CodeRabbit comments addressed, the
   last (relative socket sources in `Mounts`) answered rather than changed, since Docker
   Engine rejects relative bind sources there. Merged 2026-11-03, released in v1.6.1. Closed
   [#265](https://github.com/socktainer/socktainer/pull/265) tried a narrower SSH-only
   approach.
10. **Volume `nocopy` is ignored** — filed 2026-11-03 as
    [socktainer#497](https://github.com/socktainer/socktainer/issues/497). socktainer copies
    image content into every empty ext4 named volume; DDEV's `project_mutagen` volume (mounted
    with `nocopy`) gets the web image's `/var/www/html`, which Mutagen then syncs into the
    user's project (`index.nginx-debian.html`). The Binds form `-v vol:/x:nocopy` instead
    hangs container start. Repro `handoff/nocopyrepro/repro.sh`. DDEV no longer depends on
    it: [ddev#9061](https://github.com/ddev/ddev/pull/9061) empties `/var/www/html` in the
    web image.
11. **Network names over 63 characters, or with uppercase, fail** with a bare 500 —
    [socktainer#498](https://github.com/socktainer/socktainer/issues/498). The limit is
    apple/container's network ID validation; socktainer passes the Docker name through.
    Compose names networks `<project>_default`, so long project names hit it. Repro
    `handoff/netnamerepro/repro.sh`.
12. **`docker exec -w <missing>` creates the directory** —
    [socktainer#499](https://github.com/socktainer/socktainer/issues/499); Docker fails with
    exit 127. The `mkdir` is in apple/containerization's `vmexec`. Repro
    `handoff/execworkdirrepro/repro.sh`.
13. **A container's `Hostname` doesn't resolve from peers** on a user-defined network —
    filed 2026-11-06 as [socktainer#510](https://github.com/socktainer/socktainer/issues/510),
    no PR yet. DDEV doesn't depend on it.
14. **Concurrent image loads lose a tag** — filed 2026-11-11 as
    [socktainer#518](https://github.com/socktainer/socktainer/issues/518); see "socktainer v1.6.1" above.
    Repros in `handoff/inlinerecreaterepro/`. Open the PR (shared `ImageStore`) once #513 or
    #515 lands. Issue text in `~/tmp/sk-upstream/concurrent-load-issue.md`.

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

**What fails on socktainer v1.6.0 (fixed in v1.6.1):** anything that makes Compose recreate a container
while the old one exists — `ddev snapshot restore` ("hostname(s) already exist:
["appletest-web", "appletest-web"]"), and `ddev start` over a running project whose config
changed. See "Hostname uniqueness" below. `ddev restart` recovers from both. Both work on
socktainer v1.6.1.

Merged DDEV-side hardenings from this work: [#8657](https://github.com/ddev/ddev/pull/8657),
[#8658](https://github.com/ddev/ddev/pull/8658), [#8659](https://github.com/ddev/ddev/pull/8659),
[#8928](https://github.com/ddev/ddev/pull/8928), and upstream's split of the global cache
for [#8895](https://github.com/ddev/ddev/issues/8895):
[#8945](https://github.com/ddev/ddev/pull/8945) (router's own `ddev-router-traefik`),
[#9024](https://github.com/ddev/ddev/pull/9024) (per-project web and db cache volumes),
[#8956](https://github.com/ddev/ddev/pull/8956) (global commands copied into each container),
[#9041](https://github.com/ddev/ddev/pull/9041) (start-chown skips volumes a running container
holds, on every provider),
[#9042](https://github.com/ddev/ddev/pull/9042) (router files cleaned through the running router),
[#9045](https://github.com/ddev/ddev/pull/9045) (mkcert CA copied into web, so web and db no
longer mount `ddev-global-cache`), and
[#9040](https://github.com/ddev/ddev/pull/9040) (image labels re-checked client-side).
Draft: [#8656](https://github.com/ddev/ddev/pull/8656), this branch.

Also merged: [#9046](https://github.com/ddev/ddev/pull/9046) (single-writer volume test) and
[#9061](https://github.com/ddev/ddev/pull/9061) (empty `/var/www/html` in the web image).
[#9044](https://github.com/ddev/ddev/pull/9044) (seed
`ddev-router-traefik` before web) was closed unmerged.

### What still has to be done by hand

- For SSH keys in web, `ddev config global --ssh-agent-upstream=host` (or a socket path) —
  see "ssh-agent" below.
- `ddev config global --router-http-port=8080 --router-https-port=8443`. The **global**
  setting matters: with 80/443 there, the router fails with "listen tcp :80: bind:
  permission denied" even when the project sets 8080/8443.
- A `traefik_monitor_port` other than OrbStack's 10999 (11999 here), when another provider's
  DDEV router might also be running.

Mutagen works and is the default, as on any macOS provider (verified 2026-11-02: sync both ways,
`upload_dirs` as a virtiofs mount inside the `project_mutagen` block volume, restart, stop/start,
and `TestMutagenConfigChange`, `TestMutagenDiagnose` and `TestCmdMutagen` passing). The branch
used to force it off because web mounted `project_mutagen` twice; upstream #8944 made that one mount.

## DDEV changes on the branch

After upstream's #8895 work, including #9045, no named volume needs a stand-in: web and db
don't mount `ddev-global-cache`, and the router volume and the per-project cache volumes each
have one writer. Only short-lived helpers (start-chown, the mkcert copy when no router runs,
the router's one-time seed) and add-ons still mount `ddev-global-cache`, one at a time.

- `pkg/dockerutil/providers.go` — `IsSocktainer()`, `IsAppleContainer()`,
  `GetSocktainerVersion()`.
- `pkg/dockerutil/requirements.go`, `versionconstants.go` — warn on `ddev start` and
  `ddev utility dockercheck` when socktainer is older than `SocktainerMinVersion` (1.6.1, the
  release with socktainer#492, #494, #496 and #509). The version comes from
  the `socktainer` server component; source builds report `unspecified` or `0.0.0-dev` and
  aren't compared.
- `pkg/ddevapp/compose_build.go` — `composeBuild` builds one service at a time on this
  provider, since concurrent image loads lose a tag (upstream item 14).
- `pkg/dockerutil/containers.go` — `CopyIntoContainer` extracts with `tar` through
  `ExecWithStdin` on this provider, working around the volume copy bug above.
- `ddevapp.go` (`GetOmittedContainers`, a `chmod` in `Start`), `ssh_auth.go`
  (`AppleContainerSSHAgentSocket`, `host` allowed), `config.go`, `app_compose_template.yaml`,
  `auth-ssh.go` — web uses the host's SSH agent directly; see "ssh-agent" below.
- `environment.go`, `version.go`, `utility-port-diagnose.go`, `addons.go` — report the
  provider as `apple-container`, and give add-on installs the virtiofs propagation delay.

Dropped in the 2026-10-31 rebase, as upstream made them unnecessary: the router's bind-mounted
Traefik directory, `NotifyRouterOfTraefikConfigChange()`, `mergeCopyDir()` and the
host-side list/remove helpers, and the global-commands copy into the cache. Dropped in the
2026-11-02 rebase, now upstream as #9041 and #9042: the branch's start-chown skip
(`isServiceRunning`) and router cleanup (`runInRouterTraefikVolume`). Dropped after the
second 2026-11-02 rebase, onto #9045: the bind-mounted global cache (`UseBindGlobalCache()`,
`GlobalCacheSource()`, `GlobalCacheMount()`, `DDEV_BIND_GLOBAL_CACHE`, `copyIntoGlobalCache`,
and the `BindGlobalCache` and `GlobalCacheMount` template variables); `pkg/ddevapp/ddevapp.go`,
`config.go` and `app_compose_template.yaml` now match upstream. Verified 2026-11-02 with
`ddev restart` of `appletest`: starts, 200 over HTTPS on 8443, no `ddev-global-cache` mount
in web; the CA copy into web fails as in item 7 above. Dropped in the 2026-11-03 rebase, now
upstream as #9040: `FindImagesByLabels`'s client-side label re-check.

## Open blockers

### Hostname uniqueness collides with Compose's recreate — fixed in socktainer v1.6.1

Fixed by [socktainer#509](https://github.com/socktainer/socktainer/pull/509), merged
2026-11-09 and released in v1.6.1. On `main` as of 2026-11-09, `ddev snapshot restore` and `ddev
start` over a running project with changed config both work. The history below can go once
`SocktainerMinVersion` is 1.6.1.

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

The uniqueness rule is apple/container's `AttachmentNameValidator`, and it is deliberate.
Apple Container also has no guest hostname apart from the attachment's: it takes the
attachment hostname up to the first dot. So the fix in
[socktainer#509](https://github.com/socktainer/socktainer/pull/509) (issue
[#508](https://github.com/socktainer/socktainer/issues/508)) gives each attachment
`<hostname>.<random suffix>`. The guest still sees the requested hostname, and
`ddev snapshot restore` completes with it. socktainer was also reporting the hostname error
as a 409 about the container name, which #509 fixes as well. Upstream's open socktainer#487
(a mode resolving names with apple/container's DNS) would see the suffixed names.

Seen again 2026-11-03 on `d11`: after `ddev config global --ssh-agent-upstream=…` changed
web's mounts, `ddev restart` and a `ddev poweroff`/`ddev start` in quick succession both failed
with `create?name=ac5755627b26_ddev-d11-web` (Compose's recreate name), so Compose was
recreating running containers. A later clean `ddev stop` plus `ddev start`, and a `ddev
restart` after another global change, both worked. The exact trigger wasn't pinned down.

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

Where it still hits DDEV after #8895 and #9045: add-ons whose services mount
`ddev-global-cache`, if two of them run at once. (`ddev-ssh-agent_socket_dir` was the other;
see "ssh-agent" below.) Helpers skip a volume a running
container holds (upstream #9041).

### ssh-agent: web uses the host's agent directly (resolved 2026-11-03)

`ddev-ssh-agent` can't serve web here: web can't attach the socket volume ssh-agent holds
(`VZErrorDomain Code=2` on `ddev start`), and a Unix socket inside another VM is unreachable
anyway. But Apple Container relays a bind-mounted **host** socket into a container's VM over
vsock (what `container run --ssh` uses). So on Apple Container the branch:

- always treats `ddev-ssh-agent` as omitted (`GetOmittedContainers`), so a default project
  starts;
- with `ssh_agent_upstream` set (`host` uses `$SSH_AUTH_SOCK`), bind-mounts that host socket
  at `/home/.ssh-agent/socket` in web (`AppleContainerSSHAgentSocket`), and `chmod 0666`s it
  as root after start: the relayed socket is root's, with the host socket's mode (1Password's
  is 0600);
- makes `ddev auth ssh` explain this instead of adding keys to an agent web can't reach.

Verified 2026-11-03 on `d11` and by `TestAppleContainerSSHAgent`: web lists the host agent's
keys. socktainer 1.6.0 dropped a socket named `*.sock`, such as 1Password's (#495,
fixed by #496 in v1.6.1); DDEV's check and the socat relay advice went 2026-11-11. A symlink to the
socket hangs container start.
DDEV's own agent (`ddev auth ssh` with key files) has no equivalent; it would need a TCP
transport. ssh in web also can't save `known_hosts`, since `/home/.ssh-agent` is now a
root-owned mount point rather than the volume.

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
- `handoff/sockbindrepro/repro.sh` — a host socket bind-mounted as `echo` and as `echo.sock`
  (socktainer#495).
- `handoff/cpvolrepro/repro.sh` — `docker cp` of a directory into a running container, at a
  volume mount and on the rootfs.
- `handoff/cpmetarepro/repro.sh` — owner and mtime of files after `docker cp` into a running
  container (socktainer#514).
- `handoff/inlinerecreaterepro/concurrent-load.sh` — two `docker load`s at once leave a tag
  on its old image (`SERIAL=1` for the control); `repro.sh` there shows the effect through
  `docker compose build` and `up`; `imageid.sh` checks image-ID stability.
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

## Current machine state (2026-11-11)

- Apple `container` **1.6.1** from the signed installer at `/usr/local`
  (`container-apiserver` running). Homebrew's `container` is 1.7.1, and Homebrew's socktainer 1.6.1 is installed and first on `PATH` (2026-11-11).
- socktainer running is Homebrew's 1.6.1 (`/opt/homebrew/bin/socktainer --no-auto-start`,
  started 2026-11-11). The #515 build is in
  `~/workspace/socktainer-worktrees/archive-owner-mtime/.build/release/socktainer`.
  The #513 build is in `~/workspace/socktainer-worktrees/network-connect-aliases`.
  `~/workspace/socktainer/.build/release/socktainer` is stock `main` of the same day.
  Log: `~/tmp/socktainer.log`. Start `container system start --disable-kernel-install`;
  without the flag it waits on a prompt.
- Docker context **`socktainer`**.
- Global router ports 8080/8443, `traefik_monitor_port` 11999.
- `appletest` and `appletest2` (`~/tmp/`) powered off, both served 200 when last up; both have
  `omit_containers: [ddev-ssh-agent]`. `appletest3` untested; `appledel` was a throwaway,
  deleted (its `.ddev` remains in `~/tmp/appledel`).
- The `ddev-global-cache` block volume on socktainer is used again now that the bind mount is
  gone; `~/.ddev/global-cache-bind` is no longer read and can be deleted.
  `ensureRouterTraefikVolume` seeded `ddev-router-traefik` from the volume once.
- The root Homebrew dnsmasq service is stopped.

### Repositories and worktrees

socktainer, main checkout `~/workspace/socktainer` (remotes `upstream`, `rfay`), on
`tmp/upstream-main-test` (local only, fast-forwarded to upstream `main`). The #505 fix is in
worktree `~/workspace/socktainer-worktrees/network-connect-aliases`, branch
`fix/network-connect-aliases`. Worktree
`~/tmp/socktainer-imgfilter` holds `fix/image-list-label-filter`, pushed to `rfay` for
upstream PR #492. The worktrees `~/tmp/socktainer-hcfix-timeout` (#456) and `~/tmp/socktainer-portbindings` (#458) are
merged upstream and can be removed, as can
`~/workspace/socktainer-worktrees/version-engine-version` (#459 closed). Drafts and posted
text for every upstream issue, PR and fork comment are in `~/tmp/sk-upstream/`.

DDEV worktree: this branch is in
`~/workspace/ddev-worktrees/20260802_rfay_apple_container_experiment`.
