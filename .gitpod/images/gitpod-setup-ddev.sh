#!/usr/bin/env bash

# Set up ddev for use on gitpod

set -eu -o pipefail

if [ $# != 1 ]; then
  echo "Usage: gitpod-setup-ddev.sh <project-path>" && exit 1
fi

PROJDIR="$1/.ddev"
cd ${PROJDIR}

# Generate a config.gitpod.yaml that adds the gitpod
# proxied ports so they're known to ddev.
shortgpurl="${GITPOD_WORKSPACE_URL#'https://'}"

cat <<CONFIGEND > ${PROJDIR}/config.gitpod.yaml
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

# We need host.docker.internal inside the container,
# So add it via docker-compose.host-docker-internal.yaml
hostip=$(awk "\$2 == \"$HOSTNAME\" { print \$1; }" /etc/hosts)

cat <<COMPOSEEND >${PROJDIR}/docker-compose.host-docker-internal.yaml
#ddev-gitpod-generated
version: "3.6"
services:
  web:
    extra_hosts:
    - "host.docker.internal:${hostip}"
COMPOSEEND
