### ---------------------------base--------------------------------------
### Build the base Debian image that will be used in every other image
FROM debian:bookworm-slim AS base
ARG TARGETPLATFORM
ARG BUILDPLATFORM

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
    lsb-release \
    procps \
    wget
RUN url="https://github.com/mikefarah/yq/releases/latest/download/yq_linux_${TARGETPLATFORM#linux/}"; wget ${url} -q -O /usr/bin/yq && chmod +x /usr/bin/yq
ADD generic-files /
#END base

### ---------------------------ddev-xdebug-build--------------------------------------
### Xdebug compile specific version because 3.3 is not fully reliable
### See https://github.com/ddev/ddev/issues/6159
### We don't need to recompile every Xdebug library https://xdebug.org/docs/compat (only PHP 8.0, 8.1, 8.2 can have Xdebug 3.2)
FROM base AS ddev-xdebug-build
SHELL ["/bin/bash", "-c"]
RUN curl -sSLo /tmp/debsuryorg-archive-keyring.deb https://packages.sury.org/debsuryorg-archive-keyring.deb && \
    dpkg -i /tmp/debsuryorg-archive-keyring.deb && rm -f /tmp/debsuryorg-archive-keyring.deb && \
    echo "deb [signed-by=/usr/share/keyrings/deb.sury.org-php.gpg] https://packages.sury.org/php/ $(lsb_release -sc) main" > /etc/apt/sources.list.d/php.list && apt-get update
RUN /usr/local/bin/build_php_extension.sh "8.1" "xdebug" "3.2.2" "/usr/lib/php/20210902/xdebug.so"
RUN /usr/local/bin/build_php_extension.sh "8.2" "xdebug" "3.2.2" "/usr/lib/php/20220829/xdebug.so"
#END ddev-xdebug-build

### ---------------------------ddev-php-base--------------------------------------
### Build ddev-php-base, which is the base for ddev-php-prod and ddev-webserver-*
### This combines the packages and features of DDEV’s ddev-webserver and PHP image
FROM base AS ddev-php-base
ARG PHP_DEFAULT_VERSION="8.2"
ENV DDEV_PHP_VERSION=$PHP_DEFAULT_VERSION
ARG PHP_VERSIONS="php8.1 php8.2 php8.3 php8.4"
ENV PHP_INI=/etc/php/$PHP_DEFAULT_VERSION/fpm/php.ini
ARG DRUSH_VERSION=8.4.12
ENV NODE_VERSION=20
# composer normally screams about running as root, we don't need that.
ENV COMPOSER_ALLOW_SUPERUSER=1
ENV COMPOSER_PROCESS_TIMEOUT=2000

# TARGETPLATFORM is Docker buildx's target platform (e.g. linux/arm64), while
# BUILDPLATFORM is the platform of the build host (e.g. linux/amd64)
ARG TARGETPLATFORM
ARG BUILDPLATFORM

SHELL ["/bin/bash", "-c"]

RUN curl https://nginx.org/keys/nginx_signing.key | gpg --dearmor \
    | tee /usr/share/keyrings/nginx-archive-keyring.gpg >/dev/null
RUN echo "deb [signed-by=/usr/share/keyrings/nginx-archive-keyring.gpg] \
http://nginx.org/packages/debian `lsb_release -cs` nginx" > /etc/apt/sources.list.d/nginx.list

RUN curl -sSLo /tmp/debsuryorg-archive-keyring.deb https://packages.sury.org/debsuryorg-archive-keyring.deb && \
    dpkg -i /tmp/debsuryorg-archive-keyring.deb && rm -f /tmp/debsuryorg-archive-keyring.deb && \
    echo "deb [signed-by=/usr/share/keyrings/deb.sury.org-php.gpg] https://packages.sury.org/php/ $(lsb_release -sc) main" > /etc/apt/sources.list.d/php.list && apt-get update

RUN apt-get -qq update
RUN apt-get -qq install --no-install-recommends --no-install-suggests -y \
    apache2 \
    file \
    ghostscript \
    imagemagick \
    gunicorn \
    graphicsmagick \
    jq \
    msmtp \
    nginx \
    sqlite3

RUN curl -L --fail -o /usr/local/bin/n -sSL https://raw.githubusercontent.com/tj/n/master/bin/n && chmod ugo+wx /usr/local/bin/n
# Install node without cache, make a symlink for nodejs
RUN n install --cleanup "${NODE_VERSION}" && ln -sf "$(which node)" "$(which node)js"
RUN npm install --unsafe-perm=true --global gulp-cli yarn
# Normal user needs to be able to write to php sessions
RUN set -eu -o pipefail && LATEST=$(curl -L --fail --silent "https://api.github.com/repos/nvm-sh/nvm/releases/latest" | jq -r .tag_name) && curl --fail -sL https://raw.githubusercontent.com/nvm-sh/nvm/${LATEST}/install.sh -o /usr/local/bin/install_nvm.sh && chmod +x /usr/local/bin/install_nvm.sh

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

### ---------------------------ddev-xdebug-build--------------------------------------
### The dates from /usr/lib/php/YYYYMMDD/ represent PHP API versions https://unix.stackexchange.com/a/591771
RUN apt-get -qq remove -y php8.1-xdebug php8.2-xdebug
COPY --from=ddev-xdebug-build /usr/lib/php/20210902/xdebug.so /usr/lib/php/20210902/xdebug.so
COPY --from=ddev-xdebug-build /usr/lib/php/20220829/xdebug.so /usr/lib/php/20220829/xdebug.so
#END ddev-xdebug-build
RUN phpdismod xhprof
RUN apt-get -qq autoremove -y
RUN curl -L --fail -o /usr/local/bin/composer -sSL https://getcomposer.org/composer-stable.phar && chmod ugo+wx /usr/local/bin/composer
RUN curl -L --fail -sSL "https://github.com/drush-ops/drush/releases/download/${DRUSH_VERSION}/drush.phar" -o /usr/local/bin/drush8 && chmod +x /usr/local/bin/drush8
RUN curl --fail -sSL -o /usr/local/bin/wp-cli -O https://raw.githubusercontent.com/wp-cli/builds/gh-pages/phar/wp-cli.phar && chmod +x /usr/local/bin/wp-cli && ln -sf /usr/local/bin/wp-cli /usr/local/bin/wp
ADD ddev-php-files /
RUN apt-get -qq autoremove && apt-get -qq clean -y && rm -rf /var/lib/apt/lists/* /tmp/*
RUN ln -sf /usr/sbin/php-fpm${DDEV_PHP_VERSION} /usr/sbin/php-fpm
RUN mkdir -p /run/php && chown -R www-data:www-data /run
RUN chmod 777 /var/lib/php/sessions
#END ddev-php-base

### ---------------------------ddev-php-prod--------------------------------------
### Build ddev-php-prod from ddev-php-base as a single layer
### There aren't any differences
FROM scratch AS ddev-php-prod
COPY --from=ddev-php-base / /
EXPOSE 8080 8585
CMD ["/usr/sbin/php-fpm", "-F"]
#END ddev-php-prod
