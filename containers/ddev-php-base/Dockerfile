### ---------------------------base--------------------------------------
### Build the base Debian image that will be used in every other image
FROM debian:bullseye-slim as base

RUN ls -l $(which dpkg-split) && ls -l $(which dpkg-deb)
RUN for item in dpkg-split dpkg-deb; do \
  if [ ! -f /usr/sbin/$item ]; then \
    ln -sf /usr/bin/$item /usr/sbin/$item; \
  fi; \
done
RUN for item in tar rm; do \
  if [ ! -f /usr/sbin/$item ]; then \
    ln -sf /bin/$item /usr/sbin/$item; \
  fi; \
done

RUN ls -l /usr/sbin/dpkg-split /usr/sbin/dpkg-deb /usr/sbin/tar /usr/sbin/rm

RUN apt-get -qq update
RUN apt-get -qq install --no-install-recommends --no-install-suggests -y \
    apt-transport-https \
    ca-certificates \
    bzip2 \
    curl \
    git \
    gnupg \
    less \
    lsb-release \
    procps \
    tree \
    vim \
    wget
#END base

### ---------------------------ddev-php-base--------------------------------------
### Build ddev-php-base, which is the base for ddev-php-prod and ddev-webserver-*
### This combines the packages and features of DDEV’s ddev-webserver and PHP image
FROM base AS ddev-php-base
ARG PHP_DEFAULT_VERSION="8.1"
ENV DDEV_PHP_VERSION=$PHP_DEFAULT_VERSION
ENV PHP_VERSIONS="php5.6 php7.0 php7.1 php7.2 php7.3 php7.4 php8.0 php8.1 php8.2 php8.3"
ENV PHP_INI=/etc/php/$PHP_DEFAULT_VERSION/fpm/php.ini
ENV YQ_VERSION=v4.30.5
ENV DRUSH_VERSION=8.4.12
ENV NODE_VERSION=18
# composer normally screams about running as root, we don't need that.
ENV COMPOSER_ALLOW_SUPERUSER 1
ENV COMPOSER_PROCESS_TIMEOUT 2000

# TARGETPLATFORM is Docker buildx's target platform (e.g. linux/arm64), while
# BUILDPLATFORM is the platform of the build host (e.g. linux/amd64)
ARG TARGETPLATFORM
ARG BUILDPLATFORM

SHELL ["/bin/bash", "-c"]

RUN wget -q -O /tmp/nginx_signing.key http://nginx.org/keys/nginx_signing.key && \
    apt-key add /tmp/nginx_signing.key && \
    rm /tmp/nginx_signing.key && \
    echo "deb http://nginx.org/packages/debian/ $(lsb_release -sc) nginx" > /etc/apt/sources.list.d/nginx.list

RUN curl -sSLo /usr/share/keyrings/deb.sury.org-php.gpg https://packages.sury.org/php/apt.gpg && \
    echo "deb [signed-by=/usr/share/keyrings/deb.sury.org-php.gpg] https://packages.sury.org/php/ $(lsb_release -sc) main" > /etc/apt/sources.list.d/php.list && apt-get update
RUN curl -sSL --fail https://deb.nodesource.com/setup_${NODE_VERSION}.x | bash -

RUN apt-get -qq update
RUN apt-get -qq install --no-install-recommends --no-install-suggests -y \
    apache2 \
    file \
    ghostscript \
    imagemagick \
    gunicorn \
    graphicsmagick \
    jq \
    mariadb-client \
    msmtp \
    nginx \
    nodejs \
    php-uploadprogress \
    postgresql-client \
    sqlite3

RUN npm install --unsafe-perm=true --global gulp-cli yarn
# Normal user needs to be able to write to php sessions
RUN chmod 777 /var/lib/php/sessions
RUN set -eu -o pipefail && LATEST=$(curl -L --fail --silent "https://api.github.com/repos/nvm-sh/nvm/releases/latest" | jq -r .tag_name) && curl --fail -sL https://raw.githubusercontent.com/nvm-sh/nvm/${LATEST}/install.sh -o /usr/local/bin/install_nvm.sh && chmod +x /usr/local/bin/install_nvm.sh

