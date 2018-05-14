#!/bin/bash

# nginx and php-fpm healthcheck

set -eo pipefail

curl --fail localhost/fpmstatus
curl --fail localhost:8025 >/dev/null 2>&1