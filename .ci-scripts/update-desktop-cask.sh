#!/usr/bin/env bash
# Writes Casks/ddev-desktop.rb in the Homebrew tap for a release, from
# desktop/packaging/ddev-desktop.rb.tmpl. goreleaser attaches the zip to the
# release but cannot write a cask for an .app bundle it did not build.
# Usage: update-desktop-cask.sh <tag> <DDEV.app.zip> <owner>
# Needs GITHUB_TOKEN with write access to <owner>/homebrew-ddev.

set -o errexit -o nounset -o pipefail

tag=$1
zip=$2
owner=$3
version=${tag#v}
sha256=$(sha256sum "${zip}" | awk '{print $1}')

cask=$(sed -e "s/@VERSION@/${version}/g" -e "s/@SHA256@/${sha256}/g" -e "s/@OWNER@/${owner}/g" \
  "$(dirname "$0")/../desktop/packaging/ddev-desktop.rb.tmpl")

path="Casks/ddev-desktop.rb"
repo="${owner}/homebrew-ddev"
existing_sha=$(GH_TOKEN="${GITHUB_TOKEN}" gh api "repos/${repo}/contents/${path}" --jq .sha 2>/dev/null || true)
args=(-X PUT "repos/${repo}/contents/${path}"
  -f message="ddev-desktop ${version}"
  -f content="$(printf '%s\n' "${cask}" | base64 -w0)")
if [ -n "${existing_sha}" ]; then
  args+=(-f sha="${existing_sha}")
fi
GH_TOKEN="${GITHUB_TOKEN}" gh api "${args[@]}" >/dev/null
echo "Updated ${repo}/${path} to ${version}"
