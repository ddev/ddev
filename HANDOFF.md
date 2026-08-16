# HANDOFF — PR #8707 review fixes (#8609 phase 2)

Temporary file. Delete before merging.

Working tree on branch `20260814_rfay_docker_update_phase_2`, uncommitted.
Nothing has been pushed.

## What changed and why

### 1. `wait-for-images.sh` waited for a tag nothing ever pushes — blocking

It computed `<current-branch>-<hash>`, but the tag `ddev` actually pulls is the
one committed in `versionconstants.go`. `autotag.sh` rewrites that line only
when the *hash* changes, and when it does it rewrites the whole tag including
the branch prefix — so the two agree only on a branch that changed the image.

Every test runner calls this script unconditionally, while
`image-build-push.yml` only triggers on `containers/**`. So any PR not touching
`containers/` would poll 20 minutes and fail every Buildkite and GitHub test
job. Reproduced on this branch before the fix:

```text
versionconstants.go WebTag  = 20260721_rfay_content_addressed_image_tags-36bceca65e   → EXISTS
wait-for-images.sh computed = 20260814_rfay_docker_update_phase_2-36bceca65e          → MISSING
```

Fix: new `containers/required-image-tag.sh` resolves the tag the same way
`autotag.sh` does and reports which case applies:

* `committed <tag>` — hash still matches, so that exact tag (stale branch
  prefix and all) is what gets pulled and what must exist in the registry.
* `recomputed <tag>` — content changed, so `make` builds it locally on the
  runner and there is nothing to wait for.

`wait-for-images.sh` no longer needs a branch name at all, which removed the
`WAIT_FOR_IMAGES_BRANCH` plumbing from four callers.

### 2. `detect` rebuilt and re-pushed unchanged images

Same root cause: `detect` checked `<branch>-<hash>`, so any PR touching
`containers/` re-pushed all five images under a fresh branch-prefixed tag even
with no image change — and, on a fork, asked a maintainer to approve that
no-op. It now uses `required-image-tag.sh` too. Verified: `detect` on this PR
now yields `matrix=[]`.

### 3. Script injection via `github.head_ref` — security

`BRANCH="${{ github.head_ref || github.ref_name }}"` spliced attacker-controlled
text into a `run:` block (git ref names permit `"`, backtick, `$`, `;`).
`actionlint` flagged it independently. `detect` has no secrets itself, but it
emits `is_fork`; injected code could set `is_fork=false` and route fork content
into `build-and-push`, the job that loads `PUSH_SERVICE_ACCOUNT_TOKEN`.

Fixed by passing it through `env:`. `is_fork` also moved into its own step, so
nothing the per-image loop does can reach the output that decides whether the
push secret loads. The `WAIT_FOR_IMAGES_BRANCH` removal in (1) also deleted a
PowerShell/bash splice of the same value in `test-wsl2-reusable.yml`.

### 4. `image-push.yml` validated the tag but not the repository — security

`repos.txt` comes straight out of the fork-produced artifact and was pushed to
verbatim, so an approved fork build could publish to any repo the credential
can write. New `containers/validate-image-repo.sh` enforces
`$DOCKER_ORG/<known-suffix>` (with a pattern for `ddev-dbserver-<engine>-<ver>`).

### 5. Silent-failure paths in `image-push.yml`

* Added `actions: read` — `download-artifact@v8` needs it to reach another
  run's artifacts, and `continue-on-error: true` was masking that as
  "nothing needed pushing".
* New ungated `check-artifacts` job lists artifacts via the API and gates the
  environment job, so a fork PR with nothing to push no longer requests an
  approval. Approving something that turns out to be a no-op trains people to
  click without looking.
* Removed `continue-on-error` from the download; an empty push summary now
  fails the job instead of commenting success.
* Artifact `retention-days` 1 → 7. The gate is a human approval that may not
  come the same day.

### 6. `DDEV_IMAGE_TAG` not passed to the builds

`push-tagged-image.yml` passes it; the new jobs didn't, so
`com.ddev.image-tag` was baked as `<tag>-amd64` instead of `<tag>`. That label
is what `imageVersionMismatch()` in `pkg/ddevapp/config_custom.go` compares
against, so pinned-image users would have seen spurious mismatch notes.

### 7. Smaller items

* `DOCKER_ORG` now falls back to `ddev` in both workflows (was empty on a repo
  without the variable, producing `/ddev-webserver`).
* `validate-image-tag.sh`: the reserved-literal and `vX.Y.Z` checks were
  unreachable — nothing that reaches them can match. They now test the part
  before the hash, so `latest-0123456789` and `v1.2.3-0123456789` are rejected.
  Also requires a leading character Docker accepts.
* New `containers/image-configs.sh` is the single source for the image list,
  sourced by both `wait-for-images.sh` and `detect` (was duplicated, with
  "keep in sync" comments).
* BSD `wc -l` padding broke 4 checks in `wait_for_images_test.sh` and 1 in
  `autotag_test.sh` on macOS. Fixed in both.

## Test status

All 73 checks pass locally (macOS) and are wired into `container-tests.yml`:

| Harness | Checks |
| --- | --- |
| `containers/autotag_test.sh` | 17 |
| `containers/required_image_tag_test.sh` | 7 (new) |
| `containers/registry_tag_exists_test.sh` | 4 |
| `containers/validate_image_tag_test.sh` | 15 |
| `containers/validate_image_repo_test.sh` | 16 (new) |
| `containers/wait_for_images_test.sh` | 14 |

