#!/bin/bash
set -eu -o pipefail

# WINDOWS_CAROOT must be set
if [ -z "${WINDOWS_CAROOT:-}" ]; then
  echo "WINDOWS_CAROOT must be set to the Windows-side CAROOT"
  exit 1
fi

echo WINDOWS_CAROOT="$WINDOWS_CAROOT"

# This method of setting CAROOT for mkcert is awkward, but
# elevated Windows installer doesn't pass CAROOT to WSL.
export CAROOT=$(wslpath -u "$WINDOWS_CAROOT")
mkcert -install
mkcert -CAROOT