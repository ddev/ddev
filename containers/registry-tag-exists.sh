#!/usr/bin/env bash
# registry-tag-exists.sh <image-repo> <tag>
#
# Checks whether <image-repo>:<tag> already exists in the registry, without
# pulling it. Exit 0 if it exists, exit 1 if it doesn't (or the registry
# can't be reached). No local Docker daemon build/pull is triggered either
# way - this only talks to the registry.

set -eu -o pipefail

if [ "$#" -ne 2 ]; then
  echo "Usage: $0 <image-repo> <tag>" >&2
  exit 2
fi

IMAGE_REPO="$1"
TAG="$2"

docker buildx imagetools inspect "${IMAGE_REPO}:${TAG}" >/dev/null 2>&1
