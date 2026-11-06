#!/usr/bin/env bash

# Starts a DDEV release: creates a draft GitHub release from the prepared notes
# and dispatches the release workflow, which creates the tag, attaches every
# asset to the draft, and publishes it. Releases are immutable once published,
# so nothing here publishes. See docs/content/developers/release-management.md.

set -eu -o pipefail

usage() {
  cat <<'EOF'
Usage: scripts/release.sh <tag> (--notes-issue <number> | --notes-file <file>) [options]

  <tag>                   vX.Y.Z, or vX.Y.Z-alpha1/-beta1/-rc1 for a prerelease
  --notes-issue <number>  Issue whose body or comment holds the notes, starting
                          at the line "## Installation"
  --notes-comment <id>    Comment to use when more than one has the notes
  --notes-file <file>     Use a local file as the notes instead
  --branch <branch>       Branch to release from and run the workflow on (default: main)
  --since <tag>           Previous release for generated notes (default: the
                          latest stable release, or the latest release for a prerelease)
  --repo <owner/name>     Repository (default: ddev/ddev)
  --no-image-check        Skip the check that versionconstants.go names the release
  --dry-run               Show the notes and checks, create nothing
  --yes                   Do not ask before starting the workflow
EOF
}

TAG=""
REPO="ddev/ddev"
BRANCH="main"
NOTES_ISSUE=""
NOTES_COMMENT=""
NOTES_FILE=""
SINCE=""
IMAGE_CHECK=true
DRY_RUN=false
YES=false

while [ $# -gt 0 ]; do
  case "$1" in
    --notes-issue) NOTES_ISSUE="$2"; shift 2 ;;
    --notes-comment) NOTES_COMMENT="$2"; shift 2 ;;
    --notes-file) NOTES_FILE="$2"; shift 2 ;;
    --branch) BRANCH="$2"; shift 2 ;;
    --since) SINCE="$2"; shift 2 ;;
    --repo) REPO="$2"; shift 2 ;;
    --no-image-check) IMAGE_CHECK=false; shift ;;
    --dry-run) DRY_RUN=true; shift ;;
    --yes) YES=true; shift ;;
    -h|--help) usage; exit 0 ;;
    -*) echo "Unknown option: $1" >&2; usage >&2; exit 2 ;;
    *) [ -z "$TAG" ] || { echo "Unexpected argument: $1" >&2; exit 2; }; TAG="$1"; shift ;;
  esac
done

die() { echo "release: $*" >&2; exit 1; }

[ -n "$TAG" ] || { usage >&2; exit 2; }
if [ -n "$NOTES_ISSUE" ] && [ -n "$NOTES_FILE" ]; then die "use --notes-issue or --notes-file, not both"; fi
if [ -z "$NOTES_ISSUE" ] && [ -z "$NOTES_FILE" ]; then die "--notes-issue or --notes-file is required"; fi
command -v gh >/dev/null || die "gh is required"
command -v jq >/dev/null || die "jq is required"

# Same shapes containers/release-prep.sh accepts. Test repositories may use any
# semver prerelease, such as v1.28.8-PR2022FixStuff.
if [[ "$TAG" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-(alpha|beta|rc)[0-9.]*)?$ ]]; then
  :
elif [ "$REPO" != "ddev/ddev" ] && [[ "$TAG" =~ ^v1\.[0-9]+\.[0-9]+-[0-9A-Za-z.-]+$ ]]; then
  :
else
  die "'$TAG' is not a release tag (vX.Y.Z or vX.Y.Z-alpha1/-beta1/-rc1)"
fi
PRERELEASE=false
[[ "$TAG" == *-* ]] && PRERELEASE=true

releases_json="$(gh api "repos/${REPO}/releases?per_page=100")"
existing="$(jq -c --arg t "$TAG" '[.[] | select(.tag_name == $t)]' <<<"$releases_json")"
REUSE_DRAFT=false
case "$(jq -r 'map(if .draft then "draft" else "published" end) | sort | join(",")' <<<"$existing")" in
  "") ;;
  draft) REUSE_DRAFT=true ;;
  *) die "${REPO} already has a published release, or several drafts, for ${TAG}" ;;
esac

SHA="$(gh api "repos/${REPO}/commits/${BRANCH}" --jq .sha)"
if gh api "repos/${REPO}/git/ref/tags/${TAG}" >/dev/null 2>&1; then
  tag_sha="$(gh api "repos/${REPO}/commits/${TAG}" --jq .sha)"
  [ "$tag_sha" = "$SHA" ] || die "tag ${TAG} already exists at ${tag_sha}, not at ${BRANCH} (${SHA})"
fi

