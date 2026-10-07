#!/usr/bin/env bash
# registry-tag-exists.sh <image-repo> <tag>
#
# Checks whether <image-repo>:<tag> exists in the registry, without pulling
# it. Exit 0 if it exists, 1 if the registry says it doesn't, 3 if the
# registry still couldn't answer (rate limit, network, 5xx) after 5 tries
# spread over about 75 seconds.

set -eu -o pipefail

if [ "$#" -ne 2 ]; then
  echo "Usage: $0 <image-repo> <tag>" >&2
  exit 2
fi

IMAGE_REPO="$1"
TAG="$2"
ATTEMPTS=5
wait_seconds=5

attempt=1
while true; do
  if output="$(docker buildx imagetools inspect "${IMAGE_REPO}:${TAG}" 2>&1)"; then
    exit 0
  fi
  # Docker Hub answers an anonymous request for a repo that doesn't exist yet
  # (a brand-new image) with an auth error rather than "not found".
  case "$output" in
    *": not found"* | *"repository does not exist"*) exit 1 ;;
  esac
  if [ "$attempt" -ge "$ATTEMPTS" ]; then
    echo "registry-tag-exists.sh: could not check ${IMAGE_REPO}:${TAG} after ${ATTEMPTS} attempts: ${output}" >&2
    exit 3
  fi
  echo "registry-tag-exists.sh: checking ${IMAGE_REPO}:${TAG} failed, retrying (attempt ${attempt}/${ATTEMPTS}): ${output}" >&2
  sleep "$wait_seconds"
  wait_seconds=$((wait_seconds * 2))
  attempt=$((attempt + 1))
done
