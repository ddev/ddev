#!/bin/bash

# ddev-webserver healthcheck

set -eo pipefail

curl --fail -s 127.0.0.1/phpstatus >/dev/null && printf "phpstatus: OK, " || (printf "phpstatus FAILED" && exit 1)
ls /var/www/html >/dev/null && printf "/var/www/html: OK, " || (printf "/var/www/html access FAILED" && exit 2)
curl --fail -s localhost:8025 >/dev/null && printf "mailhog: OK" || (printf "mailhog FAILED" && exit 3)