`shellcheck -x` clean on all new/changed scripts. `actionlint` reports no
untrusted-input findings; the remaining SC2086/SC2046 notes in
`test-reusable.yml` are pre-existing and untouched.

## Verification Claude can do without credentials

These run against the real registry (read-only, anonymous) and this checkout.
None of them push, and none need Docker running —
`docker buildx imagetools inspect` talks to the registry directly (verified
with `DOCKER_HOST` pointed at a dead socket).

1. **Unit harnesses** — `for t in containers/*_test.sh; do $t; done`.
2. **The blocking regression** — `WAIT_FOR_IMAGES_ATTEMPTS=1
   containers/wait-for-images.sh` must find all five tags and exit 0. Before
   the fix it failed on the first image.
3. **`detect` dry run** — source `containers/image-configs.sh`, loop
   `required-image-tag.sh` + `registry-tag-exists.sh`, confirm `matrix=[]` on a
   PR that changes no image content.
4. **Changed-image path** — append a line to `containers/ddev-xhgui/Dockerfile`,
   re-run (3): only `ddev-xhgui` should say `BUILD ... (recomputed)`, and
   `wait-for-images.sh` should skip it with "not waiting" while still finding
   the other four. `git checkout` the file afterwards.
5. **Injection** — run (4) with `REQUIRED_IMAGE_TAG_BRANCH='evil"; id; #'`.
   Expect the tag `evil-id--<hash>` and no command execution. (Done: passes.)
6. **Artifact round-trip against a local registry** — not yet done, and the
   most valuable thing left that needs no secrets. Run `registry:2` in a
   container, `docker save` a small image the way the `build` job does, write
   `repos.txt`/`tag.txt`/`arch.txt`, then run `image-push.yml`'s load/validate/
   push loop against `localhost:5000`. Feed it a hostile `repos.txt`
   (`ddev/ddev-webserver`, `attacker/evil`) and confirm
   `validate-image-repo.sh` stops it before any push. Exercises the multi-arch
   `imagetools create` grouping logic, which no unit test covers.
7. **`make` still builds** — `make` at the repo root, confirm `autotag-images`
   no-ops and `versionconstants.go` is untouched.

Items 1–5 have been run and pass. 6 and 7 have not.

## Verification only a human can do

Everything below needs `ddev-test/ddev` with the `image-push` environment and
`PUSH_SERVICE_ACCOUNT_TOKEN` configured. Do not run these against `ddev/ddev`.

1. **Go-only PR.** A PR touching no `containers/` file. Every test job should
   reach "Wait for pushed images", print five `found …` lines within seconds,
   and continue. This is the fix for the blocking bug and nothing in CI has
   ever exercised it — every commit on this branch carries `[skip ci]`, so the
   four green Buildkite checks on #8707 either skipped or predate the change.
2. **No-op `containers/` PR.** Add a file under `containers/` that isn't in any
   hash path. `detect` should report five `already exists (committed)` lines
   and build nothing.
3. **Real image change, same-repo branch.** Edit
   `containers/ddev-xhgui/Dockerfile`, run `make`, commit the
   `versionconstants.go` change. Expect: `detect` lists only xhgui →
   `build-and-push` runs both arches with no approval → `create-manifests`
   comments → `imagetools inspect` shows both platforms and the per-arch tags
   are gone. Then confirm the `com.ddev.image-tag` label reads `<tag>`, not
   `<tag>-amd64` (item 6 above).
4. **Real image change, fork branch.** Same edit from a fork. Confirm the
   `build` job shows no secret-loading step, that `check-artifacts` finds the
   artifacts, that `image-push` requests approval once, and that the download
   succeeds — this is the path where the missing `actions: read` would have
   shown up as a false "nothing needed pushing".
5. **Fork PR touching `containers/` with no image change.** Confirm *no*
   approval request appears (previously it always did).
6. **Adversarial artifact.** On a fork branch, add a step overwriting
   `repos.txt` with `ddev/ddev-webserver` before upload. Approve, and confirm
   the push job fails at `validate-image-repo.sh` rather than publishing.
7. **Hostile branch name.** Push a fork branch literally named
   ``test`touch /tmp/pwned` `` and read the `detect` job log. The branch should
   appear only as sanitized data.
8. **Expired artifact.** Trigger a fork build, wait past `retention-days`, then
   approve. The run must fail loudly.

## Still open

* **The PR description is stale.** It still describes "an `approval` job gates
  on a new `image-push` GitHub Environment before any expensive/untrusted build
  work runs", which commits a7ec6b5c9 / 2bdced7ef removed. The in-repo docs are
  correct. Needs a maintainer edit.
* **`actions: read` on `download-artifact@v8`** is added on the documented
  requirement; it has not been observed failing or passing on a live run.
  Item 4 above confirms it.
* **Registry pollution has no cleanup path.** Each image change adds a
  multi-arch tag that is never removed. Fine for now; worth a follow-up issue
  alongside the `TODO(#8609)` about the 18 unbuilt `ddev-dbserver` variants.
* **`registry-tag-exists.sh` cannot distinguish "missing" from "registry
  unreachable."** A DockerHub blip costs a redundant rebuild in `detect`, or 20
  minutes and a red build in `wait-for-images.sh`. Acceptable, but it's the
  most likely source of a confusing intermittent failure.
* **Commits.** None made — the fixes are uncommitted in the working tree, and
  the branch still has one unpushed local commit (`77d90bd97`, empty).
