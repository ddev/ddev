#!/usr/bin/env bash

# Set up ddev for use on gitpod

set -eu -o pipefail

if [ $# != 1 ]; then
  echo "Usage: gitpod-setup-ddev.sh <project-path>" && exit 1
fi

PROJDIR="$1/.ddev"
mkdir -p ${PROJDIR} && cd "${PROJDIR}"

# Generate a config.gitpod.yaml that adds the gitpod
# proxied ports so they're known to ddev.
cat <<CONFIGEND > "${PROJDIR}"/config.gitpod.yaml
web_environment:
- DRUSH_OPTIONS_URI=$(gp url 8080)

bind_all_interfaces: true
host_webserver_port: 8080
# Will ignore the direct-bind https port, which will land on 2222
host_https_port: 2222
# Allows local db clients to run
host_db_port: 3306
# Assign MailHog port
host_mailhog_port: "8025"
# Assign phpMyAdmin port
host_phpmyadmin_port: 8036
CONFIGEND
