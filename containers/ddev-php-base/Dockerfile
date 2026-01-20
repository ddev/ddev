### ---------------------------------base-----------------------------------------
### Build the base Debian image that will be used in every other image
FROM debian:trixie-slim AS base
ARG TARGETPLATFORM
ARG BUILDPLATFORM
SHELL ["/bin/bash", "-eu","-o", "pipefail",  "-c"]

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

# tzdata-legacy - to avoid issues with deprecated timezones
RUN apt-get -qq update && apt-get -qq install --no-install-recommends --no-install-suggests -y \
    apt-transport-https \
    ca-certificates \
    bzip2 \
    curl \
    git \
    gnupg \
    lsb-release \
    procps \
    tzdata-legacy \
    wget
RUN url="https://github.com/mikefarah/yq/releases/latest/download/yq_linux_${TARGETPLATFORM#linux/}"; wget ${url} -q -O /usr/bin/yq && chmod +x /usr/bin/yq

# Install Xdebug dbgpClient and xdebugctl tools
# dbgpClient - Command-line DBGp client for testing Xdebug connections
# xdebugctl - Tool to control Xdebug settings at runtime
RUN ARCH="${TARGETPLATFORM#linux/}"; \
    if [ "$ARCH" = "amd64" ]; then ARCH=""; else ARCH="-${ARCH}"; fi; \
    wget -q https://xdebug.org/files/binaries/xdebugctl${ARCH} -O /usr/local/bin/xdebugctl && \
    chmod +x /usr/local/bin/xdebugctl

ADD generic-files /

# This curl -I is debugging for intermittent failures on https://github.com/oerdnj/deb.sury.org/issues/2046
RUN curl -s -I https://packages.sury.org/php/

RUN curl -sSLo /tmp/debsuryorg-archive-keyring.deb https://packages.sury.org/debsuryorg-archive-keyring.deb && \
    dpkg -i /tmp/debsuryorg-archive-keyring.deb
RUN rm -f /tmp/debsuryorg-archive-keyring.deb && \
    printf "Types: deb\nURIs: https://packages.sury.org/php/\nSuites: $(lsb_release -sc)\nComponents: main\nSigned-By: /usr/share/keyrings/deb.sury.org-php.gpg\n" > /etc/apt/sources.list.d/php.sources
RUN apt-get -qq update
### -------------------------------END base---------------------------------------

### ------------------------ddev-php-extension-build------------------------------
### Uncomment the lines below to enable custom PHP extension builds:
#FROM base AS ddev-php-extension-build
#SHELL ["/bin/bash", "-eu","-o", "pipefail",  "-c"]
### Use this section to add PHP extensions that need to be built manually.
### Example: The following line installs Xdebug version 3.2.2 for PHP 8.1:
#RUN /usr/local/bin/build_php_extension.sh "php8.1" "xdebug" "3.2.2" "/usr/lib/php/20210902/xdebug.so"
### -----------------------END ddev-php-extension-build---------------------------

### ---------------------------ddev-php-base--------------------------------------
### Build ddev-php-base, which is the base for ddev-php-prod and ddev-webserver-*
### This combines the packages and features of DDEV’s ddev-webserver and PHP image
FROM base AS ddev-php-base
SHELL ["/bin/bash", "-eu","-o", "pipefail",  "-c"]
ARG PHP_DEFAULT_VERSION="8.4"
ARG XDEBUG_MODE=off
ENV DDEV_PHP_VERSION=$PHP_DEFAULT_VERSION
ARG PHP_VERSIONS="php8.2 php8.3 php8.4 php8.5"
ENV PHP_INI=/etc/php/$PHP_DEFAULT_VERSION/fpm/php.ini
ARG DRUSH_VERSION=8.5.0
ENV NODE_VERSION=24
# composer normally screams about running as root, we don't need that.
ENV COMPOSER_ALLOW_SUPERUSER=1
ENV COMPOSER_PROCESS_TIMEOUT=2000

# TARGETPLATFORM is Docker buildx's target platform (e.g. linux/arm64), while
# BUILDPLATFORM is the platform of the build host (e.g. linux/amd64)
ARG TARGETPLATFORM
ARG BUILDPLATFORM

