#!/bin/bash
set -x
set -o errexit nounset pipefail

# If DDEV_PHP_VERSION isn't set, use a reasonable default
DDEV_PHP_VERSION=${DDEV_PHP_VERSION:-$PHP_DEFAULT_VERSION}

# Update full path NGINX_DOCROOT if DOCROOT env is provided
if [ -n "$DOCROOT" ] ; then
    export NGINX_DOCROOT="/var/www/html/$DOCROOT"
fi

if [ -f "/mnt/ddev_config/nginx-site.conf" ] ; then
    export NGINX_SITE_TEMPLATE="/mnt/ddev_config/nginx-site.conf"
fi

# Update the default PHP and FPM versions a DDEV_PHP_VERSION like '5.6' or '7.0' is provided
# Otherwise it will use the default version configured in the Dockerfile
if [ -n "$DDEV_PHP_VERSION" ] ; then
	update-alternatives --set php /usr/bin/php${DDEV_PHP_VERSION}
	ln -sf /usr/sbin/php-fpm${DDEV_PHP_VERSION} /usr/sbin/php-fpm
	export PHP_INI=/etc/php/${DDEV_PHP_VERSION}/fpm/php.ini
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
    rm -f /home/.drush/commands/backdrop
    mkdir -p /home/.drush/commands && ln -s /var/tmp/backdrop_drush_commands /home/.drush/commands/backdrop
fi


# Get and link a specific nginx-site.conf for our project type (if it exists)
rm -f /etc/nginx/nginx-site.conf
if [ -f /etc/nginx/nginx-site-$DDEV_PROJECT_TYPE.conf ] ; then
    ln -s /etc/nginx/nginx-site-$DDEV_PROJECT_TYPE.conf /etc/nginx/nginx-site.conf
else
    ln -s /etc/nginx/nginx-site-default.conf /etc/nginx/nginx-site.conf
fi

# Substitute values of environment variables in nginx configuration
envsubst "$NGINX_SITE_VARS" < "$NGINX_SITE_TEMPLATE" > /etc/nginx/sites-enabled/nginx-site.conf

# Disable xdebug by default. Users can enable with /usr/local/bin/enable_xdebug
if [ "$DDEV_XDEBUG_ENABLED" != "true" ]; then
    disable_xdebug
fi

echo 'Server started'

exec /usr/bin/supervisord -n -c /etc/supervisord.conf
