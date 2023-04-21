#!/bin/bash
set -x
set -o errexit nounset pipefail

rm -f /tmp/healthy

# If user has not been created via normal template (like blackfire uid 999)
# then try to grab the required files from /etc/skel
if [ ! -f ~/.gitconfig ]; then cp -r /etc/skel/. ~/ || true; fi

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
    mkdir -p ~/.drush/commands && ln -s /var/tmp/backdrop_drush_commands ~/.drush/commands/backdrop
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

# Disable xhprof by default. Users can enable with /usr/local/bin/enable_xhprof
disable_xhprof

ls /var/www/html >/dev/null || (echo "/var/www/html does not seem to be healthy/mounted; docker may not be mounting it., exiting" && exit 101)

mkdir -p /mnt/ddev-global-cache/{bashhistory/${HOSTNAME},mysqlhistory/${HOSTNAME},nvm_dir/${HOSTNAME},npm,yarn/classic,yarn/berry}
ln -sf /mnt/ddev-global-cache/nvm_dir/${HOSTNAME} ~/.nvm
if [ ! -f ~/.nvm/nvm.sh ]; then (install_nvm.sh || true); fi

# The following ensures a persistent and shared "global" cache for
# yarn1 (classic) and yarn2 (berry). In the case of yarn2, the global cache
# will only be used if the project is configured to use it through it's own
# enableGlobalCache configuration option. Assumes ~/.yarn/berry as the default
# global folder.
( (cd ~ || echo "unable to cd to home directory"; exit 22) && yarn config set cache-folder /mnt/ddev-global-cache/yarn/classic || true)
# ensure default yarn2 global folder is there to symlink cache afterwards
mkdir -p ~/.yarn/berry
ln -sf /mnt/ddev-global-cache/yarn/berry ~/.yarn/berry/cache
ln -sf /mnt/ddev-global-cache/nvm_dir/${HOSTNAME} ~/.nvm
if [ ! -f ~/.nvm/nvm.sh ]; then (install_nvm.sh || true); fi

# chown of ddev-global-cache must be done with privileged container in app.Start()
# chown -R "$(id -u):$(id -g)" /mnt/ddev-global-cache/

if [ -d /mnt/ddev_config/.homeadditions ]; then
    cp -r /mnt/ddev_config/.homeadditions/. ~/
fi

# It's possible CAROOT does not exist or is not writeable (if host-side mkcert -install not run yet)
mkdir -p ${CAROOT}
# This will install the certs from $CAROOT (/mnt/ddev-global-cache/mkcert)
mkcert -install

# VIRTUAL_HOST is a comma-delimited set of fqdns, convert it to space-separated and mkcert
CAROOT=$CAROOT mkcert -cert-file /etc/ssl/certs/master.crt -key-file /etc/ssl/certs/master.key ${VIRTUAL_HOST//,/ } localhost 127.0.0.1 ${DOCKER_IP} web ddev-${DDEV_PROJECT:-}-web ddev-${DDEV_PROJECT:-}-web.ddev
echo 'Server started'

# We don't want the various daemons to know about PHP_IDE_CONFIG
unset PHP_IDE_CONFIG

# In the unusual case where someone is using ddev-webserver standalone
# without DDEV, they'll want it to start services as done in post-start.sh
if [ "${DDEV_AUTO_RUN_SUPERVISORD:-false}" = "true" ]; then
  /usr/bin/supervisord -c "/etc/supervisor/supervisord-${DDEV_WEBSERVER_TYPE}.conf"
fi

# Now just wait forever, keeping the container up
exec sleep infinity
