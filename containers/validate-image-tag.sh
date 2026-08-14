#!/usr/bin/env bash
# validate-image-tag.sh <tag>
#
# Validates a content-addressed image tag before it's used in any `docker
# push`/`docker buildx imagetools create` command. This is the trusted-side
# check on a tag string that arrived via a build artifact from a job that
# may have run untrusted (fork PR) content - see image-push.yml.
#
# Requires:
#   - strict charset, matching the same sanitization autotag.sh applies
#   - must end in exactly HASH_LEN lowercase hex characters (the part
#     tooling treats as authoritative)
#   - must not be a reserved literal (e.g. "latest") or a release-tag
#     shape (vX.Y.Z), so a forged tag can never collide with a real one
#
# Env:
#   HASH_LEN - hash length in hex chars (default 10, must match hash-paths.sh)

set -eu -o pipefail

HASH_LEN="${HASH_LEN:-10}"

RESERVED_TAGS=(latest stable edge)

if [ "$#" -ne 1 ]; then
  echo "Usage: $0 <tag>" >&2
  exit 2
fi

TAG="$1"

if ! [[ "$TAG" =~ ^[A-Za-z0-9_.-]+-[0-9a-f]{${HASH_LEN}}$ ]]; then
  echo "validate-image-tag.sh: '${TAG}' does not match <name>-<${HASH_LEN}-hex-char-hash>" >&2
  exit 1
fi

for reserved in "${RESERVED_TAGS[@]}"; do
  if [ "$TAG" = "$reserved" ]; then
    echo "validate-image-tag.sh: '${TAG}' is a reserved tag" >&2
    exit 1
  fi
done

if [[ "$TAG" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "validate-image-tag.sh: '${TAG}' looks like a release tag, not a content-hash tag" >&2
  exit 1
fi

exit 0
