#!/bin/sh

# This uses /bin/sh so it doesn't initialize profile/bashrc/etc

# ddev-webserver healthcheck

set -e

sleeptime=59

# Make sure that mounted code, and mailhog
# are working.
# Since docker doesn't provide a lazy period for startup,
# we track health. If the last check showed healthy
# as determined by existence of /tmp/healthy, then
# sleep at startup. This requires the timeout to be set
# higher than the sleeptime used here.
if [ -f /tmp/healthy ]; then
    printf "container was previously healthy, so sleeping ${sleeptime} seconds before continuing healthcheck...  "
    sleep ${sleeptime}
fi

phpstatus="false"
htmlaccess="false"
gunicornstatus="false"
mailhog="false"

if ls /var/www/html >/dev/null; then
    htmlaccess="true"
    printf "/var/www/html:OK "
else
    printf "/var/www/html:FAILED"
fi

if curl --fail -s 127.0.0.1:8025 >/dev/null; then
    mailhog="true"
    printf "mailhog:OK " ;
else
    printf "mailhog:FAILED "
fi

# If DDEV_WEBSERVER_TYPE is not set, use reasonable default
DDEV_WEBSERVER_TYPE=${DDEV_WEBSERVER_TYPE:-nginx-fpm}

if [ "${DDEV_WEBSERVER_TYPE#*-}" = "gunicorn" ]; then
  phpstatus="true"
  if pkill -0 gunicorn; then
    gunicornstatus="true"
    printf "gunicorn:OK"
  else
    printf "gunicorn:FAILED "
  fi

fi

if [ "${DDEV_WEBSERVER_TYPE#*-}" = "fpm"  ]; then
  gunicornstatus="true"
  if curl --fail -s 127.0.0.1/phpstatus >/dev/null ; then
    phpstatus="true"
    printf "phpstatus:OK "
  else
    printf "phpstatus:FAILED "
  fi
fi

if [ "${phpstatus}" = "true" ] && [ "${gunicornstatus}" = "true" ] && [ "${htmlaccess}" = "true" ] &&  [ "${mailhog}" = "true" ] ; then
    touch /tmp/healthy
    exit 0
fi
rm -f /tmp/healthy

exit 1


