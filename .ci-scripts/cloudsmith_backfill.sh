#!/usr/bin/env bash
# One-off/manual tool to backfill historical ddev deb/rpm packages (already
# published as GitHub release assets) into a Cloudsmith repository. This is
# NOT run by CI - the "cloudsmiths" goreleaser publisher in .goreleaser.yml
# handles new releases going forward. This script exists to backfill history
# that predates that publisher.
#
# Packages are pushed to the "any-distro/any-version" distribution, same as
# the "cloudsmiths" goreleaser publisher, so they land in the same universal
# repo new releases will use.
#
# Requires:
#   - gh, authenticated with access to ddev/ddev
#   - jq
#   - cloudsmith-cli (`pipx install cloudsmith-cli` or `pip install cloudsmith-cli`)
#   - CLOUDSMITH_API_KEY set to a Cloudsmith API token with push access to ORG/REPO
#
# Usage: ./.ci-scripts/cloudsmith_backfill.sh [ORG] [REPO] [START_TAG] [END_TAG]
#   ORG        Cloudsmith organization (default: ddev)
#   REPO       Cloudsmith repository   (default: ddev-test)
#   START_TAG  First ddev/ddev release tag to backfill, inclusive (default: v1.24.0)
#   END_TAG    Last release tag to backfill, inclusive (default: latest stable release)
#
# Set DRY_RUN=true to pass --dry-run through to `cloudsmith push` without
# actually uploading anything, to sanity-check the tag range and file list first.
#
# Safe to re-run: without --republish, Cloudsmith flags repeat uploads of an
# already-present version as duplicates rather than erroring destructively, so
# failures on a re-run are usually just that - check the log before assuming
# a real problem.

set -o nounset
set -o pipefail

GH_REPO="ddev/ddev"
ORG=${1:-ddev}
REPO=${2:-ddev-test}
START_TAG=${3:-v1.24.0}
END_TAG=${4:-}
DRY_RUN=${DRY_RUN:-false}

for cmd in gh jq cloudsmith; do
  command -v "${cmd}" >/dev/null 2>&1 || {
    echo "ERROR: '${cmd}' is required but not found in PATH" >&2
    exit 1
  }
done

if [ -z "${CLOUDSMITH_API_KEY:-}" ]; then
  echo "ERROR: CLOUDSMITH_API_KEY must be set to a Cloudsmith API token with push access to ${ORG}/${REPO}" >&2
  exit 1
fi

DRY_RUN_FLAG=()
if [ "${DRY_RUN}" = "true" ]; then
  DRY_RUN_FLAG=(--dry-run)
  echo "DRY_RUN=true: cloudsmith push calls will not actually upload anything"
fi

if [ -z "${END_TAG}" ]; then
  END_TAG=$(gh release list --repo "${GH_REPO}" --json tagName,isPrerelease,isDraft -L 300 |
    jq -r '.[] | select(.isPrerelease==false and .isDraft==false) | .tagName' |
    sort -V | tail -1)
  echo "No END_TAG given, resolved latest stable release: ${END_TAG}"
fi

# Only stable (non-prerelease) releases are considered, matching the
# "cloudsmiths" publisher's own disable-on-prerelease behavior.
TAGS=$(gh release list --repo "${GH_REPO}" --json tagName,isPrerelease,isDraft -L 300 |
  jq -r '.[] | select(.isPrerelease==false and .isDraft==false) | .tagName' |
  sort -V |
  awk -v start="${START_TAG}" -v end="${END_TAG}" '
      $0==start { inrange=1 }
      inrange { print }
      $0==end  { inrange=0 }
    ')

if [ -z "${TAGS}" ]; then
  echo "ERROR: no stable releases found between ${START_TAG} and ${END_TAG}" >&2
  exit 1
fi

echo "Backfilling into Cloudsmith ${ORG}/${REPO} (any-distro/any-version), tags:"
echo "${TAGS}"
echo

LOGFILE="$(mktemp -d)/cloudsmith_backfill.log"
echo "Logging full output to ${LOGFILE}"
echo

PUSHED=0
FAILED=0
FAILED_FILES=()

while IFS= read -r TAG; do
  [ -z "${TAG}" ] && continue
  echo "=== ${TAG} ==="
  WORKDIR=$(mktemp -d)

  # Older releases don't have every package (e.g. ddev-wsl2 was added later),
  # so pull whatever .deb/.rpm assets actually exist for this tag.
  if ! gh release download "${TAG}" --repo "${GH_REPO}" --dir "${WORKDIR}" \
    --pattern '*.deb' --pattern '*.rpm' --clobber >>"${LOGFILE}" 2>&1; then
    echo "  no deb/rpm assets found for ${TAG}, skipping"
    rm -rf "${WORKDIR}"
    continue
  fi

  for FILE in "${WORKDIR}"/*.deb; do
    [ -e "${FILE}" ] || continue
    echo "  pushing $(basename "${FILE}") (deb)"
    if cloudsmith push deb "${ORG}/${REPO}/any-distro/any-version" "${FILE}" \
      --component main --no-wait-for-sync -k "${CLOUDSMITH_API_KEY}" \
      "${DRY_RUN_FLAG[@]}" >>"${LOGFILE}" 2>&1; then
      PUSHED=$((PUSHED + 1))
    else
      FAILED=$((FAILED + 1))
      FAILED_FILES+=("${TAG}/$(basename "${FILE}")")
      echo "  FAILED: $(basename "${FILE}") - see ${LOGFILE}"
    fi
  done

  for FILE in "${WORKDIR}"/*.rpm; do
    [ -e "${FILE}" ] || continue
    echo "  pushing $(basename "${FILE}") (rpm)"
    if cloudsmith push rpm "${ORG}/${REPO}/any-distro/any-version" "${FILE}" \
      --no-wait-for-sync -k "${CLOUDSMITH_API_KEY}" \
      "${DRY_RUN_FLAG[@]}" >>"${LOGFILE}" 2>&1; then
      PUSHED=$((PUSHED + 1))
    else
      FAILED=$((FAILED + 1))
      FAILED_FILES+=("${TAG}/$(basename "${FILE}")")
      echo "  FAILED: $(basename "${FILE}") - see ${LOGFILE}"
    fi
  done

  rm -rf "${WORKDIR}"
done <<<"${TAGS}"

echo
echo "Done. Pushed: ${PUSHED}, Failed: ${FAILED}"
if [ "${FAILED}" -gt 0 ]; then
  echo "Failed files (often just 'already exists' duplicates on a re-run - check ${LOGFILE}):"
  printf '  %s\n' "${FAILED_FILES[@]}"
fi
echo "Full log: ${LOGFILE}"
