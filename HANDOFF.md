# HANDOFF — PR #8707 review fixes (#8609 phase 2)

Temporary file. Delete before merging.

Branch `20260814_rfay_docker_update_phase_2`. Nothing has been pushed.

## MUST TEST BEFORE MERGE

### 0. Every image has to be republished at its bare-hash tag

Tags are now bare content hashes (`ddev/ddev-webserver:36bceca65e`), which
invalidates every previously published content-hash tag. Nothing in either
organization carries the new tags yet — confirmed missing for webserver,
ssh-agent, and the db variants. **Integration tests cannot pull until the
first CI run publishes them**, which is 44 build jobs across 24 images.
Locally, `go test -run TestCmdVersion ./cmd/ddev/cmd/...` currently fails with
`manifest for ddev/ddev-ssh-agent:8e8bf1217c not found`, which is this and
nothing else.

This is a one-time migration cost of the scheme change; after that first push
the tags stop depending on branch or organization, which is the whole point.

Two further things have never run in CI and no test harness covers. Both are
cheap to get wrong and expensive to discover after merge.

### 1. A `containers/ddev-dbserver` change must build and push all 20 variants

This is the bug that motivated the second round of fixes, and this PR is
itself the first live test of the fix — adding `variants.txt` under
`containers/ddev-dbserver/` changed that directory's hash, so `BaseDBTag` is
now `20260814_rfay_docker_update_phase_2-1dc90407ef` and **CI must build and
push 36 jobs across 20 repositories before any test using a non-default
database can pass.**

Watch for, on this PR's first real CI run:

- `detect` reports 36 build jobs / 20 images, not 1.
- `create-manifests` (or `image-push`) produces a multi-arch manifest for
  every one of the 20 `ddev/ddev-dbserver-*` repos at that tag, with the four
  oldest (`mariadb-5.5`, `mariadb-10.0`, `mysql-5.5`, `mysql-5.6`) amd64-only.
- `TestDdevAllDatabases` passes, along with the tests in `db_test.go`,
  `snapshot_test.go`, `config_test.go`, and `debug-migrate-database_test.go`
  that pin a non-default database.

Verify independently of CI's own reporting:

```bash
TAG=$(grep -E '^var BaseDBTag' pkg/versionconstants/versionconstants.go | sed -E 's/.*"([^"]*)".*/\1/')
for r in $(containers/ddev-dbserver/variants.sh repos); do
  printf '%-34s ' "$r"; containers/registry-tag-exists.sh "ddev/$r" "$TAG" && echo EXISTS || echo MISSING
done
```

Every line must say EXISTS. Any MISSING means the matrix regressed and
non-default database tests will fail.

Note that item 1 below was written before the tag scheme changed: the tag to
check is now the bare `BaseDBTag` hash, not a branch-prefixed string, and the
`variants.sh repos` loop still works unchanged.

### 2. `push-tagged-dbimage.yml` still works after the DRY refactor

Its matrix, its `MULTI_ARCH_IMAGES` list, and its multi-arch/single-arch
decision were three separate hardcoded copies of the variant list; all three
now come from `variants.sh`. This is the release-time push path, so a mistake
here surfaces during a release.

Its `tag` input is now optional — left empty it derives the tag the checkout
needs, which is the fix for having to transcribe `main-1dc90407ef` by hand.
Test **both** paths: empty (derives the bare hash) and an explicit `vX.Y.Z`
(the release path).

Run it manually on `ddev-test/ddev` and confirm:

- The `variants` job runs first and its matrix expands to **36** `build-db-arch`
  jobs — the same count as before (20 variants × 2 arches, minus 4 arm64
  exclusions).
- With an empty tag input, the resolve step logs the derived tag and it matches
  `BaseDBTag` in `versionconstants.go`.
- The four amd64-only variants get `multi_arch=false` in their `meta` step and
  push an unsuffixed tag; the other 16 get `multi_arch=true` and push
  `-amd64`/`-arm64`.
- `create-manifests` combines exactly the 16 multi-arch variants and deletes
  the intermediary per-arch tags.
- All 20 repos carry the throwaway tag at the end.

Locally I confirmed the generated lists are byte-identical to the ones they
replaced (`build-targets` for both host arches, `single-arch-targets`,
`test-targets` for both, and the `MULTI_ARCH_IMAGES` set), and that
`make -n` still resolves a target from each list including the amd64-only
ones under `CURRENT_ARCH=amd64`. That is static equivalence, not a live run.

## Round 1 fixes — review findings

### The tag-resolution bug (blocking)