# The number of permutations of php packages available on each architecture because
# too much to handle, so has been codified here instead of in obscure logic
ENV php56_amd64="apcu bcmath bz2 curl cli common fpm gd imagick intl json ldap mbstring mcrypt memcached mysql opcache pgsql readline redis soap sqlite3 uploadprogress xdebug xhprof xml xmlrpc zip"
ENV php56_arm64="apcu bcmath bz2 curl cli common fpm gd imagick intl json ldap mbstring mcrypt mysql opcache pgsql readline soap sqlite3 uploadprogress xdebug xml xhprof xmlrpc zip"
ENV php70_amd64="apcu apcu-bc bcmath bz2 curl cli common fpm gd imagick intl json ldap mbstring mcrypt memcached mysql opcache pgsql readline redis soap sqlite3 uploadprogress xdebug xhprof xml xmlrpc zip"
ENV php70_arm64=$php70_amd64
ENV php71_amd64=$php70_amd64
ENV php71_arm64=$php70_arm64
ENV php72_amd64="apcu apcu-bc bcmath bz2 curl cli common fpm gd imagick intl json ldap mbstring memcached mysql opcache pgsql readline redis soap sqlite3 uploadprogress xdebug xhprof xml xmlrpc zip"
ENV php72_arm64=$php72_amd64
ENV php73_amd64=$php72_amd64
ENV php73_arm64=$php72_arm64
ENV php74_amd64="apcu apcu-bc bcmath bz2 curl cli common fpm gd imagick intl json ldap mbstring memcached mysql opcache pgsql readline redis soap sqlite3 uploadprogress xdebug xhprof xml xmlrpc zip"
ENV php74_arm64=$php74_amd64

# As of php8.0 json is now part of core package and xmlrpc has been removed from PECL
ENV php80_amd64="apcu bcmath bz2 curl cli common fpm gd imagick intl ldap mbstring memcached mysql opcache pgsql readline redis soap sqlite3 uploadprogress xdebug xhprof xml xmlrpc zip"
ENV php80_arm64=$php80_amd64

ENV php81_amd64=$php80_amd64
ENV php81_arm64=$php81_amd64
ENV php82_amd64=$php81_amd64
ENV php82_arm64=$php82_amd64
# php8.3 is still missing apcu, imagick, memcached, redis, uploadprogress, xdebug, xhprof, xmlrpc
ENV php83_amd64="bcmath bz2 curl cli common fpm gd intl ldap mbstring mysql opcache pgsql readline soap sqlite3 xml zip"
ENV php83_arm64=$php83_amd64

RUN for v in $PHP_VERSIONS; do \
    targetarch=${TARGETPLATFORM#linux/}; \
    pkgvar=${v//.}_${targetarch}; \
    pkgs=$(echo ${!pkgvar} | awk -v v="$v" ' BEGIN {RS=" "; }  { printf "%s-%s ",v,$0 ; }' ); \
    [[ ${pkgs// } != "" ]] && (apt-get -qq install --no-install-recommends --no-install-suggests -y $pkgs || exit $?) \
done
RUN phpdismod xhprof
RUN apt-get -qq autoremove -y
RUN curl -L --fail -o /usr/local/bin/composer -sSL https://getcomposer.org/composer-stable.phar && chmod ugo+wx /usr/local/bin/composer
RUN curl -L --fail -sSL "https://github.com/drush-ops/drush/releases/download/${DRUSH_VERSION}/drush.phar" -o /usr/local/bin/drush8 && chmod +x /usr/local/bin/drush8
RUN curl --fail -sSL -o /usr/local/bin/wp-cli -O https://raw.githubusercontent.com/wp-cli/builds/gh-pages/phar/wp-cli.phar && chmod +x /usr/local/bin/wp-cli && ln -sf /usr/local/bin/wp-cli /usr/local/bin/wp
RUN url="https://github.com/mikefarah/yq/releases/download/${YQ_VERSION}/yq_linux_${TARGETPLATFORM#linux/}"; wget ${url} -O /usr/bin/yq && chmod +x /usr/bin/yq
ADD ddev-php-files /
RUN apt-get -qq autoremove && apt-get -qq clean -y && rm -rf /var/lib/apt/lists/*
RUN	update-alternatives --set php /usr/bin/php${DDEV_PHP_VERSION}
RUN ln -sf /usr/sbin/php-fpm${DDEV_PHP_VERSION} /usr/sbin/php-fpm
RUN mkdir -p /run/php && chown -R www-data:www-data /run

#END ddev-php-base

### ---------------------------ddev-php-prod--------------------------------------
### Build ddev-php-prod from ddev-php-base as a single layer
### There aren't any differences
FROM scratch AS ddev-php-prod
COPY --from=ddev-php-base / /
EXPOSE 8080 8585
CMD ["/usr/sbin/php-fpm", "-F"]
#END ddev-php-prod
