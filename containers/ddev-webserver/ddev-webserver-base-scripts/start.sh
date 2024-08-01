#!/bin/bash
set -x
set -o errexit nounset pipefail

rm -f /tmp/healthy

# If supervisord happens to be running (ddev start when already running) then kill it off
if pkill -0 supervisord; then
  supervisorctl stop all || true
  supervisorctl shutdown || true
fi
rm -f /var/run/supervisor.sock

export DDEV_WEB_ENTRYPOINT=/mnt/ddev_config/web-entrypoint.d

source /functions.sh

# If user has not been created via normal template (like uid 999)
# then try to grab the required files from /etc/skel
if [ ! -f ~/.gitconfig ]; then (sudo cp -r /etc/skel/. ~/ && sudo chown -R "$(id -u -n)" ~ ) || true; fi

# If DDEV_PHP_VERSION isn't set, use a reasonable default
DDEV_PHP_VERSION="${DDEV_PHP_VERSION:-$PHP_DEFAULT_VERSION}"

# If DDEV_WEBSERVER_TYPE isn't set, use a reasonable default
DDEV_WEBSERVER_TYPE="${DDEV_WEBSERVER_TYPE:-nginx-fpm}"

# Update the default PHP and FPM versions a DDEV_PHP_VERSION like '5.6' or '7.0' is provided
# Otherwise it will use the default version configured in the Dockerfile
if [ -n "$DDEV_PHP_VERSION" ] ; then
  update-alternatives --set php /usr/bin/php${DDEV_PHP_VERSION}
  ln -sf /usr/sbin/php-fpm${DDEV_PHP_VERSION} /usr/sbin/php-fpm
  export PHP_INI=/etc/php/${DDEV_PHP_VERSION}/fpm/php.ini
fi

# Set PHP timezone to configured $TZ if there is one
if [ ! -z ${TZ} ]; then
    perl -pi -e "s%date.timezone *=.*$%date.timezone = $TZ%g" $(find /etc/php -name php.ini)
fi

