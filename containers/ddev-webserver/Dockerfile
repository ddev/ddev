
### ---------------------------ddev-webserver-base--------------------------------------
### Build ddev-php-base from ddev-webserver-base
### ddev-php-base is the basic of ddev-php-prod
### and ddev-webserver-* (For DDEV-Local)
FROM ddev/ddev-php-base:20230720_ted933_doc_installation_ubuntu_apt_vulnerability as ddev-webserver-base

ENV BACKDROP_DRUSH_VERSION=1.4.0
ENV DEBIAN_FRONTEND=noninteractive

ENV NGINX_SITE_TEMPLATE /etc/nginx/nginx-site.conf
ENV APACHE_SITE_TEMPLATE /etc/apache2/apache-site.conf
ENV TERMINUS_CACHE_DIR=/mnt/ddev-global-cache/terminus/cache
ENV CAROOT /mnt/ddev-global-cache/mkcert

# TARGETPLATFORM is Docker buildx's target platform (e.g. linux/arm64), while
# BUILDPLATFORM is the platform of the build host (e.g. linux/amd64)
ARG TARGETPLATFORM
ARG BUILDPLATFORM

ADD ddev-webserver-etc-skel /
RUN /sbin/mkhomedir_helper www-data

RUN apt-get -qq update
RUN DEBIAN_FRONTEND=noninteractive apt-get -qq install -y -o Dpkg::Options::="--force-confold" --no-install-recommends --no-install-suggests -y libcap2-bin locales-all supervisor

# Arbitrary user needs to be able to bind to privileged ports (for nginx and apache2)
RUN setcap CAP_NET_BIND_SERVICE=+eip /usr/sbin/nginx
RUN setcap CAP_NET_BIND_SERVICE=+eip /usr/sbin/apache2

# magerun and magerun2 for magento
RUN curl --fail -sSL https://files.magerun.net/n98-magerun-latest.phar -o /usr/local/bin/magerun && chmod 777 /usr/local/bin/magerun
RUN curl --fail -sSL https://raw.githubusercontent.com/netz98/n98-magerun/develop/res/autocompletion/bash/n98-magerun.phar.bash -o /etc/bash_completion.d/n98-magerun.phar
RUN curl --fail -sSL https://files.magerun.net/n98-magerun2-latest.phar -o /usr/local/bin/magerun2 && chmod 777 /usr/local/bin/magerun2
RUN curl --fail -sSL https://raw.githubusercontent.com/netz98/n98-magerun2/develop/res/autocompletion/bash/n98-magerun2.phar.bash -o /etc/bash_completion.d/n98-magerun2.phar && chmod +x /usr/local/bin/magerun

