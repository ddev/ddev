#!/bin/bash
set -eu
set -o pipefail

# Check nginx config
nginx -t || exit 1
# Check our healthcheck endpoint
curl -s --fail --connect-timeout 2 --retry 2 http://127.0.0.1/healthcheck || (echo "ddev-router healthcheck endpoint not responding" && exit 2)
