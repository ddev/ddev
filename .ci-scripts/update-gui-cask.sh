#!/usr/bin/env bash
# Writes Casks/ddev-gui.rb in the Homebrew tap for a release, from
# gui/packaging/ddev-gui.rb.tmpl. goreleaser attaches the zip to the
# release but cannot write a cask for an .app bundle it did not build.
# Usage: update-gui-cask.sh <tag> <DDEV.app.zip> <owner>
# Needs GITHUB_TOKEN with write access to <owner>/homebrew-ddev.

set -o errexit -o nounset -o pipefail

tag=$1
zip=$2
owner=$3
version=${tag#v}
sha256=$(sha256sum "${zip}" | awk '{print $1}')

cask=$(sed -e "s/@VERSION@/${version}/g" -e "s/@SHA256@/${sha256}/g" -e "s/@OWNER@/${owner}/g" \
  "$(dirname "$0")/../gui/packaging/ddev-gui.rb.tmpl")

path="Casks/ddev-gui.rb"
repo="${owner}/homebrew-ddev"
# On a 404, as before the first release, gh prints the error body to stdout.
if ! existing_sha=$(GH_TOKEN="${GITHUB_TOKEN}" gh api "repos/${repo}/contents/${path}" --jq .sha 2>/dev/null); then
  existing_sha=""
fi
args=(-X PUT "repos/${repo}/contents/${path}"
  -f message="ddev-gui ${version}"
  -f content="$(printf '%s\n' "${cask}" | base64 -w0)")
if [ -n "${existing_sha}" ]; then
  args+=(-f sha="${existing_sha}")
fi
GH_TOKEN="${GITHUB_TOKEN}" gh api "${args[@]}" >/dev/null
echo "Updated ${repo}/${path} to ${version}"
