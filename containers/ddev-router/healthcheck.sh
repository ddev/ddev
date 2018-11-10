#!/bin/bash
set -eu
set -o pipefail

# Check nginx config
nginx -t || exit 1
# Check our healthcheck endpoint
curl -s --fail http://127.0.0.1/healthcheck >/dev/null || (echo "healthcheck endpoint not responding" && exit 2)