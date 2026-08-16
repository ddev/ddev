#!/usr/bin/env bash
# wait-for-images.sh
#
# Neither Buildkite nor the GitHub-hosted test-reusable.yml/
# test-wsl2-reusable.yml runners hold image-push credentials, so any of them
# can race the image-push.yml GitHub Actions workflow: if this commit needs an
# image tag that only just got built, that tag might still be waiting on a
# maintainer's approval when this test run starts. Before running anything
# that pulls a DDEV image, poll the registry for the tags this checkout needs.
#
# Only the tags this checkout will actually *pull* are waited for, which
# required-image-tag.sh resolves the same way autotag.sh does:
#
#   - hash unchanged -> versionconstants.go's committed tag is what gets
#     pulled, so wait for exactly that, branch prefix and all. Recomputing a
#     <this-branch>-<hash> tag here would wait forever on most pull requests,
#     since nothing pushes a tag under a branch that didn't change the image.
#   - hash changed -> `make` rebuilds the image locally on this runner, so
#     there is nothing to wait for.
#
# Fast path (the common case - nothing changed): one registry check per image,
# no wait.
#
# Env:
#   WAIT_FOR_IMAGES_ATTEMPTS - poll attempts before giving up (default 40)
#   WAIT_FOR_IMAGES_SLEEP    - seconds between attempts (default 30)
#
# Defaults give ~20 minutes - ddev-webserver alone takes ~6-8 minutes to build.

set -eu -o pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REGISTRY_TAG_EXISTS="$SCRIPT_DIR/registry-tag-exists.sh"
REQUIRED_IMAGE_TAG="$SCRIPT_DIR/required-image-tag.sh"
DOCKER_ORG="${DOCKER_ORG:-ddev}"

ATTEMPTS="${WAIT_FOR_IMAGES_ATTEMPTS:-40}"
SLEEP_SECONDS="${WAIT_FOR_IMAGES_SLEEP:-30}"

# shellcheck source=containers/image-configs.sh
source "$SCRIPT_DIR/image-configs.sh"

for entry in "${DDEV_IMAGE_CONFIGS[@]}"; do
  IFS='|' read -r repo_suffix tag_var hash_paths _ _ _ _ built_by_make _ <<< "$entry"
  # shellcheck disable=SC2086 # hash_paths is a space-separated path list
  read -r state tag <<< "$("$REQUIRED_IMAGE_TAG" "$tag_var" $hash_paths)"
  image_repo="${DOCKER_ORG}/${repo_suffix}"

  if [ "$state" != "committed" ]; then
    if [ "$built_by_make" = "true" ]; then
      echo "wait-for-images.sh: ${image_repo} content differs from versionconstants.go; make builds ${tag} locally, not waiting"
      continue
    fi
    # No local build to fall back on, and the tag `make` will invent here
    # depends on this runner's branch name (detached HEAD on a PR checkout),
    # so it may not match what image-build-push.yml pushed. Fail now with
    # something actionable instead of timing out on a tag nobody pushed.
    echo "wait-for-images.sh: ${image_repo} content differs from the tag committed in versionconstants.go," >&2
    echo "wait-for-images.sh: and make does not build this image locally." >&2
    echo "wait-for-images.sh: run 'make' and commit the ${tag_var} change in pkg/versionconstants/versionconstants.go." >&2
    exit 1
  fi

  attempt=1
  while true; do
    if "$REGISTRY_TAG_EXISTS" "$image_repo" "$tag"; then
      echo "wait-for-images.sh: found ${image_repo}:${tag}"
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
