#!/bin/bash

## mysql health check for docker. original source: https://github.com/docker-library/healthcheck/blob/master/mysql/docker-healthcheck

set -eo pipefail

mysql --host=127.0.0.1 -udb -pdb --database=db -e "SHOW DATABASES LIKE 'db';" >/dev/null