if [ "$IMAGE_CHECK" = true ]; then
  # release-prep sets every image's *TagBranch to the release.
  constants="$(gh api -H 'Accept: application/vnd.github.raw' "repos/${REPO}/contents/pkg/versionconstants/versionconstants.go?ref=${SHA}")"
  total="$(grep -cE '^var [A-Za-z]+TagBranch = ' <<<"$constants" || true)"
  matching="$(grep -cE "^var [A-Za-z]+TagBranch = \"${TAG}\"" <<<"$constants" || true)"
  if [ "$total" -eq 0 ] || [ "$matching" -ne "$total" ]; then
    die "${matching} of ${total} image tags in versionconstants.go at ${BRANCH} name ${TAG}. Merge the 'make release-prep TAG=${TAG}' pull request first, or pass --no-image-check"
  fi
fi

if [ -z "$SINCE" ]; then
  if [ "$PRERELEASE" = true ]; then
    SINCE="$(jq -r '[.[] | select(.draft | not)][0].tag_name // empty' <<<"$releases_json")"
  else
    SINCE="$(jq -r '[.[] | select((.draft | not) and (.prerelease | not))][0].tag_name // empty' <<<"$releases_json")"
  fi
  [ -n "$SINCE" ] || die "could not find the previous release; pass --since"
fi

notes="$(mktemp "${TMPDIR:-/tmp}/ddev-release-notes.XXXXXX")"
trap 'rm -f "$notes"' EXIT

if [ -n "$NOTES_FILE" ]; then
  tr -d '\r' <"$NOTES_FILE" >"$notes"
else
  # The issue body and each comment, as {id, body}; the body has id "issue".
  candidates="$( {
    gh api "repos/${REPO}/issues/${NOTES_ISSUE}" --jq '{id: "issue", body: (.body // "")}'
    gh api --paginate "repos/${REPO}/issues/${NOTES_ISSUE}/comments" --jq '.[] | {id: (.id | tostring), body: (.body // "")}'
  } | jq -s '[.[] | .body |= gsub("\r"; "") | select(.body | test("(^|\n)## Installation\n"))]')"
  if [ -n "$NOTES_COMMENT" ]; then
    candidates="$(jq --arg id "$NOTES_COMMENT" '[.[] | select(.id == $id)]' <<<"$candidates")"
  fi
  count="$(jq length <<<"$candidates")"
  if [ "$count" -ne 1 ]; then
    echo "Candidates with '## Installation' in issue ${NOTES_ISSUE}: $(jq -r 'map(.id) | join(", ")' <<<"$candidates")" >&2
    die "found ${count} notes candidates; pass --notes-comment <id>"
  fi
  jq -r '.[0].body' <<<"$candidates" | sed -n '/^## Installation$/,$p' >"$notes"
  # Drafts write "@ user" so they don't notify anyone before the release.
  sed -i.bak -E 's/@ ([A-Za-z0-9][A-Za-z0-9-]*)/@\1/g' "$notes" && rm -f "$notes.bak"
  if [ "$PRERELEASE" = false ]; then
    sed -i.bak 's#docs\.ddev\.com/en/latest/#docs.ddev.com/en/stable/#g' "$notes" && rm -f "$notes.bak"
  fi
fi
[ -s "$notes" ] || die "the release notes are empty"

echo "Repository:    ${REPO}"
echo "Release:       ${TAG}$([ "$PRERELEASE" = true ] && echo ' (prerelease)')"
echo "Target:        ${BRANCH} at ${SHA}"
echo "Notes since:   ${SINCE}"
echo "Notes:         $(wc -l <"$notes" | tr -d ' ') lines, starting: $(head -1 "$notes")"
if [ "$DRY_RUN" = true ]; then
  echo "---- notes ----"
  cat "$notes"
  exit 0
fi

if [ "$REUSE_DRAFT" = true ]; then
  echo "A draft for ${TAG} already exists; leaving it as it is."
else
  args=(--repo "$REPO" --draft --target "$SHA" --title "$TAG"
        --notes-file "$notes" --generate-notes --notes-start-tag "$SINCE")
  [ "$PRERELEASE" = true ] && args+=(--prerelease)
  gh release create "$TAG" "${args[@]}"
fi
draft_url="$(gh api "repos/${REPO}/releases?per_page=100" --jq ".[] | select(.tag_name == \"${TAG}\" and .draft) | .html_url")"
echo "Draft: ${draft_url}"

if [ "$YES" != true ]; then
  read -r -p "Review the draft, then start the release workflow? [y/N] " answer
  [[ "$answer" =~ ^[Yy] ]] || { echo "Not started. Run again to start it; the draft is kept."; exit 0; }
fi

gh workflow run main-build.yml --repo "$REPO" --ref "$BRANCH" -f release_tag="$TAG" -f previous_tag="$SINCE"
echo "Started. Follow it with: gh run list --repo ${REPO} --workflow main-build.yml --limit 1"
