#!/bin/bash

set -x
set -o errexit nounset pipefail

source /functions.sh

# VIRTUAL_HOST is a comma-delimited set of fqdns, convert it to space-separated and mkcert
CAROOT=$CAROOT mkcert -cert-file /etc/ssl/certs/master.crt -key-file /etc/ssl/certs/master.key ${VIRTUAL_HOST//,/ } localhost 127.0.0.1 ${DOCKER_IP} web ddev-${DDEV_PROJECT:-}-web ddev-${DDEV_PROJECT:-}-web.ddev

# We don't want the various daemons to know about PHP_IDE_CONFIG
unset PHP_IDE_CONFIG

# Run any python/django4 activities.
ddev_python_setup

# Run any custom init scripts (.ddev/.web-entrypoint.d/*.sh)
if [ -d ${DDEV_WEB_ENTRYPOINT} ]; then
  if [[ -n $(find ${DDEV_WEB_ENTRYPOINT} -type f -regex ".*\.\(sh\)") ]] && [[ ! -f "${DDEV_WEB_ENTRYPOINT}/.user_scripts_initialized" ]] ; then
    ddev_custom_init_scripts;
  fi
fi

echo 'Server started'

nohup /usr/bin/supervisord -n -c "/etc/supervisor/supervisord-${DDEV_WEBSERVER_TYPE}.conf" >/var/log/nohup.out 2>&1 &
