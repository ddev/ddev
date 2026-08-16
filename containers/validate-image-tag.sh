#!/usr/bin/env bash
# validate-image-tag.sh <tag>
#
# Validates a content-addressed image tag before it's used in any `docker
# push`/`docker buildx imagetools create` command. This is the trusted-side
# check on a tag string that arrived via a build artifact from a job that
# may have run untrusted (fork PR) content - see image-push.yml.
#
# Requires:
#   - strict charset, matching the same sanitization autotag.sh applies, and
#     a leading character Docker actually accepts in a tag
#   - must end in exactly HASH_LEN lowercase hex characters (the part
#     tooling treats as authoritative)
#   - neither the whole tag nor the part before the hash may be a reserved
#     literal ("latest") or a release-tag shape (vX.Y.Z), so a forged tag can
#     never be mistaken for a real one
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

reject_reserved() {
  local candidate="$1" what="$2"
  for reserved in "${RESERVED_TAGS[@]}"; do
    if [ "$candidate" = "$reserved" ]; then
      echo "validate-image-tag.sh: '${TAG}' ${what} is the reserved tag '${reserved}'" >&2
      exit 1
    fi
  done
  if [[ "$candidate" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    echo "validate-image-tag.sh: '${TAG}' ${what} looks like a release tag, not a content-hash tag" >&2
    exit 1
  fi
}

reject_reserved "$TAG" "is"

if ! [[ "$TAG" =~ ^[A-Za-z0-9_][A-Za-z0-9_.-]*-[0-9a-f]{${HASH_LEN}}$ ]]; then
  echo "validate-image-tag.sh: '${TAG}' does not match <name>-<${HASH_LEN}-hex-char-hash>" >&2
  exit 1
fi

# Without this, "latest-0123456789" or "v1.2.3-0123456789" would sail through
# the format check above and land next to the real tags in the registry.
reject_reserved "${TAG%-*}" "starts with what"

exit 0
