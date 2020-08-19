
### ---------------------------ddev-webserver-base--------------------------------------
### Build ddev-php-base from ddev-webserver-base
### ddev-php-base is the basic of ddev-php-prod (for DDEV-Live)
### and ddev-webserver-* (For DDEV-Local)
FROM dennisameling/ddev-php-base:20200817 as ddev-webserver-base

ENV PHP_VERSIONS="php5.6 php7.0 php7.1 php7.2 php7.3 php7.4"
ENV BACKDROP_DRUSH_VERSION=1.4.0
ENV MKCERT_VERSION=v1.4.1
ENV DEBIAN_FRONTEND=noninteractive

ENV NGINX_SITE_TEMPLATE /etc/nginx/nginx-site.conf
ENV APACHE_SITE_TEMPLATE /etc/apache2/apache-site.conf
ENV TERMINUS_CACHE_DIR=/mnt/ddev-global-cache/terminus/cache
ENV CAROOT /mnt/ddev-global-cache/mkcert

# TARGETPLATFORM is Docker buildx's target platform (e.g. linux/arm64), while 
# BUILDPLATFORM is the platform of the build host (e.g. linux/amd64)
ARG TARGETPLATFORM
ARG BUILDPLATFORM

RUN wget -q -O /tmp/nginx_signing.key http://nginx.org/keys/nginx_signing.key && \
    apt-key add /tmp/nginx_signing.key && \
    echo "deb http://nginx.org/packages/debian/ $(lsb_release -sc) nginx" > /etc/apt/sources.list.d/nginx.list

RUN apt-get -qq update
RUN DEBIAN_FRONTEND=noninteractive apt-get -qq install -y -o Dpkg::Options::="--force-confold" --no-install-recommends --no-install-suggests -y apache2 libcap2-bin locales-all nginx supervisor

# Arbitrary user needs to be able to bind to privileged ports (for nginx and apache2)
RUN setcap CAP_NET_BIND_SERVICE=+eip /usr/sbin/nginx
RUN setcap CAP_NET_BIND_SERVICE=+eip /usr/sbin/apache2