`wait-for-images.sh` computed `<current-branch>-<hash>`, but the tag ddev
pulls is the one committed in `versionconstants.go`, and `autotag.sh` rewrites
that line (branch prefix included) only when the hash changes. The two agree
only on a branch that changed the image, so any PR not touching `containers/`
would poll 20 minutes and fail every Buildkite and GitHub test job. `detect`
had the mirror-image bug: it re-pushed all images under a fresh branch prefix
on any `containers/` change, no-op or not.

`containers/required-image-tag.sh` now resolves the tag once, for both
callers, reporting `committed` (hash matches — wait for that exact tag) or
`recomputed` (content changed — `make` builds it locally). This also removed
the `WAIT_FOR_IMAGES_BRANCH` plumbing from four callers, since the branch name
is no longer needed.

### Security

- `github.head_ref` reached a `run:` block spliced into the script. Git ref
  names permit quotes and backticks, and `detect` emits `is_fork`, so injected
  code could set `is_fork=false` and route fork content into `build-and-push`,
  the job that loads `PUSH_SERVICE_ACCOUNT_TOKEN`. It now arrives via `env:`,
  and `is_fork` moved to its own step.
- `image-push.yml` validated the tag but pushed to whatever repository names
  the fork-produced artifact listed. `validate-image-repo.sh` constrains them
  to `$DOCKER_ORG` plus an exact allowlist.

### Silent failures

- `image-push.yml` lacked `actions: read` for a cross-run artifact download,
  and `continue-on-error` turned that into a "nothing needed pushing" comment.
  A new ungated `check-artifacts` job gates the environment job instead, so a
  fork build with nothing to push no longer asks for approval, and a failed
  download is now fatal.
- Artifact retention 1 → 7 days; the gate is a human approval.

### Smaller

- `DDEV_IMAGE_TAG` was not passed, so `com.ddev.image-tag` recorded
  `<tag>-<arch>` rather than the tag people pull, which
  `imageVersionMismatch()` compares against.
- `DOCKER_ORG` falls back to `ddev` instead of producing `/ddev-webserver`.
- `validate-image-tag.sh`'s reserved-literal and `vX.Y.Z` checks were
  unreachable behind the format check; they now test the part before the hash.
- BSD `wc -l` padding failed 5 checks on macOS.

## Round 2 — the db variant matrix

