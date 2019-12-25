#!/bin/bash
set -x
set -o errexit nounset pipefail

rm -f /tmp/healthy

# If DDEV_PHP_VERSION isn't set, use a reasonable default
DDEV_PHP_VERSION="${DDEV_PHP_VERSION:-$PHP_DEFAULT_VERSION}"

# If DDEV_WEBSERVER_TYPE isn't set, use a reasonable default
DDEV_WEBSERVER_TYPE="${DDEV_WEBSERVER_TYPE:-nginx-fpm}"

# Update full path WEBSERVER_DOCROOT if DOCROOT env is provided
if [ -n "$DOCROOT" ] ; then
    export WEBSERVER_DOCROOT="/var/www/html/$DOCROOT"
    # NGINX_DOCROOT is for backward compatibility of custom config only
    export NGINX_DOCROOT=$WEBSERVER_DOCROOT
fi

if [ -f "/mnt/ddev_config/nginx-site.conf" ] ; then
    export NGINX_SITE_TEMPLATE="/mnt/ddev_config/nginx-site.conf"
fi
if [ -f "/mnt/ddev_config/apache/apache-site.conf" ]; then
    export APACHE_SITE_TEMPLATE="/mnt/ddev_config/apache/apache-site.conf"
fi

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
        cp /mnt/ddev_config/php/*.ini /etc/php/${DDEV_PHP_VERSION}/apache2/conf.d/
    fi
fi

if [ "$DDEV_PROJECT_TYPE" = "backdrop" ] ; then
    # Start can be executed when the container is already running.
    mkdir -p ~/.drush/commands && ln -s /var/tmp/backdrop_drush_commands ~/.drush/commands/backdrop
fi


# Get and link a specific nginx-site.conf for our project type (if it exists)
rm -f /etc/nginx/nginx-site.conf
if [ -f /etc/nginx/nginx-site-$DDEV_PROJECT_TYPE.conf ] ; then
    ln -s /etc/nginx/nginx-site-$DDEV_PROJECT_TYPE.conf /etc/nginx/nginx-site.conf
else
    ln -s /etc/nginx/nginx-site-default.conf /etc/nginx/nginx-site.conf
fi

# Get and link a specific apache-site-<project>.conf for our project type (if it exists)
rm -f /etc/apache2/apache-site.conf
if [ -f /etc/apache2/ddev_apache-$DDEV_PROJECT_TYPE.conf ] ; then
    ln -s -f /etc/apache2/apache-site-$DDEV_PROJECT_TYPE.conf /etc/apache2/apache-site.conf
else
    ln -s -f /etc/apache2/apache-site-default.conf /etc/apache2/apache-site.conf
fi

# Substitute values of environment variables in nginx and apache configuration
envsubst "$NGINX_SITE_VARS" < "$NGINX_SITE_TEMPLATE" > /etc/nginx/sites-enabled/nginx-site.conf
envsubst "$APACHE_SITE_VARS" < "$APACHE_SITE_TEMPLATE" > /etc/apache2/sites-enabled/apache-site.conf

# Change the apache run user to current user/group
printf "\nexport APACHE_RUN_USER=$(id -un)\nexport APACHE_RUN_GROUP=$(id -gn)\n" >>/etc/apache2/envvars

a2enmod access_compat alias auth_basic authn_core authn_file authz_core authz_host authz_user autoindex deflate dir env filter mime mpm_prefork negotiation reqtimeout rewrite setenvif status
a2enconf charset localized-error-pages other-vhosts-access-log security serve-cgi-bin

if [ "$DDEV_WEBSERVER_TYPE" = "apache-cgi" ] ; then
    a2enmod php${DDEV_PHP_VERSION}
    a2dismod proxy_fcgi
    a2dissite 000-default
fi
if [ "$DDEV_WEBSERVER_TYPE" = "apache-fpm" ] ; then
    a2enmod proxy_fcgi 
    a2enconf php${DDEV_PHP_VERSION}-fpm
    a2dissite 000-default
fi

# Disable xdebug by default. Users can enable with /usr/local/bin/enable_xdebug
if [ "$DDEV_XDEBUG_ENABLED" != "true" ]; then
    disable_xdebug
fi

ls /var/www/html >/dev/null || (echo "/var/www/html does not seem to be healthy/mounted; docker may not be mounting it., exiting" && exit 101)

# Make sure the TERMINUS_CACHE_DIR (/mnt/ddev-global-cache/terminus/cache) exists
sudo mkdir -p ${TERMINUS_CACHE_DIR}

# /home/.* is a prototype for the created user's homedir; copy it in.
sudo cp -r /home/{.ssh*,.drush,.gitconfig,.my.cnf} ~/
sudo mkdir -p /mnt/ddev-global-cache/bashhistory/${HOSTNAME}
sudo chown -R "$(id -u):$(id -g)" /mnt/ddev-global-cache/ ~/{.ssh*,.drush,.gitconfig,.my.cnf}

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