ADD ddev-webserver-base-files /
ADD ddev-webserver-base-scripts /
RUN apt-get -qq autoremove && apt-get -qq clean -y && rm -rf /var/lib/apt/lists/*

# END ddev-webserver-base

### ---------------------------ddev-webserver-prod--------------------------------------
### Build ddev-webserver-prod, the hardened version of ddev-webserver-base
### (Withut dev features, single layer)
#FROM scratch as ddev-webserver-prod
#ENV NGINX_SITE_TEMPLATE /etc/nginx/nginx-site.conf
#ENV APACHE_SITE_TEMPLATE /etc/apache2/apache-site.conf
#ENV TERMINUS_CACHE_DIR=/mnt/ddev-global-cache/terminus/cache
#ENV DRUSH_LAUNCHER_FALLBACK=/usr/local/bin/drush8
#
#COPY --from=ddev-webserver-base / /
# END ddev-webserver-prod

### ---------------------------ddev-webserver-dev-base--------------------------------------
### Build ddev-webserver-dev-base from ddev-webserver-base
FROM ddev-webserver-base as ddev-webserver-dev-base
ENV MAILHOG_VERSION=1.0.0
ENV CAROOT /mnt/ddev-global-cache/mkcert
ENV PHP_DEFAULT_VERSION="7.3"
ENV DDEV_LIVE_DOWNLOAD_URL https://downloads.ddev.com/ddev-live-cli/latest/linux/ddev-live.zip

RUN wget -q -O - https://packages.blackfire.io/gpg.key | apt-key add -
RUN echo "deb http://packages.blackfire.io/debian any main" > /etc/apt/sources.list.d/blackfire.list
RUN apt-get update

SHELL ["/bin/bash", "-c"]
# blackfire-php is not available for arm64 (yet)
RUN if [[ $TARGETPLATFORM != "linux/arm64" ]]; then \
    DEBIAN_FRONTEND=noninteractive apt-get install -o Dpkg::Options::="--force-confold" blackfire-php -y --allow-unauthenticated; \
    fi
RUN DEBIAN_FRONTEND=noninteractive apt-get -qq install -o Dpkg::Options::="--force-confold" --no-install-recommends --no-install-suggests -y \
    fontconfig \
    gettext \
    git \
    iproute2 \
    iputils-ping \
    jq \
    libpcre3 \
    locales-all \
    nano \
    ncurses-bin \
    netcat \
    openssh-client \
    patch \
    rsync \
    sqlite3 \
    sudo \
    telnet \
    unzip \
    zip

ADD ddev-webserver-base-files /
RUN if [[ $TARGETPLATFORM == "linux/arm64" ]]; then \
        # Issue for official arm64 support: https://github.com/mailhog/MailHog/issues/307
        curl -sSL "https://github.com/mailhog/MailHog/releases/download/v${MAILHOG_VERSION}-ddev/MailHog_linux_arm64" -o /usr/local/bin/mailhog; \
    else \
        curl -sSL "https://github.com/mailhog/MailHog/releases/download/v${MAILHOG_VERSION}/MailHog_linux_amd64" -o /usr/local/bin/mailhog; \
    fi

RUN curl -sSL https://github.com/pantheon-systems/terminus/releases/download/$(curl --silent "https://api.github.com/repos/pantheon-systems/terminus/releases/latest" | perl -nle'print $& while m{"tag_name": "\K.*?(?=")}g')/terminus.phar --output /usr/local/bin/terminus && chmod +x /usr/local/bin/terminus
RUN curl -sSL -O $DDEV_LIVE_DOWNLOAD_URL && unzip ddev-live.zip && mv ddev-live /usr/local/bin && chmod +x /usr/local/bin/ddev-live && rm ddev-live.zip

# magerun and magerun2 for magento
RUN curl -sSL https://files.magerun.net/n98-magerun-latest.phar -o /usr/local/bin/magerun
RUN curl -sSL https://raw.githubusercontent.com/netz98/n98-magerun/${MAGERUN_VERSION}/res/autocompletion/bash/n98-magerun.phar.bash -o /etc/bash_completion.d/magerun
RUN curl -sSL https://files.magerun.net/n98-magerun2-latest.phar -o /usr/local/bin/magerun2
RUN curl -sSL https://raw.githubusercontent.com/netz98/n98-magerun2/${MAGERUN2_VERSION}/res/autocompletion/bash/n98-magerun2.phar.bash -o /etc/bash_completion.d/magerun2

RUN curl -sSL https://github.com/backdrop-contrib/drush/releases/download/${BACKDROP_DRUSH_VERSION}/backdrop-drush-extension.zip -o /tmp/backdrop-drush-extension.zip && unzip -o /tmp/backdrop-drush-extension.zip -d /var/tmp/backdrop_drush_commands

RUN mkdir -p /etc/nginx/sites-enabled /var/log/apache2 /var/run/apache2 /var/lib/apache2/module/enabled_by_admin /var/lib/apache2/module/disabled_by_admin && \
    touch /var/log/php-fpm.log && \
    chmod ugo+rw /var/log/php-fpm.log && \
    chmod ugo+rwx /var/run && \
    touch /var/log/nginx/access.log && \
    touch /var/log/nginx/error.log && \
    chmod -R ugo+rw /var/log/nginx/ && \
    chmod ugo+rx /usr/local/bin/* && \
    update-alternatives --set php /usr/bin/php${PHP_DEFAULT_VERSION} && \
    ln -sf /usr/sbin/php-fpm${PHP_DEFAULT_VERSION} /usr/sbin/php-fpm

RUN chmod -R 777 /var/log

# /home is a prototype for the actual user dir, but leave it writable
RUN mkdir -p /home/.composer /home/.drush/commands /home/.drush/aliases /mnt/ddev-global-cache/mkcert /run/php /var/cache/nginx && chmod -R ugo+rw /home /mnt/ddev-global-cache/

RUN chmod -R ugo+w /usr/sbin /usr/bin /etc/nginx /var/cache/nginx /run /var/www /etc/php/*/*/conf.d/ /var/lib/php/modules /etc/alternatives /usr/lib/node_modules /etc/php /etc/apache2 /var/log/apache2/ /var/run/apache2 /var/lib/apache2 /mnt/ddev-global-cache/*

RUN if [[ $TARGETPLATFORM == "linux/arm64" ]]; then \
        # PR for official arm64 support pending: https://github.com/FiloSottile/mkcert/pull/284
        curl -sSL https://github.com/dennisameling/mkcert/releases/download/$MKCERT_VERSION-ddev/mkcert-$MKCERT_VERSION-linux-arm64 -o /usr/local/bin/mkcert && chmod +x /usr/local/bin/mkcert; \
    else \
        curl -sSL https://github.com/FiloSottile/mkcert/releases/download/$MKCERT_VERSION/mkcert-$MKCERT_VERSION-linux-amd64 -o /usr/local/bin/mkcert && chmod +x /usr/local/bin/mkcert; \
    fi

# Except that .my.cnf can't be writeable or mysql won't use it.
RUN chmod 644 /home/.my.cnf

RUN touch /var/log/nginx/error.log /var/log/nginx/access.log /var/log/php-fpm.log && \
  chmod 666 /var/log/nginx/error.log /var/log/nginx/access.log /var/log/php-fpm.log

RUN a2dismod mpm_event
RUN a2enmod ssl headers expires

# ssh is very particular about permissions in ~/.ssh
RUN chmod -R go-w /home/.ssh

# scripts added last because they're most likely place to make changes, speeds up build
ADD ddev-webserver-base-scripts /
RUN chmod ugo+x /start.sh /healthcheck.sh

RUN addgroup --gid 98 testgroup && adduser testuser --ingroup testgroup --disabled-password --gecos "" --uid 98

EXPOSE 80 443 8025
HEALTHCHECK --interval=1s --retries=10 --timeout=120s --start-period=10s CMD ["/healthcheck.sh"]
CMD ["/start.sh"]
RUN apt-get -qq clean -y && rm -rf /var/lib/apt/lists/*
#END ddev-webserver-dev-base

### ---------------------------ddev-webserver--------------------------------------
### This could be known as ddev-webserver-dev as it's development-env targeted
### But for historical reasons, it's just ddev-webserver
### Build ddev-webserver by turning ddev-webserver-dev-base into one layer
FROM scratch as ddev-webserver
ENV PHP_DEFAULT_VERSION="7.3"
ENV NGINX_SITE_TEMPLATE /etc/nginx/nginx-site.conf
ENV APACHE_SITE_TEMPLATE /etc/apache2/apache-site.conf
ENV TERMINUS_CACHE_DIR=/mnt/ddev-global-cache/terminus/cache
ENV CAROOT /mnt/ddev-global-cache/mkcert
ENV DRUSH_LAUNCHER_FALLBACK=/usr/local/bin/drush8

ENV DDEV_LIVE_CONFIG_FILE_PATH /mnt/ddev-global-cache/ddev-live/cli-config.json
ENV DDEV_LIVE_NO_VERSION_PROMPT true

ENV COMPOSER_ALLOW_SUPERUSER=1
ENV COMPOSER_CACHE_DIR=/mnt/ddev-global-cache/composer
ENV COMPOSER_PROCESS_TIMEOUT=2000
ENV DEBIAN_FRONTEND noninteractive
ENV TERM xterm
ENV MH_SMTP_BIND_ADDR 127.0.0.1:1025

ENV BASH_ENV /etc/bash.nointeractive.bashrc

COPY --from=ddev-webserver-dev-base / /
EXPOSE 80 8025
HEALTHCHECK --interval=1s --retries=10 --timeout=120s --start-period=10s CMD ["/healthcheck.sh"]
CMD ["/start.sh"]
#END ddev-webserver

