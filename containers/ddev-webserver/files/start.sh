#!/bin/bash
set -x
set -o errexit nounset pipefail

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
printf "\nexport APACHE_RUN_USER=uid_$(id -u)\nexport APACHE_RUN_GROUP=gid_$(id -g)\n" >>/etc/apache2/envvars
if [ "$DDEV_WEBSERVER_TYPE" = "apache-cgi" ] ; then
    a2enmod php${DDEV_PHP_VERSION}
    a2dismod proxy_fcgi
    a2enmod rewrite
    a2dissite 000-default
fi
if [ "$DDEV_WEBSERVER_TYPE" = "apache-fpm" ] ; then
    a2enmod proxy_fcgi setenvif
    a2enconf php${DDEV_PHP_VERSION}-fpm
    a2enmod rewrite
    a2dissite 000-default
fi

# Disable xdebug by default. Users can enable with /usr/local/bin/enable_xdebug
if [ "$DDEV_XDEBUG_ENABLED" != "true" ]; then
    disable_xdebug
fi

echo 'Server started'

exec /usr/bin/supervisord -n -c "/etc/supervisor/supervisord-${DDEV_WEBSERVER_TYPE}.conf"