# If the user has provided custom PHP configuration, copy it into a directory
# where PHP will automatically include it.
if [ -d /mnt/ddev_config/php ] ; then
    # If there are files in the mount
    if [ -n "$(ls -A /mnt/ddev_config/php/*.ini 2>/dev/null)" ]; then
        cp /mnt/ddev_config/php/*.ini /etc/php/${DDEV_PHP_VERSION}/cli/conf.d/
        cp /mnt/ddev_config/php/*.ini /etc/php/${DDEV_PHP_VERSION}/fpm/conf.d/
    fi
fi

if [ -d /mnt/ddev_config/nginx_full ]; then
  rm -rf /etc/nginx/sites-enabled
  cp -r /mnt/ddev_config/nginx_full /etc/nginx/sites-enabled/
fi
if [ -d /mnt/ddev_config/apache ]; then
  rm -rf /etc/apache2/sites-enabled
  cp -r /mnt/ddev_config/apache /etc/apache2/sites-enabled
fi

if [ "$DDEV_PROJECT_TYPE" = "backdrop" ] ; then
    # Start can be executed when the container is already running.
    mkdir -p ~/.drush/commands && ln -sf /var/tmp/backdrop_drush_commands ~/.drush/commands/backdrop
fi

if [ "${DDEV_PROJECT_TYPE}" = "drupal6" ] || [ "${DDEV_PROJECT_TYPE}" = "drupal7" ] || [ "${DDEV_PROJECT_TYPE}" = "backdrop" ]; then
  ln -sf /usr/local/bin/drush8 /usr/local/bin/drush
fi

# Change the apache run user to current user/group
printf "\nexport APACHE_RUN_USER=$(id -un)\nexport APACHE_RUN_GROUP=$(id -gn)\n" >>/etc/apache2/envvars

a2enmod access_compat alias auth_basic authn_core authn_file authz_core authz_host authz_user autoindex deflate dir env filter mime mpm_prefork negotiation reqtimeout rewrite setenvif status
a2enconf charset localized-error-pages other-vhosts-access-log security serve-cgi-bin

if [ "$DDEV_WEBSERVER_TYPE" = "apache-fpm" ] ; then
    a2enmod proxy_fcgi
    a2enconf php${DDEV_PHP_VERSION}-fpm
    a2dissite 000-default
fi

# Disable xdebug by default. Users can enable with /usr/local/bin/enable_xdebug
if [ "$DDEV_XDEBUG_ENABLED" = "true" ]; then
  enable_xdebug
else
  disable_xdebug
fi

# Enable assertions by default.
phpenmod assert

ls /var/www/html >/dev/null || (echo "/var/www/html does not seem to be healthy/mounted; docker may not be mounting it., exiting" && exit 101)

# Make sure the TERMINUS_CACHE_DIR (/mnt/ddev-global-cache/terminus/cache) exists
sudo mkdir -p ${TERMINUS_CACHE_DIR}

sudo mkdir -p /mnt/ddev-global-cache/{bashhistory/${HOSTNAME},mysqlhistory/${HOSTNAME},n_prefix/${HOSTNAME},nvm_dir/${HOSTNAME},npm,yarn/classic,yarn/berry,corepack}
sudo chown -R "$(id -u):$(id -g)" /mnt/ddev-global-cache/ /var/lib/php

if [ "${N_PREFIX:-}" != "" ] && [ "${N_INSTALL_VERSION:-}" != "" ]; then
  n-install.sh || true
fi

# The following ensures a persistent and shared "global" cache for
# yarn classic (frozen v1) and yarn berry (active). In the case of berry, the global cache
# will only be used if the project is configured to use it through it's own
# enableGlobalCache configuration option. Assumes ~/.yarn/berry as the default
# global folder.
(if cd ~ || (echo "unable to cd to home directory"; exit 22); then
  yarn config set cache-folder /mnt/ddev-global-cache/yarn/classic >/dev/null || true
fi)
# ensure default yarn berry global folder is there to symlink cache afterwards
mkdir -p ~/.yarn/berry
ln -sf /mnt/ddev-global-cache/yarn/berry ~/.yarn/berry/cache
ln -sf /mnt/ddev-global-cache/nvm_dir/${HOSTNAME} ~/.nvm
if [ ! -f ~/.nvm/nvm.sh ]; then (timeout 30 install_nvm.sh || true); fi

# /mnt/ddev_config/.homeadditions may be either
# a bind-mount, or a volume mount, but we don't care,
# should all be set up with both global and local
# either way.
if [ -d /mnt/ddev_config/.homeadditions ]; then
    cp -r /mnt/ddev_config/.homeadditions/. ~/
fi

# It's possible CAROOT does not exist or is not writeable (if host-side mkcert -install not run yet)
# This will install the certs from $CAROOT (/mnt/ddev-global-cache/mkcert)
# It also creates them if they don't already exist
if [ ! -f  "${CAROOT}/rootCA.pem" ]; then
  echo "rootCA.pem not found in ${CAROOT}"
fi
mkcert -install

# VIRTUAL_HOST is a comma-delimited set of fqdns, convert it to space-separated and mkcert
CAROOT=$CAROOT mkcert -cert-file /etc/ssl/certs/master.crt -key-file /etc/ssl/certs/master.key ${VIRTUAL_HOST//,/ } localhost 127.0.0.1 ${DOCKER_IP} web ddev-${DDEV_PROJECT:-}-web ddev-${DDEV_PROJECT:-}-web.ddev
echo 'Server started'

# We don't want the various daemons to know about PHP_IDE_CONFIG
unset PHP_IDE_CONFIG

# Run any python/django4 activities.
ddev_python_setup

# Run any custom init scripts (.ddev/.web-entrypoint.d/*.sh)
ddev_custom_init_scripts

# Make sure /var/tmp/logpipe gets logged; only for standalone non-ddev usages
logpipe=/var/tmp/logpipe
if [[ ! -p ${logpipe} ]]; then
    mkfifo ${logpipe}
    cat < ${logpipe} >/proc/1/fd/1 &
fi

exec /usr/bin/supervisord -n -c "/etc/supervisor/supervisord-${DDEV_WEBSERVER_TYPE}.conf"
