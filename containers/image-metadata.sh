#!/usr/bin/env bash
# image-metadata.sh <labels|annotations|pairs> <image-tag> [<title>]
#
# One definition of the descriptive metadata every DDEV image carries, in
# whichever form the caller needs:
#
#   pairs        key=value lines
#   labels       --label "key=value" ... for a docker/buildx build
#   annotations  --annotation "index:key=value" ... for `imagetools create`
#
# Labels and annotations are not interchangeable, which is why both exist:
#
#   - Labels live in the image config, so they travel with a `docker pull` and
#     `docker inspect` shows them offline. That is what a support report needs.
#   - Annotations live on the manifest index, which is what a tag points at, so
#     they describe the tag as a whole rather than one platform, and a registry
#     can show them without pulling. A tag has no comment field; an index
#     annotation is the closest standard equivalent.
#
# Env:
#   DDEV_GIT_REVISION - commit to record (default: current HEAD)
#   SOURCE_DATE_EPOCH - build timestamp override, for reproducibility

set -eu -o pipefail

if [ "$#" -lt 2 ]; then
  echo "Usage: $0 <labels|annotations|pairs> <image-tag> [<title>]" >&2
  exit 2
fi

FORM="$1"
IMAGE_TAG="$2"
TITLE="${3:-}"

REVISION="${DDEV_GIT_REVISION:-$(git rev-parse HEAD 2>/dev/null || echo unknown)}"
if [ -n "${SOURCE_DATE_EPOCH:-}" ]; then
  CREATED="$(date -u -r "$SOURCE_DATE_EPOCH" +%Y-%m-%dT%H:%M:%SZ 2>/dev/null || date -u -d "@$SOURCE_DATE_EPOCH" +%Y-%m-%dT%H:%M:%SZ)"
else
  CREATED="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
fi

PAIRS=(
  "org.opencontainers.image.source=https://github.com/ddev/ddev"
  "org.opencontainers.image.url=https://ddev.com"
  "org.opencontainers.image.documentation=https://docs.ddev.com"
  "org.opencontainers.image.vendor=DDEV"
  "org.opencontainers.image.licenses=Apache-2.0"
  "org.opencontainers.image.revision=${REVISION}"
  "org.opencontainers.image.version=${IMAGE_TAG}"
  "org.opencontainers.image.created=${CREATED}"
  # The tag is a bare content hash, so this is the one field that says what
  # the image is for without cross-referencing versionconstants.go.
  "com.ddev.image-tag=${IMAGE_TAG}"
)
[ -n "$TITLE" ] && PAIRS+=("org.opencontainers.image.title=${TITLE}")

case "$FORM" in
  pairs)
    printf '%s\n' "${PAIRS[@]}"
    ;;
  labels)
    for p in "${PAIRS[@]}"; do printf -- '--label %q ' "$p"; done
    echo
    ;;
  annotations)
    for p in "${PAIRS[@]}"; do printf -- '--annotation %q ' "index:$p"; done
    echo
    ;;
  *)
    echo "$0: unknown form '$FORM'" >&2
    exit 2
    ;;
esac