RUN apt-get -qq update && apt-get -qq install --no-install-recommends --no-install-suggests -y \
    apache2 \
    file \
    ghostscript \
    imagemagick \
    graphicsmagick \
    jq \
    msmtp \
    nginx \
    sqlite3

RUN curl -L --fail -o /usr/local/bin/n -sSL https://raw.githubusercontent.com/tj/n/master/bin/n && chmod ugo+wx /usr/local/bin/n
# Install node without cache, make a symlink for nodejs
RUN n install --cleanup "${NODE_VERSION}" && ln -sf "$(which node)" "$(which node)js"
RUN npm install --unsafe-perm=true --global gulp-cli yarn

# Loop through $PHP_VERSIONS, selecting packages for the target architecture.
RUN for v in ${PHP_VERSIONS}; do \
    /usr/local/bin/install_php_extensions.sh "$v" "${TARGETPLATFORM#linux/}"; \
done
RUN update-alternatives --set php /usr/bin/php${PHP_DEFAULT_VERSION}
RUN mkdir -p /etc/nginx/sites-enabled /var/lock/apache2 /var/log/apache2 /var/run/apache2 /var/lib/apache2/module/enabled_by_admin /var/lib/apache2/module/disabled_by_admin && \
    touch /var/log/php-fpm.log && \
    chmod ugo+rw /var/log/php-fpm.log && \
    chmod ugo+rwx /var/run && \
    touch /var/log/nginx/access.log && \
    touch /var/log/nginx/error.log && \
    chmod -R ugo+rw /var/log/nginx/ && \
    chmod ugo+rx /usr/local/bin/* && \
    ln -sf /usr/sbin/php-fpm${PHP_DEFAULT_VERSION} /usr/sbin/php-fpm

### ------------------------ddev-php-extension-build------------------------------
### Remove any existing PHP extensions that will be replaced by custom-built versions:
#RUN apt-get -qq remove -y php8.1-xdebug
### After building the custom extensions, copy them from the multi-stage build:
### Note: The dates in /usr/lib/php/YYYYMMDD/ represent PHP API versions.
### For more info, see: https://unix.stackexchange.com/a/591771
#COPY --from=ddev-php-extension-build /usr/lib/php/20210902/xdebug.so /usr/lib/php/20210902/xdebug.so
### -----------------------END ddev-php-extension-build---------------------------

RUN phpdismod xhprof xdebug
RUN apt-get -qq autoremove -y
RUN curl -L --fail -o /usr/local/bin/composer -sSL https://getcomposer.org/composer-stable.phar && chmod ugo+wx /usr/local/bin/composer
RUN <<ENDDRUSH
    mkdir -p /usr/local/src/drush
    curl -sSfL -o /tmp/drush.tgz https://github.com/drush-ops/drush/archive/refs/tags/${DRUSH_VERSION}.tar.gz
    tar -C /usr/local/src/drush --strip-components=1 -zxf /tmp/drush.tgz
    pushd /usr/local/src/drush >/dev/null
    composer install
    ln -s /usr/local/src/drush/drush /usr/local/bin/drush8
    popd >/dev/null
ENDDRUSH
RUN curl --fail -sSL -o /usr/local/bin/wp-cli -O https://raw.githubusercontent.com/wp-cli/builds/gh-pages/phar/wp-cli.phar && chmod +x /usr/local/bin/wp-cli && ln -sf /usr/local/bin/wp-cli /usr/local/bin/wp
ADD ddev-php-files /
RUN apt-get -qq autoremove && apt-get -qq clean -y && rm -rf /var/lib/apt/lists/* /tmp/*
RUN ln -sf /usr/sbin/php-fpm${DDEV_PHP_VERSION} /usr/sbin/php-fpm
RUN mkdir -p /run/php && chown -R www-data:www-data /run
RUN chmod 777 /var/lib/php/sessions
### -------------------------END ddev-php-base------------------------------------

### ---------------------------ddev-php-prod--------------------------------------
### Build ddev-php-prod from ddev-php-base as a single layer
### There aren't any differences
FROM scratch AS ddev-php-prod
ARG XDEBUG_MODE=off
SHELL ["/bin/bash", "-eu","-o", "pipefail",  "-c"]
COPY --from=ddev-php-base / /
EXPOSE 8080 8585
CMD ["/usr/sbin/php-fpm", "-F"]
### -------------------------END ddev-php-prod------------------------------------
