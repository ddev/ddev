#!/usr/bin/env bash
# variants.sh <view> [<host-arch>]
#
# Renders the variant matrix in variants.txt for each of its consumers, so the
# list lives in exactly one place. Views:
#
#   build-targets <host-arch>   make targets for a `make build` on that host:
#                               <variant>_both, or <variant>_<arch> for a
#                               variant that only builds on one. Variants the
#                               host can't build are omitted.
#   single-arch-targets         <variant>_amd64 <variant>_arm64 for every
#                               multi-arch variant, for one-arch-at-a-time
#                               builds (a multi-arch push builds each side
#                               separately, then combines them).
#   multi-arch-variants         the variants that get a combined manifest
#   test-targets <host-arch>    <variant>_test for what the host can build
#   list                        <variant>|<arches> for shell consumers
#   repos                       ddev-dbserver-<type>-<version> repository names
#   json                        [{"dbtype":…,"arch":…}, …] for a GitHub matrix

set -eu -o pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
VARIANTS_FILE="${VARIANTS_FILE:-$SCRIPT_DIR/variants.txt}"

if [ "$#" -lt 1 ]; then
  echo "Usage: $0 <build-targets|single-arch-targets|multi-arch-variants|test-targets|list|repos|json> [host-arch]" >&2
  exit 2
fi

VIEW="$1"
HOST_ARCH="${2:-}"

VARIANTS=()
ARCHES=()
while read -r variant arches; do
  [ -z "$variant" ] && continue
  case "$variant" in \#*) continue ;; esac
  VARIANTS+=("$variant")
  ARCHES+=("$arches")
done < "$VARIANTS_FILE"

if [ "${#VARIANTS[@]}" -eq 0 ]; then
  echo "$0: no variants found in $VARIANTS_FILE" >&2
  exit 1
fi

host_can_build() {
  local arches="$1"
  [ -z "$HOST_ARCH" ] && return 0
  [[ " $arches " == *" $HOST_ARCH "* ]]
}

require_host_arch() {
  if [ -z "$HOST_ARCH" ]; then
    echo "$0: $VIEW needs a host arch argument" >&2
    exit 2
  fi
}

OUT=()
for i in "${!VARIANTS[@]}"; do
  variant="${VARIANTS[$i]}"
  arches="${ARCHES[$i]}"
  # shellcheck disable=SC2086 # arches is a space-separated list
  set -- $arches
  arch_count="$#"

  case "$VIEW" in
    build-targets)
      require_host_arch
      host_can_build "$arches" || continue
      if [ "$arch_count" -gt 1 ]; then
        OUT+=("${variant}_both")
      else
        OUT+=("${variant}_$1")
      fi
      ;;
    single-arch-targets)
      [ "$arch_count" -gt 1 ] || continue
      for arch in $arches; do
        OUT+=("${variant}_${arch}")
      done
      ;;
    multi-arch-variants)
      [ "$arch_count" -gt 1 ] || continue
      OUT+=("$variant")
      ;;
    test-targets)
      require_host_arch
      host_can_build "$arches" || continue
      OUT+=("${variant}_test")
      ;;
    list)
      OUT+=("${variant}|${arches}")
      ;;
    repos)
      OUT+=("ddev-dbserver-${variant/_/-}")
      ;;
    json)
      for arch in $arches; do
        OUT+=("$(printf '{"dbtype":"%s","arch":"%s"}' "$variant" "$arch")")
      done
      ;;
    *)
      echo "$0: unknown view '$VIEW'" >&2
      exit 2
      ;;
  esac
done

case "$VIEW" in
  json)
    printf '[%s]\n' "$(IFS=,; echo "${OUT[*]}")"
    ;;
  list|repos)
    printf '%s\n' "${OUT[@]}"
    ;;
  *)
    echo "${OUT[*]}"
    ;;
esac
