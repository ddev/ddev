#!/bin/bash
set -x
set -o errexit nounset pipefail

rm -f /tmp/healthy

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

if [ "$DDEV_PROJECT_TYPE" = "backdrop" ] ; then
    # Start can be executed when the container is already running.
    mkdir -p ~/.drush/commands && ln -s /var/tmp/backdrop_drush_commands ~/.drush/commands/backdrop
fi

if [ "${DDEV_PROJECT_TYPE}" = "drupal6" ] || [ "${DDEV_PROJECT_TYPE}" = "drupal7" ] ; then
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

ls /var/www/html >/dev/null || (echo "/var/www/html does not seem to be healthy/mounted; docker may not be mounting it., exiting" && exit 101)

# Make sure the TERMINUS_CACHE_DIR (/mnt/ddev-global-cache/terminus/cache) exists
# Along with ddev-live, platform.sh equivalents
sudo mkdir -p ${TERMINUS_CACHE_DIR} ${PLATFORMSH_CLI_HOME} /mnt/ddev-global-cache/ddev-live

sudo mkdir -p /mnt/ddev-global-cache/bashhistory/${HOSTNAME}
sudo chown -R "$(id -u):$(id -g)" /mnt/ddev-global-cache/ ~/{.ssh*,.drush,.gitconfig,.my.cnf}

if [ -d /mnt/ddev_config/.homeadditions ]; then
    cp -r /mnt/ddev_config/.homeadditions/. ~/
fi
if [ -d /mnt/ddev_config/homeadditions ]; then
    cp -r /mnt/ddev_config/homeadditions/. ~/
fi

# It's possible CAROOT does not exist or is not writeable (if host-side mkcert -install not run yet)
sudo mkdir -p ${CAROOT} && sudo chmod -R ugo+rw ${CAROOT}
# This will install the certs from $CAROOT (/mnt/ddev-global-cache/mkcert)
mkcert -install

# VIRTUAL_HOST is a comma-delimited set of fqdns, convert it to space-separated and mkcert
sudo CAROOT=$CAROOT mkcert -cert-file /etc/ssl/certs/master.crt -key-file /etc/ssl/certs/master.key ${VIRTUAL_HOST//,/ } localhost 127.0.0.1 ${DOCKER_IP} web ddev-${DDEV_PROJECT:-}-web ddev-${DDEV_PROJECT:-}-web.ddev_default && sudo chown $UID /etc/ssl/certs/master.*

echo 'Server started'

exec /usr/bin/supervisord -n -c "/etc/supervisor/supervisord-${DDEV_WEBSERVER_TYPE}.conf"
