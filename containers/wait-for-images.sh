#!/usr/bin/env bash
# wait-for-images.sh
#
# Neither Buildkite nor the GitHub-hosted test-reusable.yml/
# test-wsl2-reusable.yml runners hold image-push credentials, and none of
# them rebuild a changed image locally (autotag.sh's no-op fast path trusts
# the tag already committed in versionconstants.go, so a fresh runner with an
# empty Docker cache won't build it) - so any of them can race the
# image-push.yml GitHub Actions workflow: if this commit's containers/
# changed, the image it needs might still be waiting on a maintainer's
# approval when this test run starts. Before running anything that pulls a
# DDEV image, poll the registry for the tags this checkout actually needs
# and wait for them to land.
#
# The tag is recomputed from real content (branch + hash-paths.sh), the same
# way image-build-push.yml's detect job does it - never read from
# versionconstants.go. That file's committed tag only has its hash kept
# current locally (autotag.sh skips rewriting the branch prefix when the hash
# hasn't changed), so it can carry a stale branch name from whatever branch
# last touched that image, while the registry holds the tag under *this*
# branch's name. Trusting the committed string would then wait forever for a
# tag nothing ever pushed.
#
# Fast path (the common case - nothing changed): one registry check per
# image, no wait.
#
# Env:
#   WAIT_FOR_IMAGES_BRANCH   - branch name to compute tags for (required) -
#                              pass the same value detect uses: for GitHub
#                              Actions that's head_ref || ref_name, for
#                              Buildkite it's $BUILDKITE_BRANCH.
#   WAIT_FOR_IMAGES_ATTEMPTS - poll attempts before giving up (default 40)
#   WAIT_FOR_IMAGES_SLEEP    - seconds between attempts (default 30)
#
# Defaults give ~20 minutes - ddev-webserver alone takes ~6-8 minutes to build.

set -eu -o pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
REGISTRY_TAG_EXISTS="$REPO_ROOT/containers/registry-tag-exists.sh"
HASH_PATHS_SH="$REPO_ROOT/containers/hash-paths.sh"
DOCKER_ORG="${DOCKER_ORG:-ddev}"

ATTEMPTS="${WAIT_FOR_IMAGES_ATTEMPTS:-40}"
SLEEP_SECONDS="${WAIT_FOR_IMAGES_SLEEP:-30}"

BRANCH="${WAIT_FOR_IMAGES_BRANCH:?wait-for-images.sh: WAIT_FOR_IMAGES_BRANCH must be set}"
SANITIZED_BRANCH="$(echo "$BRANCH" | sed -E 's/[^A-Za-z0-9_.-]+/-/g')"

# repo_suffix|hash paths - keep in sync with image-build-push.yml's detect
# job and the Makefile's autotag-images target.
CONFIGS=(
  'ddev-webserver|containers/ddev-webserver containers/containers_shared.mk'
  'ddev-traefik-router|containers/ddev-traefik-router containers/containers_shared.mk'
  'ddev-ssh-agent|containers/ddev-ssh-agent containers/containers_shared.mk'
  'ddev-xhgui|containers/ddev-xhgui containers/containers_shared.mk'
  'ddev-dbserver-mariadb-11.8|containers/ddev-dbserver containers/get_arch.sh'
)

for entry in "${CONFIGS[@]}"; do
  IFS='|' read -r repo_suffix hash_paths <<< "$entry"
  hash="$("$HASH_PATHS_SH" $hash_paths)"
  tag="${SANITIZED_BRANCH}-${hash}"
  image_repo="${DOCKER_ORG}/${repo_suffix}"

  attempt=1
  while true; do
    if "$REGISTRY_TAG_EXISTS" "$image_repo" "$tag"; then
      break
    fi
    if [ "$attempt" -ge "$ATTEMPTS" ]; then
      echo "wait-for-images.sh: gave up waiting for ${image_repo}:${tag} after ${ATTEMPTS} attempts." >&2
      echo "wait-for-images.sh: has the maintainer approved the image-push run for this PR yet?" >&2
      exit 1
    fi
    echo "wait-for-images.sh: ${image_repo}:${tag} not yet available, waiting... (attempt ${attempt}/${ATTEMPTS})"
    sleep "$SLEEP_SECONDS"
    attempt=$((attempt + 1))
  done
done
