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
# Fast path (the common case - nothing changed): one registry check per
# image, no wait.
#
# Env:
#   WAIT_FOR_IMAGES_ATTEMPTS - poll attempts before giving up (default 20)
#   WAIT_FOR_IMAGES_SLEEP    - seconds between attempts (default 15)

set -eu -o pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
REGISTRY_TAG_EXISTS="$REPO_ROOT/containers/registry-tag-exists.sh"
VERSIONCONSTANTS_FILE="${VERSIONCONSTANTS_FILE:-$REPO_ROOT/pkg/versionconstants/versionconstants.go}"
DOCKER_ORG="${DOCKER_ORG:-ddev}"

ATTEMPTS="${WAIT_FOR_IMAGES_ATTEMPTS:-20}"
SLEEP_SECONDS="${WAIT_FOR_IMAGES_SLEEP:-15}"

tag_for() {
  grep -E "^var $1 = " "$VERSIONCONSTANTS_FILE" | sed -E "s/^var $1 = \"([^\"]*)\".*/\\1/"
}

# image-repo:tag-var-name pairs for the images Phase 1's autotag-images
# manages automatically. Keep in sync with Makefile's autotag-images target.
IMAGES=(
  "${DOCKER_ORG}/ddev-webserver:WebTag"
  "${DOCKER_ORG}/ddev-traefik-router:TraefikRouterTag"
  "${DOCKER_ORG}/ddev-ssh-agent:SSHAuthTag"
  "${DOCKER_ORG}/ddev-xhgui:XhguiTag"
  "${DOCKER_ORG}/ddev-dbserver-mariadb-11.8:BaseDBTag"
)

for entry in "${IMAGES[@]}"; do
  image_repo="${entry%%:*}"
  tag_var="${entry##*:}"
  tag="$(tag_for "$tag_var" || true)"
  if [ -z "$tag" ]; then
    echo "wait-for-images.sh: could not find 'var ${tag_var} = \"...\"' in $VERSIONCONSTANTS_FILE" >&2
    exit 1
  fi

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