`GetDBImage()` in `pkg/docker/images.go` builds every variant's reference from
one shared `BaseDBTag`, so a dbserver change moves the tag for all 20 while
`make` built only `mariadb_11.8` and CI pushed only `ddev-dbserver-mariadb-11.8`.
The other 19 were referenced at a tag that existed nowhere. Introduced by
phase 1 (#8612): before that, `BaseDBTag` was hand-bumped after someone ran
`push-tagged-dbimage.yml` for all 20.

Fixed by making `detect`'s matrix cover every variant. The build matrix is now
one entry per (image, arch) rather than a cross product, because the oldest
variants are amd64-only, and `create-manifests` takes its arch list from
`detect` instead of assuming both. Artifact names key on `repo_suffix`, not
`make_dir` — all 20 db variants share a `make_dir` and would have collided.

`wait-for-images.sh` now also fails fast, with the command to run, when a
non-locally-built image is out of date. There is no local fallback for those
19 variants, and the tag `make` would invent depends on the runner's branch
name (detached HEAD on a PR checkout), so it may not match what CI pushed.

### DRY

The variant list was duplicated in four places. It now lives in
`containers/ddev-dbserver/variants.txt`, read through `variants.sh`, which
renders each consumer's view:

| Consumer | View |
| --- | --- |
| `containers/ddev-dbserver/Makefile` | `build-targets`, `single-arch-targets`, `test-targets` |
| `containers/image-configs.sh` | `list` |
| `containers/validate-image-repo.sh` | `repos` |
| `.github/workflows/push-tagged-dbimage.yml` | `json`, `multi-arch-variants` |

`variants.txt` sits inside the hashed dbserver directory on purpose: adding a
database version has to change the content hash, or `detect` would decide the
tag already exists and never build the new variant.

## Test status

127 checks across seven harnesses, all passing locally (macOS), all wired into
`container-tests.yml`:

| Harness | Checks |
| --- | --- |
| `containers/autotag_test.sh` | 18 |
| `containers/db_variants_test.sh` | 15 (new) |
| `containers/registry_tag_exists_test.sh` | 4 |
| `containers/required_image_tag_test.sh` | 12 (new) |
| `containers/validate_image_repo_test.sh` | 37 (new) |
| `containers/validate_image_tag_test.sh` | 20 |
| `containers/wait_for_images_test.sh` | 21 |

`shellcheck -x` clean on all new and changed scripts. `actionlint` clean on
all changed workflows apart from pre-existing SC2086/SC2046 notes in untouched
parts of `test-reusable.yml`.

## Verification Claude ran

Read-only against the real registry, plus this checkout. Nothing pushed.

1. All seven harnesses.
2. `wait-for-images.sh` finds the four non-db images and correctly waits on the
   db variants at the new `BaseDBTag`.
3. `detect` dry run: `matrix=[]` when nothing changed; 36 build jobs / 20
   manifests after a dbserver change; only xhgui after an xhgui change.
4. Hostile branch name `evil"; id; #` sanitizes to `evil-id-`, no execution.
5. Generated db lists byte-identical to the four hardcoded copies they replace;
   `make -n` resolves a target from each list on both host arches.
6. `make autotag-images` built `mariadb_11.8` locally and rewrote `BaseDBTag`.
7. `docker buildx imagetools inspect` works with no Docker daemon running,
   so the early placement of the wait step in the Buildkite scripts is fine.

Not run: the artifact round-trip against a local `registry:2` (worth doing —
it needs no secrets and would exercise the multi-arch `imagetools create`
grouping that no unit test covers), and a full `make` build.

## Other human verification

On `ddev-test/ddev`, with the `image-push` environment configured. Confirm the
rule actually took: `gh api repos/<owner>/<repo>/environments/image-push` — an
environment referenced before it exists is auto-created with no protection.

1. **A PR touching no `containers/` file.** Five `found …` lines within seconds,
   then the tests run. This is the round-1 blocking fix and has never run in CI.
2. **A `containers/` file that isn't in any hash path.** `detect` reports
   `already exists (committed)` for everything and builds nothing.
3. **An `ddev-xhgui` change on a same-repo branch.** Only xhgui builds; no
   approval; `com.ddev.image-tag` on the result is the final tag, not
   `<tag>-amd64`.
4. **The same from a fork.** No secret-loading step in `build`; exactly one
   approval; the download succeeds rather than silently commenting
   "nothing needed pushing".
5. **A fork PR touching `containers/` with no image change.** No approval
   request at all.
6. **Adversarial artifact.** Overwrite `repos.txt` with `ddev/ddev-webserver`
   before upload; the push must fail at `validate-image-repo.sh`.
7. **Hostile branch name** ``test`touch /tmp/pwned` ``; check the `detect` log.
8. **Expired artifact.** Approve after `retention-days`; must fail loudly.

## Round 3 — hash-only tags

The branch prefix in a tag carried no information but forced every consumer to
agree on a string. Tags are now the bare content hash; the branch survives as a
trailing comment in `versionconstants.go`, a `<Name>TagBranch` variable shown by
`ddev version` (`image-tag-branches`, collapsed when all images share a
branch), and a `<branch>-<hash>` alias published off the same manifest.

`validate-image-tag.sh` accepts both forms, and the alias goes through it, so a
fork branch named `v1.25.0` can't publish something that reads as a release —
which finally makes the reserved/release checks load-bearing rather than dead.

Comparison against `versionconstants.go` is exact now, so a line still in the
old form counts as stale and `make` migrates it. `wait-for-images.sh` lost its
fail-fast branch: the tag a non-locally-built image needs is branch-independent,
so waiting for it is well defined.

The manual push workflows' `tag` input is optional; empty derives the tag via
`containers/image-tag-for.sh`.

## Still open

- **Fork artifact volume.** A fork changing `ddev-dbserver` now produces 36
  `docker save` tarballs. Local db images are 550–730MB, so that is roughly
  20GB of artifacts for one PR, compressed by `upload-artifact` but still
  large, and now retained 7 days. This is the cost of building the full matrix
  on the untrusted path; if it proves impractical, the fallback is to keep the
  full matrix on the non-fork path and have forks use
  `push-tagged-dbimage.yml`. Watch the first fork-side dbserver PR.
- **`actions: read` on `download-artifact@v8`** is added on the documented
  requirement; not yet observed on a live run.
- **Registry pollution has no cleanup path.** Each image change adds tags that
  are never removed — now 20 at a time for a dbserver change.
- **`registry-tag-exists.sh` cannot distinguish "missing" from "registry
  unreachable."** A DockerHub blip costs a redundant rebuild in `detect` or 20
  minutes and a red build in `wait-for-images.sh`. It now makes 24 registry
  calls per test run instead of 5, so the odds are higher.
- **The PR description** was rewritten for round 1 but does not yet mention the
  db variant matrix.