RUN apt-get -qq autoremove && apt-get -qq clean -y && rm -rf /var/lib/apt/lists/*

ADD ddev-webserver-base-files /
ADD ddev-webserver-base-scripts /

# /usr/local/bin may need to be updated by start.sh, etc
RUN chmod -f ugo+rwx /usr/local/bin /usr/local/bin/composer
# END ddev-webserver-base


### ---------------------------ddev-webserver-dev-base--------------------------------------
### Build ddev-webserver-dev-base from ddev-webserver-base
FROM ddev-webserver-base as ddev-webserver-dev-base
ENV MAILHOG_VERSION="1.0.2"
ENV CAROOT /mnt/ddev-global-cache/mkcert
ENV PHP_DEFAULT_VERSION="8.1"

RUN wget -q -O - https://packages.blackfire.io/gpg.key | apt-key add -
RUN echo "deb http://packages.blackfire.io/debian any main" > /etc/apt/sources.list.d/blackfire.list
RUN apt-get update

SHELL ["/bin/bash", "-c"]

RUN DEBIAN_FRONTEND=noninteractive apt-get -qq install -o Dpkg::Options::="--force-confold" --no-install-recommends --no-install-suggests -y \
    blackfire \
    blackfire-php \
    fontconfig \
    gettext \
    git \
    graphviz \
    iproute2 \
    iputils-ping \
    jq \
    libldap-common \
    libpcre3 \
    libpq-dev \
    libpython3-dev \
    locales-all \
    nano \
    ncurses-bin \
    netcat \
    openssh-client \
    patch \
    python-is-python3 \
    python3-pip \
    python3-psycopg2 \
    python3-venv \
    rsync \
    sqlite3 \
    sudo \
    telnet \
    unzip \
    zip

RUN curl --fail -JL -s -o /usr/local/bin/mkcert "https://dl.filippo.io/mkcert/latest?for=${TARGETPLATFORM}" && chmod +x /usr/local/bin/mkcert

# blackfire user by default is set up with /dev/null as homedir, and 999 as uid, which
# can break people. Use a real homedir
RUN mkdir -p /home/blackfire && chown blackfire:blackfire /home/blackfire && usermod -d /home/blackfire blackfire

ADD ddev-webserver-dev-base-files /
RUN phpdismod blackfire xdebug xhprof

RUN set -x; curl --fail -sSL "https://github.com/ddev/MailHog/releases/download/v${MAILHOG_VERSION}/MailHog_linux_${TARGETPLATFORM##linux/}" -o /usr/local/bin/mailhog && chmod +x /usr/local/bin/mailhog

RUN curl -sSL --fail --output /usr/local/bin/phive "https://phar.io/releases/phive.phar" && chmod 777 /usr/local/bin/phive
RUN set -o pipefail && curl --fail -sSL https://github.com/pantheon-systems/terminus/releases/download/$(curl -L --fail --silent "https://api.github.com/repos/pantheon-systems/terminus/releases/latest" | perl -nle'print $& while m{"tag_name": "\K.*?(?=")}g')/terminus.phar --output /usr/local/bin/terminus && chmod 777 /usr/local/bin/terminus
RUN set -o pipefail && curl --fail -sSL https://github.com/platformsh/platformsh-cli/releases/download/$(curl -L --fail --silent "https://api.github.com/repositories/16695539/releases/latest" | perl -nle'print $& while m{"tag_name": "\K.*?(?=")}g')/platform.phar --output /usr/local/bin/platform && chmod 777 /usr/local/bin/platform

RUN mkdir -p "/opt/phpstorm-coverage" && \
    chmod a+rw "/opt/phpstorm-coverage"

RUN curl --fail -sSL --output /usr/local/bin/acli https://github.com/acquia/cli/releases/latest/download/acli.phar && chmod 777 /usr/local/bin/acli

RUN curl --fail -sSL https://github.com/backdrop-contrib/drush/releases/download/${BACKDROP_DRUSH_VERSION}/backdrop-drush-extension.zip -o /tmp/backdrop-drush-extension.zip && unzip -o /tmp/backdrop-drush-extension.zip -d /var/tmp/backdrop_drush_commands && chmod -R ugo+w /var/tmp/backdrop_drush_commands && rm /tmp/backdrop-drush-extension.zip

RUN mkdir -p /etc/nginx/sites-enabled /var/log/apache2 /var/run/apache2 /var/lib/apache2/module/enabled_by_admin /var/lib/apache2/module/disabled_by_admin && \
    touch /var/log/php-fpm.log && \
    chmod ugo+rw /var/log/php-fpm.log && \
    chmod ugo+rwx /var/run && \
    touch /var/log/nginx/access.log && \
    touch /var/log/nginx/error.log && \
    chmod -R ugo+rw /var/log/nginx/ && \
    chmod ugo+rwx /usr/local/bin/* && \
    update-alternatives --set php /usr/bin/php${PHP_DEFAULT_VERSION} && \
    ln -sf /usr/sbin/php-fpm${PHP_DEFAULT_VERSION} /usr/sbin/php-fpm

RUN chmod -R 777 /var/log

# we need to create the /var/cache/linux and /var/lib/nginx manually for the arm64 image and chmod them, please don't remove them!
RUN mkdir -p /mnt/ddev-global-cache/mkcert /run/{php,blackfire} /var/cache/nginx /var/lib/nginx && chmod -R ugo+rw /mnt/ddev-global-cache/

RUN chmod -R ugo+w /usr/sbin /usr/bin /etc/nginx /var/cache/nginx /var/lib/nginx /run /var/www /etc/php/*/*/conf.d/ /var/lib/php/modules /etc/alternatives /usr/lib/node_modules /etc/php /etc/apache2 /var/log/apache2/ /var/run/apache2 /var/lib/apache2 /mnt/ddev-global-cache/*

RUN mkdir -p /var/xhprof && curl --fail  -o /tmp/xhprof.tgz -sSL https://pecl.php.net/get/xhprof && tar -zxf /tmp/xhprof.tgz --strip-components=1 -C /var/xhprof && chmod 777 /var/xhprof/xhprof_html && rm /tmp/xhprof.tgz

RUN touch /var/log/nginx/error.log /var/log/nginx/access.log /var/log/php-fpm.log && \
  chmod 666 /var/log/nginx/error.log /var/log/nginx/access.log /var/log/php-fpm.log

RUN a2dismod mpm_event
RUN a2enmod ssl headers expires

# scripts added last because they're most likely place to make changes, speeds up build
ADD ddev-webserver-base-scripts /
RUN chmod ugo+x /start.sh /healthcheck.sh

# Composer, etc may need to be updated by composer self-update
RUN chmod -f ugo+rwx /usr/local/bin /usr/local/bin/*

RUN chmod ugo+w /etc/ssl/certs /usr/local/share/ca-certificates

HEALTHCHECK --interval=1s --retries=120 --timeout=120s --start-period=120s CMD ["/healthcheck.sh"]
CMD ["/start.sh"]
RUN apt-get -qq clean -y && rm -rf /var/lib/apt/lists/*
#END ddev-webserver-dev-base

### ---------------------------ddev-webserver--------------------------------------
### This could be known as ddev-webserver-dev as it's development-env targeted
### But for historical reasons, it's just ddev-webserver
### Build ddev-webserver by turning ddev-webserver-dev-base into one layer
FROM scratch as ddev-webserver
ENV PHP_DEFAULT_VERSION="8.1"
ENV NGINX_SITE_TEMPLATE /etc/nginx/nginx-site.conf
ENV APACHE_SITE_TEMPLATE /etc/apache2/apache-site.conf
ENV TERMINUS_CACHE_DIR=/mnt/ddev-global-cache/terminus/cache
ENV TERMINUS_HIDE_UPDATE_MESSAGE=1
ENV CAROOT /mnt/ddev-global-cache/mkcert
ENV COMPOSER_ALLOW_SUPERUSER=1
ENV COMPOSER_CACHE_DIR=/mnt/ddev-global-cache/composer
ENV COMPOSER_PROCESS_TIMEOUT=2000
ENV DEBIAN_FRONTEND noninteractive
ENV TERM xterm
ENV MH_SMTP_BIND_ADDR 127.0.0.1:1025
ENV BASH_ENV /etc/bash.nointeractive.bashrc
ENV LANG=C.UTF-8
ENV XHPROF_OUTPUT_DIR=/tmp/xhprof
ENV PLATFORMSH_CLI_UPDATES_CHECK=0

COPY --from=ddev-webserver-dev-base / /
EXPOSE 80 8025
HEALTHCHECK --interval=1s --retries=120 --timeout=120s --start-period=120s CMD ["/healthcheck.sh"]
CMD ["/start.sh"]
#END ddev-webserver

### ---------------------------ddev-webserver-prod-base--------------------------------------
### Build ddev-webserver-prod-base from ddev-webserver-base
### This image is aimed at actual hardened production environments
FROM ddev-webserver-base as ddev-webserver-prod-base
ENV CAROOT /mnt/ddev-global-cache/mkcert
ENV PHP_DEFAULT_VERSION="8.1"
ARG TARGETPLATFORM

RUN wget -q -O - https://packages.blackfire.io/gpg.key | apt-key add -
RUN echo "deb http://packages.blackfire.io/debian any main" > /etc/apt/sources.list.d/blackfire.list
RUN apt-get update

SHELL ["/bin/bash", "-c"]

RUN DEBIAN_FRONTEND=noninteractive apt-get -qq install -o Dpkg::Options::="--force-confold" --no-install-recommends --no-install-suggests -y \
    blackfire-php \
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
    unzip \
    zip

RUN curl --fail -JL -s -o /usr/local/bin/mkcert "https://dl.filippo.io/mkcert/latest?for=${TARGETPLATFORM}" && chmod +x /usr/local/bin/mkcert

ADD ddev-webserver-prod-files /
RUN phpdismod blackfire xhprof

RUN curl --fail -sSL https://github.com/backdrop-contrib/drush/releases/download/${BACKDROP_DRUSH_VERSION}/backdrop-drush-extension.zip -o /tmp/backdrop-drush-extension.zip && unzip -o /tmp/backdrop-drush-extension.zip -d /var/tmp/backdrop_drush_commands && chmod -R ugo+w /var/tmp/backdrop_drush_commands && rm /tmp/backdrop-drush-extension.zip

RUN mkdir -p /etc/nginx/sites-enabled /var/lock/apache2 /var/log/apache2 /var/run/apache2 /var/lib/apache2/module/enabled_by_admin /var/lib/apache2/module/disabled_by_admin && \
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

# we need to create the /var/cache/linux and /var/lib/nginx manually for the arm64 image and chmod them, please don't remove them!
RUN mkdir -p /mnt/ddev-global-cache/mkcert /run/php /var/cache/nginx /var/lib/nginx && chmod -R ugo+rw /home /mnt/ddev-global-cache/

RUN chmod -R ugo+w /usr/sbin /usr/bin /etc/nginx /var/cache/nginx /var/lib/nginx /run /var/www /etc/php/*/*/conf.d/ /var/lib/php/modules /etc/alternatives /usr/lib/node_modules /etc/php /etc/apache2 /var/lock/apache2 /var/log/apache2/ /var/run/apache2 /var/lib/apache2 /mnt/ddev-global-cache/*

RUN touch /var/log/nginx/error.log /var/log/nginx/access.log /var/log/php-fpm.log && \
  chmod 666 /var/log/nginx/error.log /var/log/nginx/access.log /var/log/php-fpm.log

RUN a2dismod mpm_event
RUN a2enmod ssl headers expires

# scripts added last because they're most likely place to make changes, speeds up build
ADD ddev-webserver-prod-scripts /
RUN chmod ugo+x /start.sh /healthcheck.sh

RUN /sbin/mkhomedir_helper www-data

RUN chmod ugo+w /etc/ssl/certs /usr/local/share/ca-certificates

HEALTHCHECK --interval=1s --retries=120 --timeout=120s --start-period=120s CMD ["/healthcheck.sh"]
CMD ["/start.sh"]
RUN apt-get -qq clean -y && rm -rf /var/lib/apt/lists/*

#END ddev-webserver-prod-base

### ---------------------------ddev-webserver-prod--------------------------------------
### Build ddev-webserver-prod, the hardened version of ddev-webserver-base
### (Withut dev features, single layer)
FROM scratch as ddev-webserver-prod
ENV PHP_DEFAULT_VERSION="8.1"
ENV NGINX_SITE_TEMPLATE /etc/nginx/nginx-site.conf
ENV APACHE_SITE_TEMPLATE /etc/apache2/apache-site.conf
ENV TERMINUS_CACHE_DIR=/mnt/ddev-global-cache/terminus/cache
ENV TERMINUS_HIDE_UPDATE_MESSAGE=1
ENV CAROOT /mnt/ddev-global-cache/mkcert
ENV COMPOSER_ALLOW_SUPERUSER=1
ENV COMPOSER_CACHE_DIR=/mnt/ddev-global-cache/composer
ENV COMPOSER_PROCESS_TIMEOUT=2000
ENV DEBIAN_FRONTEND noninteractive
ENV LANG=C.UTF-8
ENV TERM xterm
ENV BASH_ENV /etc/bash.nointeractive.bashrc
ENV PLATFORMSH_CLI_UPDATES_CHECK=0

COPY --from=ddev-webserver-prod-base / /
HEALTHCHECK --interval=1s --retries=120 --timeout=120s --start-period=120s CMD ["/healthcheck.sh"]
CMD ["/start.sh"]
#END ddev-webserver-prod
