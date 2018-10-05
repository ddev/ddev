#!/bin/bash

# ddev-webserver healthcheck

set -eo pipefail

curl --fail -s localhost/phpstatus >/dev/null && printf "phpstatus: OK, " || (printf "phpstatus FAILED" && exit 1)
mysql db -e "SHOW DATABASES LIKE 'db';" >/dev/null && printf "mariadb: OK, " || (printf "mariadb FAILED" && exit 2)
ls /var/www/html >/dev/null && printf "/var/www/html: OK, " || (printf "/var/www/html access FAILED" && exit 3)
curl --fail -s localhost:8025 >/dev/null && printf "mailhog: OK" || (printf "mailhog FAILED" && exit 4)
