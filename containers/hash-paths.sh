#!/usr/bin/env bash
# hash-paths.sh <path> [<path> ...]
#
# Prints a deterministic content hash of the current working-tree content
# under the given path(s), covering both committed and dirty
# (staged/unstaged/untracked-but-not-ignored) state. No Docker, no network.
#
# Env:
#   HASH_LEN - number of hex characters to print (default 10)

set -eu -o pipefail

if [ "$#" -eq 0 ]; then
  echo "Usage: $0 <path> [<path> ...]" >&2
  exit 1
fi

HASH_LEN="${HASH_LEN:-10}"

# Pick a SHA-256 implementation. sha256sum exists on Linux, WSL2, and
# Git Bash on Windows (which ships GNU coreutils); macOS has shasum instead.
# All are invoked in binary mode, which avoids the CRLF translation that
# Cygwin/MSYS coreutils can apply in text mode. Output is normalized below
# to the classic "<hash>  <path>" form so every tool yields identical bytes.
if command -v sha256sum >/dev/null 2>&1; then
  hasher=(sha256sum -b);        stdin_hasher=(sha256sum)
elif command -v shasum >/dev/null 2>&1; then
  hasher=(shasum -a 256 -b);    stdin_hasher=(shasum -a 256)
elif command -v openssl >/dev/null 2>&1; then
  hasher=(openssl dgst -sha256 -r); stdin_hasher=(openssl dgst -sha256 -r)
else
  echo "$0: no SHA-256 tool found (need sha256sum, shasum, or openssl)" >&2
  exit 1
fi

cd "$(git rev-parse --show-toplevel)"

# A test suite or README never ends up in a built image, so it shouldn't
# force a rebuild; excluded only as a direct child of a hashed path, so the
# same name inside a directory the Dockerfile actually COPYs still counts.
EXCLUDE_NAMES=(test tests README.md LICENSE)
PATHSPECS=("$@")
for p in "$@"; do
  if [ -d "$p" ]; then
    for name in "${EXCLUDE_NAMES[@]}"; do
      PATHSPECS+=(":(exclude)$p/$name")
    done
  fi
done

files=$(
  {
    git ls-files -- "${PATHSPECS[@]}"
    git ls-files --others --exclude-standard -- "${PATHSPECS[@]}"
  } | LC_ALL=C sort -u
)

# NUL-delimited so paths containing spaces survive; the explicit emptiness
# check replaces "xargs -r", which is a GNU extension absent on macOS.
if [ -n "$files" ]; then
  printf '%s\n' "$files" | tr '\n' '\0' | xargs -0 "${hasher[@]}" | sed 's/ \*/  /'
fi | LC_ALL=C sort | "${stdin_hasher[@]}" | cut -d' ' -f1 | cut -c1-"${HASH_LEN}"
