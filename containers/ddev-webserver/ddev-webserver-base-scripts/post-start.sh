#!/bin/bash

set -x
set -o errexit nounset pipefail

# VIRTUAL_HOST is a comma-delimited set of fqdns, convert it to space-separated and mkcert
CAROOT=$CAROOT mkcert -cert-file /etc/ssl/certs/master.crt -key-file /etc/ssl/certs/master.key ${VIRTUAL_HOST//,/ } localhost 127.0.0.1 ${DOCKER_IP} web ddev-${DDEV_PROJECT:-}-web ddev-${DDEV_PROJECT:-}-web.ddev
echo 'Server started'

# We don't want the various daemons to know about PHP_IDE_CONFIG
unset PHP_IDE_CONFIG

# Run any python/django4 activities.
ddev_python_setup

# Run any custom init scripts (.ddev/.web-entrypoint.d/*.sh)
if [ -d ${ENTRYPOINT} ]; then
  if [[ -n $(find ${ENTRYPOINT} -type f -regex ".*\.\(sh\)") ]] && [[ ! -f "${ENTRYPOINT}/.user_scripts_initialized" ]] ; then
    # For web-entrypoint.d to work with code already loaded, if mutagen is enabled,
    # the code may not yet be in /var/www/html, so boost it along early
    # The .start-synced file is created after mutagen sync is done, and deleted early
    # in `ddev start`.
    if [ "${DDEV_MUTAGEN_ENABLED}" = "true" ] && [ ! -f /var/www/html/.ddev/mutagen/.start-synced ]; then
      RSYNC_CMD="sudo rsync -a /var/tmp/html/ /var/www/html/"
      if [ "${DDEV_FILES_DIR:-}" != "" ]; then
        RSYNC_CMD="${RSYNC_CMD} --exclude ${DDEV_FILES_DIR#/var/www/html/} --exclude=.git --exclude=.tarballs --exclude=.idea"
      fi
      time ${RSYNC_CMD} || true
    fi
    ddev_custom_init_scripts;
  fi
fi

nohup /usr/bin/supervisord -n -c "/etc/supervisor/supervisord-${DDEV_WEBSERVER_TYPE}.conf"
