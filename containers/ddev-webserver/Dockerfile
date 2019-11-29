FROM bitnami/minideb:stretch

ENV PHP_VERSIONS="php5.6 php7.0 php7.1 php7.2 php7.3 php7.4"
ENV PHP_DEFAULT_VERSION="7.2"
ENV PHP_INI=/etc/php/$PHP_DEFAULT_VERSION/fpm/php.ini

ENV DRUSH_VERSION=8.3.1
ENV DRUSH_LAUNCHER_VERSION=0.6.0
ENV DRUSH_LAUNCHER_FALLBACK=/usr/local/bin/drush8
ENV MAILHOG_VERSION=1.0.0
ENV BACKDROP_DRUSH_VERSION=1.0.0
ENV MKCERT_VERSION=v1.4.0

ENV DEBIAN_FRONTEND noninteractive
ENV TERM xterm
ENV MH_SMTP_BIND_ADDR 127.0.0.1:1025
ENV NGINX_SITE_TEMPLATE /etc/nginx/nginx-site.conf
ENV APACHE_SITE_TEMPLATE /etc/apache2/apache-site.conf
ENV WEBSERVER_DOCROOT /var/www/html
# For backward compatibility only
ENV NGINX_DOCROOT $WEBSERVER_DOCROOT

# composer normally screams about running as root, we don't need that.
ENV COMPOSER_ALLOW_SUPERUSER 1
ENV COMPOSER_CACHE_DIR /mnt/ddev-global-cache/composer
# Windows, especially Win10 Home/Docker toolbox, can take forever on composer build.
ENV COMPOSER_PROCESS_TIMEOUT 2000

# Defines vars in colon-separated notation to be subsituted with values for NGINX_SITE_TEMPLATE on start
# NGINX_DOCROOT is for backward compatibility only, to break less people.
ENV NGINX_SITE_VARS '$WEBSERVER_DOCROOT,$NGINX_DOCROOT'
ENV APACHE_SITE_VARS '$WEBSERVER_DOCROOT'

ENV CAROOT /mnt/ddev-global-cache/mkcert


RUN ln -fs /usr/share/zoneinfo/UTC /etc/localtime && dpkg-reconfigure --frontend noninteractive tzdata

