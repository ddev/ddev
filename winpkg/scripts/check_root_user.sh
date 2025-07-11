#!/bin/bash
set -eu -o pipefail

# Get the user ID, handling any WSL translation warnings
id=$(id -u 2>/dev/null | tail -1)
if [ "$id" -eq 0 ]; then
  echo "The user running this script must not be root."
  exit 1
fi