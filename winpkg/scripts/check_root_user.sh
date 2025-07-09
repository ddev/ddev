#!/bin/bash
set -eu -o pipefail

id=$(id -u)
if [ "$id" -eq 0 ]; then
  echo "The user running this script must not be root."
  exit 1
fi