RUN apt-get -qq update && \
    apt-get -qq install --no-install-recommends --no-install-suggests -y \
        procps \
        curl \
        ca-certificates \
        apt-transport-https \
        wget \
        fontconfig \
        bzip2 \
        ghostscript \
        gnupg \
        locales-all && \
    wget -O /etc/apt/trusted.gpg.d/php.gpg https://packages.sury.org/php/apt.gpg && \
    echo "deb https://packages.sury.org/php/ stretch main" > /etc/apt/sources.list.d/php.list && \
    wget -q -O - https://packages.blackfire.io/gpg.key | apt-key add - && \
    echo "deb http://packages.blackfire.io/debian any main" > /etc/apt/sources.list.d/blackfire.list && \
    wget -q -O /tmp/nginx_signing.key http://nginx.org/keys/nginx_signing.key && \
    apt-key add /tmp/nginx_signing.key && \
    curl -sS https://dl.yarnpkg.com/debian/pubkey.gpg | apt-key add - && \
    echo "deb https://dl.yarnpkg.com/debian/ stable main" | tee /etc/apt/sources.list.d/yarn.list && \
    echo "deb http://nginx.org/packages/debian/ stretch nginx" >> /etc/apt/sources.list && \
    curl -sL https://deb.nodesource.com/setup_12.x | bash - && \
    apt-get -qq update && \
    apt-get -qq install --no-install-recommends --no-install-suggests -y \
        less \
        git \
        mysql-client \
        nginx \
        apache2 \
        nodejs \
        libcap2-bin \
        supervisor \
        sudo \
        imagemagick \
        iputils-ping \
        patch \
        telnet \
        netcat \
        iproute2 \
        vim \
        nano \
        gettext \
        ncurses-bin \
        yarn \
        zip \
        unzip \
        rsync \
        locales-all \
        libpcre3 \
        openssh-client \
        php-imagick \
        php-uploadprogress \
        sqlite3 && \
    for v in $PHP_VERSIONS; do apt-get -qq install --no-install-recommends --no-install-suggests -y  $v-apcu $v-bcmath $v-bz2 $v-curl $v-cgi $v-cli $v-common $v-fpm $v-gd $v-intl $v-json $v-memcached $v-mysql $v-pgsql $v-mbstring $v-opcache $v-soap $v-redis $v-sqlite3 $v-readline $v-xdebug $v-xml $v-xmlrpc $v-zip libapache2-mod-$v ; done && \
    for v in php5.6 php7.0 php7.1; do apt-get -qq install --no-install-recommends --no-install-suggests -y $v-mcrypt; done && \
    apt-get install blackfire-php -y --allow-unauthenticated && \
    apt-get -qq autoremove -y && \
    apt-get -qq clean -y && \
    rm -rf /var/lib/apt/lists/*

# Arbitrary user needs to be able to bind to privileged ports (for nginx and apache2)
RUN setcap CAP_NET_BIND_SERVICE=+eip /usr/sbin/nginx
RUN setcap CAP_NET_BIND_SERVICE=+eip /usr/sbin/apache2

ADD files /

RUN curl -sS https://getcomposer.org/installer | php -- --install-dir=/usr/local/bin --filename=composer
RUN composer global require hirak/prestissimo
RUN curl -sSL "https://github.com/drush-ops/drush/releases/download/${DRUSH_VERSION}/drush.phar" -o /usr/local/bin/drush8 && chmod +x /usr/local/bin/drush8
RUN curl -sSL "https://github.com/drush-ops/drush-launcher/releases/download/${DRUSH_LAUNCHER_VERSION}/drush.phar" -o /usr/local/bin/drush && chmod +x /usr/local/bin/drush
RUN curl -sSL "https://github.com/mailhog/MailHog/releases/download/v${MAILHOG_VERSION}/MailHog_linux_amd64" -o /usr/local/bin/mailhog
RUN curl -sSL -o /usr/local/bin/wp-cli -O https://raw.githubusercontent.com/wp-cli/builds/gh-pages/phar/wp-cli.phar && chmod +x /usr/local/bin/wp-cli && ln -s /usr/local/bin/wp-cli /usr/local/bin/wp

RUN curl -sSL "https://drupalconsole.com/installer" -L -o /usr/local/bin/drupal && chmod +x /usr/local/bin/drupal


RUN curl -sSL https://github.com/backdrop-contrib/drush/releases/download/${BACKDROP_DRUSH_VERSION}/drush.zip -o /tmp/backdrop_drush.zip && unzip /tmp/backdrop_drush.zip -d /var/tmp/backdrop_drush_commands

RUN mkdir -p /etc/nginx/sites-enabled /var/log/apache2 /var/run/apache2 /var/lib/apache2/module/enabled_by_admin /var/lib/apache2/module/disabled_by_admin && \
    touch /var/log/php-fpm.log && \
    chmod ugo+rw /var/log/php-fpm.log && \
    chmod ugo+rwx /var/run && \
    touch /var/log/nginx/access.log && \
    touch /var/log/nginx/error.log && \
    chmod -R ugo+rw /var/log/nginx/ && \
    chmod ugo+rx /usr/local/bin/* && \
    update-alternatives --set php /usr/bin/php${PHP_DEFAULT_VERSION} && \
    ln -s /usr/sbin/php-fpm${PHP_DEFAULT_VERSION} /usr/sbin/php-fpm

RUN chmod -R 777 /var/log

# /home is a prototype for the actual user dir, but leave it writable
RUN mkdir -p /home/.composer /home/.drush/commands /home/.drush/aliases /mnt/ddev-global-cache/mkcert && chmod -R ugo+rw /home /mnt/ddev-global-cache/

RUN chmod -R ugo+w /usr/sbin /usr/bin /etc/nginx /var/cache/nginx /run /var/www /etc/php/*/*/conf.d/ /var/lib/php/modules /etc/alternatives /usr/lib/node_modules /etc/php /etc/apache2 /var/log/apache2/ /var/run/apache2 /var/lib/apache2 /mnt/ddev-global-cache/*

RUN curl -sSL https://github.com/FiloSottile/mkcert/releases/download/$MKCERT_VERSION/mkcert-$MKCERT_VERSION-linux-amd64 -o /usr/local/bin/mkcert && chmod +x /usr/local/bin/mkcert

# Except that .my.cnf can't be writeable or mysql won't use it.
RUN chmod 644 /home/.my.cnf

RUN touch /var/log/nginx/error.log /var/log/nginx/access.log /var/log/php-fpm.log && \
  chmod 666 /var/log/nginx/error.log /var/log/nginx/access.log /var/log/php-fpm.log

RUN for v in $PHP_VERSIONS; do a2dismod $v; done
RUN a2dismod mpm_event
RUN a2enmod ssl headers expires

# ssh is very particular about permissions in ~/.ssh
RUN chmod -R go-w /home/.ssh

# scripts added last because they're most likely place to make changes, speeds up build
ADD scripts /
RUN chmod ugo+x /start.sh /healthcheck.sh

RUN addgroup --gid 98 testgroup && adduser testuser --ingroup testgroup --disabled-password --gecos "" --uid 98

EXPOSE 80 8025
HEALTHCHECK --interval=1s --retries=10 --timeout=120s --start-period=10s CMD ["/healthcheck.sh"]

CMD ["/start.sh"]
