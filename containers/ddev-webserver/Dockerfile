### ---------------------------ddev-webserver-base--------------------------------
### Build ddev-php-base from ddev-webserver-base
### ddev-php-base is the basic of ddev-php-prod
### and ddev-webserver-* (For DDEV local Usage)
ARG DOCKER_ORG=ddev
FROM ${DOCKER_ORG}/ddev-php-base:v1.25.2 AS ddev-webserver-base
SHELL ["/bin/bash", "-eu","-o", "pipefail", "-c"]
ENV DEBIAN_FRONTEND=noninteractive

ENV NGINX_SITE_TEMPLATE=/etc/nginx/nginx-site.conf
ENV APACHE_SITE_TEMPLATE=/etc/apache2/apache-site.conf
ENV TERMINUS_CACHE_DIR=/mnt/ddev-global-cache/terminus/cache
ENV CAROOT=/mnt/ddev-global-cache/mkcert

# TARGETPLATFORM is Docker buildx's target platform (e.g. linux/arm64)
# TARGETARCH is the target architecture (e.g. arm64, amd64)
ARG TARGETPLATFORM
ARG TARGETARCH

COPY ddev-webserver-etc-skel /
RUN /sbin/mkhomedir_helper www-data

# symfony cli
RUN curl -1sLf 'https://dl.cloudsmith.io/public/symfony/stable/setup.deb.sh' | bash && \
    apt modernize-sources --assume-yes && \
    rm -f /etc/apt/sources.list.d/*.list.bak

# Set up MariaDB apt repository
# The key is from https://mariadb.com/docs/server/server-management/install-and-upgrade-mariadb/installing-mariadb/binary-packages/gpg#mariadb-community-server-debian-ubuntu-key
# Search for CHANGE_MARIADB_CLIENT to update related code.
RUN KEYRING_URL="https://supplychain.mariadb.com/mariadb-keyring-2025.gpg" && \
    SCRIPT=$(curl -fsSL https://r.mariadb.com/downloads/mariadb_repo_setup) && \
    echo "${SCRIPT}" | grep -q "${KEYRING_URL}" || (echo "ERROR: KEYRING_URL does not match mariadb_repo_setup script" >&2 && exit 1) && \
    KEYRING_FILE="${KEYRING_URL##*/}" && \
    KEYRING_PATH="/usr/share/keyrings/${KEYRING_FILE}" && \
    curl -fsSL -O "${KEYRING_URL}" && \
    curl -fsSL -O "${KEYRING_URL}.sha256" && \
    sha256sum -c --quiet "${KEYRING_FILE}.sha256" && \
    gpg --dearmor < "${KEYRING_FILE}" > "${KEYRING_PATH}" && \
    rm -f "${KEYRING_FILE}" "${KEYRING_FILE}.sha256" && \
    printf "Package: *\nPin: origin archive.mariadb.org\nPin-Priority: 1000\n" > /etc/apt/preferences.d/mariadb-archive.pref && \
    printf "Types: deb\nURIs: https://archive.mariadb.org/mariadb-11.8/repo/debian\nSuites: trixie\nComponents: main\nSigned-By: ${KEYRING_PATH}\n" > /etc/apt/sources.list.d/mariadb-archive.sources

# Set up Postgresql apt repository
RUN <<EOF
    set -eu -o pipefail
    install -d /usr/share/postgresql-common/pgdg
    curl -s -o /usr/share/postgresql-common/pgdg/apt.postgresql.org.asc --fail https://www.postgresql.org/media/keys/ACCC4CF8.asc
    apt-get -qq update
    apt-get -qq install -y -o Dpkg::Options::="--force-confold" --no-install-recommends --no-install-suggests postgresql-common
    /usr/share/postgresql-common/pgdg/apt.postgresql.org.sh -y
EOF

RUN apt-get -qq update && \
    DEBIAN_FRONTEND=noninteractive apt-get -qq install -y -o Dpkg::Options::="--force-confold" --no-install-recommends --no-install-suggests \
    bash-completion \
    cron \
    gettext \
    git \
    gpgv \
    graphviz \
    jq \
    less \
    libcap2-bin \
    # AVIF write support for ImageMagick \
    libheif-plugin-aomdec \
    libheif-plugin-aomenc \
    # HEIC write support for ImageMagick \
    libheif-plugin-libde265 \
    libheif-plugin-x265 \
    libldap-common \
    # SVG support for ImageMagick \
    libmagickcore-7.q16-10-extra \
    locales \
    mariadb-client \
    openssh-client \
    patch \
    postgresql-client \
    pv \
    python-is-python3 \
    # For QR codes in `ddev share` \
    qrencode \
    rsync \
    sqlite3 \
    supervisor \
    symfony-cli \
    unzip \
    vim-tiny \
    xz-utils \
    zip

RUN update-alternatives --install /usr/bin/vim vim /usr/bin/vim.tiny 10

# Configure APT to use gpgv for signature verification to avoid SHA1 deprecation issues
RUN echo 'APT::Key::gpgvcommand "/usr/bin/gpgv";' > /etc/apt/apt.conf.d/99-use-gpgv

# Arbitrary user needs to be able to bind to privileged ports (for nginx and apache2)
RUN setcap CAP_NET_BIND_SERVICE=+eip /usr/sbin/nginx && \
    setcap CAP_NET_BIND_SERVICE=+eip /usr/sbin/apache2

RUN apt-get -qq autoremove && apt-get -qq clean -y && rm -rf /var/lib/apt/lists/* /tmp/*

COPY ddev-webserver-base-files /
COPY ddev-webserver-base-scripts /
RUN locale-gen

# /usr/local/bin may need to be updated by start.sh, etc
RUN chmod -f ugo+rwx /usr/local/bin /usr/local/bin/composer

# Install older mariadb client and mariadb-dump into /usr/local/mariadb-old/bin
# to solve variety of problems with newer mariadb-dump.
RUN <<EOF
    set -eu -o pipefail
    cd /tmp
    if [ "$(dpkg --print-architecture)" = "arm64" ]; then
        curl -L -s --fail \
            -O "https://dlm.mariadb.com/3607865/MariaDB/mariadb-10.11.6/repo/debian/pool/main/m/mariadb/mariadb-client_10.11.6+maria~deb12_arm64.deb" \
            -O "https://dlm.mariadb.com/3607494/MariaDB/mariadb-10.11.6/repo/debian/pool/main/m/mariadb/mariadb-client-core_10.11.6+maria~deb12_arm64.deb"
    elif [ "$(dpkg --print-architecture)" = "amd64" ]; then
        curl -L -s --fail \
            -O "https://dlm.mariadb.com/3607675/MariaDB/mariadb-10.11.6/repo/debian/pool/main/m/mariadb/mariadb-client_10.11.6+maria~deb12_amd64.deb" \
            -O "https://dlm.mariadb.com/3607123/MariaDB/mariadb-10.11.6/repo/debian/pool/main/m/mariadb/mariadb-client-core_10.11.6+maria~deb12_amd64.deb"
    fi
    for item in *.deb; do dpkg-deb -x "${item}" extracted; done
    mkdir -p /usr/local/mariadb-old/bin
    cp -ap /tmp/extracted/usr/bin/{mariadb,mariadb-dump,mysql,mysqldump} /usr/local/mariadb-old/bin
EOF

# Install mariadb wrappers
# Search for CHANGE_MARIADB_CLIENT to update related code.
# Install MariaDB SSL skip wrappers to disable one-way TLS verification (enabled by default in MariaDB 11.4+)
RUN DDEV_DATABASE="mariadb:11.8" mariadb-compat-install.sh && \
    DDEV_DATABASE="mariadb:11.8" mariadb-skip-ssl-wrapper-install.sh

# Register php-fpm as an alternative so versions can be switched at runtime
RUN for v in /usr/sbin/php-fpm[0-9]*; do \
      ver="${v#/usr/sbin/php-fpm}"; \
      prio="${ver//./}"; \
      update-alternatives --install /usr/sbin/php-fpm php-fpm "$v" "$prio"; \
    done

RUN <<EOF
    set -eu -o pipefail
    gh_tag() { curl -L --fail --silent "https://api.github.com/repos/$1/releases/latest" | jq -r .tag_name; }

    # Install mkcert
    tag=$(gh_tag FiloSottile/mkcert)
    curl --fail -sSL "https://github.com/FiloSottile/mkcert/releases/download/${tag}/mkcert-${tag}-linux-${TARGETARCH}" --output /usr/local/bin/mkcert
    chmod +x /usr/local/bin/mkcert

    # Install magerun and magerun2 for Magento
    curl --fail -sSL https://files.magerun.net/n98-magerun-latest.phar -o /usr/local/bin/magerun
    chmod 755 /usr/local/bin/magerun
    curl --fail -sSL https://raw.githubusercontent.com/netz98/n98-magerun/master/res/autocompletion/bash/n98-magerun.phar.bash -o /etc/bash_completion.d/n98-magerun.phar
    curl --fail -sSL https://files.magerun.net/n98-magerun2-latest.phar -o /usr/local/bin/magerun2
    chmod 755 /usr/local/bin/magerun2
    curl --fail -sSL https://raw.githubusercontent.com/netz98/n98-magerun2/master/res/autocompletion/bash/n98-magerun2.phar.bash -o /etc/bash_completion.d/n98-magerun2.phar

    # Install backdrop drush extension
    tag=$(gh_tag backdrop-contrib/drush)
    curl --fail -sSL "https://github.com/backdrop-contrib/drush/releases/download/${tag}/backdrop-drush-extension.zip" -o /tmp/backdrop-drush-extension.zip
    unzip -o /tmp/backdrop-drush-extension.zip -d /var/tmp/backdrop_drush_commands
    chmod -R ugo+w /var/tmp/backdrop_drush_commands
    rm -f /tmp/backdrop-drush-extension.zip
EOF

# Set up Blackfire apt repository
RUN curl -s --fail https://packages.blackfire.io/gpg.key > /usr/share/keyrings/blackfire-archive-keyring.asc && \
    printf "Types: deb\nURIs: http://packages.blackfire.io/debian/\nSuites: any\nComponents: main\nSigned-By: /usr/share/keyrings/blackfire-archive-keyring.asc\n" > /etc/apt/sources.list.d/blackfire.sources

### -------------------------END ddev-webserver-base------------------------------

### -------------------------ddev-webserver-dev-base------------------------------
### Build ddev-webserver-dev-base from ddev-webserver-base
FROM ddev-webserver-base AS ddev-webserver-dev-base
SHELL ["/bin/bash", "-eu", "-o", "pipefail", "-c"]

ENV PHP_DEFAULT_VERSION="8.4"

RUN apt-get -qq update && \
    DEBIAN_FRONTEND=noninteractive apt-get -qq install -o Dpkg::Options::="--force-confold" --no-install-recommends --no-install-suggests -y \
        blackfire \
        blackfire-php \
        fontconfig \
        iproute2 \
        iputils-ping \
        libpcre3 \
        nano \
        ncurses-bin \
        netcat-traditional \
        sudo \
        telnet \
        tree

# ping needs more privileges on lima-rootless
RUN setcap cap_net_raw+p /usr/bin/ping

# blackfire user by default is set up with /dev/null as homedir, and 999 as uid, which
# can break people. Use a real homedir
RUN mkdir -p /home/blackfire && chown blackfire:blackfire /home/blackfire && usermod -d /home/blackfire blackfire

COPY ddev-webserver-dev-base-files /
RUN phpdismod blackfire xdebug xhprof

RUN <<EOF
    set -eu -o pipefail
    gh_tag() { curl -L --fail --silent "https://api.github.com/repos/$1/releases/latest" | jq -r .tag_name; }

    # Install mailpit
    tag=$(gh_tag axllent/mailpit)
    curl --fail -sSL "https://github.com/axllent/mailpit/releases/download/${tag}/mailpit-linux-${TARGETARCH}.tar.gz" -o /tmp/mailpit.tar.gz
    tar -zx -C /usr/local/bin -f /tmp/mailpit.tar.gz mailpit
    rm /tmp/mailpit.tar.gz

    # Install phive
    tag=$(gh_tag phar-io/phive)
    curl --fail -sSL "https://github.com/phar-io/phive/releases/download/${tag}/phive-${tag}.phar" --output /usr/local/bin/phive
    chmod 755 /usr/local/bin/phive

    # Install terminus cli
    tag=$(gh_tag pantheon-systems/terminus)
    curl --fail -sSL "https://github.com/pantheon-systems/terminus/releases/download/${tag}/terminus.phar" --output /usr/local/bin/terminus
    chmod 755 /usr/local/bin/terminus

    # Install platform cli
    curl -fsSL https://raw.githubusercontent.com/platformsh/cli/main/installer.sh | bash
    # Modernize only after installing upsun cli to avoid duplicate sources
    #apt modernize-sources --assume-yes

    # Install upsun cli
    curl -fsSL https://raw.githubusercontent.com/platformsh/cli/main/installer.sh | VENDOR=upsun bash
    apt modernize-sources --assume-yes
    rm -f /etc/apt/sources.list.d/*.list.bak

    # Install lagoon cli
    tag=$(gh_tag uselagoon/lagoon-cli)
    curl --fail -sSL "https://github.com/uselagoon/lagoon-cli/releases/download/$tag/lagoon-cli-$tag-linux-${TARGETARCH}" --output /usr/local/bin/lagoon
    chmod 755 /usr/local/bin/lagoon

    # Install lagoon-sync
    tag=$(gh_tag uselagoon/lagoon-sync)
    curl --fail -sSL "https://github.com/uselagoon/lagoon-sync/releases/download/${tag}/lagoon-sync_${tag:1}_linux_${TARGETARCH}" --output /usr/local/bin/lagoon-sync
    chmod 755 /usr/local/bin/lagoon-sync

    # Install acli (Acquia CLI)
    curl --fail -sSL --output /usr/local/bin/acli https://github.com/acquia/cli/releases/latest/download/acli.phar
    chmod 755 /usr/local/bin/acli
EOF

RUN mkdir -p /etc/nginx/sites-enabled /var/log/apache2 /var/run/apache2 /var/lib/apache2/module/enabled_by_admin /var/lib/apache2/module/disabled_by_admin /opt/phpstorm-coverage && \
    chmod a+rw /opt/phpstorm-coverage && \
    touch /var/log/php-fpm.log && \
    chmod ugo+rw /var/log/php-fpm.log && \
    chmod ugo+rwx /var/run && \
    touch /var/log/nginx/access.log && \
    touch /var/log/nginx/error.log && \
    chmod -R ugo+rw /var/log/nginx/ && \
    chmod ugo+rwx /usr/local/bin/* && \
    update-alternatives --set php-fpm /usr/sbin/php-fpm${PHP_DEFAULT_VERSION}

RUN chmod -R 777 /var/log

# we need to create the /var/cache/nginx and /var/lib/nginx manually for the arm64 image and chmod them, please don't remove them!
RUN mkdir -p /mnt/ddev-global-cache/mkcert /run/{php,blackfire} /var/cache/nginx /var/lib/nginx && chmod -R ugo+rw /mnt/ddev-global-cache/

RUN chmod -fR ugo+w /etc/nginx /var/cache/nginx /var/lib/nginx /run /var/www /etc/php/*/*/conf.d/ /var/lib/php/modules /etc/alternatives /var/lib/dpkg/alternatives /usr/local/lib/node_modules /etc/php /etc/apache2 /var/log/apache2/ /var/run/apache2 /var/lib/apache2 /mnt/ddev-global-cache/*

RUN mkdir -p /var/xhprof && curl --fail -o /tmp/xhprof.tgz -sSL https://pecl.php.net/get/xhprof && tar -zxf /tmp/xhprof.tgz --strip-components=1 -C /var/xhprof && chmod 777 /var/xhprof/xhprof_html && rm /tmp/xhprof.tgz

# Install https://github.com/perftools/php-profiler for XHGui
RUN COMPOSER_HOME=/tmp/composer composer global require perftools/php-profiler --dev && \
    mv /tmp/composer/vendor/perftools/php-profiler /usr/local/xhgui.collector/php-profiler && \
    rm -rf /tmp/composer && \
    chmod -R ugo+rw /usr/local/xhgui.collector

RUN touch /var/log/nginx/error.log /var/log/nginx/access.log /var/log/php-fpm.log && \
    chmod 666 /var/log/nginx/error.log /var/log/nginx/access.log /var/log/php-fpm.log

RUN a2dismod mpm_event && a2enmod ssl headers expires

# scripts added last because they're most likely place to make changes, speeds up build
COPY ddev-webserver-base-scripts /
RUN chmod ugo+x /start.sh /healthcheck.sh

# Composer, etc may need to be updated by composer self-update
RUN chmod -f ugo+rwx /usr/local/bin /usr/local/bin/*

RUN chmod ugo+w /etc/ssl/certs /usr/local/share/ca-certificates

HEALTHCHECK --interval=1s --retries=120 --timeout=120s --start-period=120s CMD ["/healthcheck.sh"]
CMD ["/start.sh"]
RUN apt-get -qq clean -y && rm -rf /var/lib/apt/lists/* /tmp/*
RUN update-alternatives --set php /usr/bin/php${PHP_DEFAULT_VERSION}

### -------------------------END ddev-webserver-dev-base--------------------------

### -----------------------------ddev-webserver-----------------------------------
### This could be known as ddev-webserver-dev as it's development-env targeted
### But for historical reasons, it's just ddev-webserver
### Build ddev-webserver by turning ddev-webserver-dev-base into one layer
FROM scratch AS ddev-webserver
ENV PHP_DEFAULT_VERSION="8.4"
ENV NGINX_SITE_TEMPLATE=/etc/nginx/nginx-site.conf
ENV APACHE_SITE_TEMPLATE=/etc/apache2/apache-site.conf
ENV TERMINUS_CACHE_DIR=/mnt/ddev-global-cache/terminus/cache
ENV TERMINUS_HIDE_UPDATE_MESSAGE=1
ENV CAROOT=/mnt/ddev-global-cache/mkcert
ENV COMPOSER_ALLOW_SUPERUSER=1
ENV COMPOSER_CACHE_DIR=/mnt/ddev-global-cache/composer
ENV COMPOSER_PROCESS_TIMEOUT=2000
ENV DEBIAN_FRONTEND=noninteractive
ENV TERM=xterm
ENV MH_SMTP_BIND_ADDR=127.0.0.1:1025
ENV BASH_ENV=/etc/bash.nointeractive.bashrc
ENV LANG=C.UTF-8
ENV XHPROF_OUTPUT_DIR=/tmp/xhprof
ENV PLATFORMSH_CLI_UPDATES_CHECK=0

COPY --from=ddev-webserver-dev-base / /
HEALTHCHECK --interval=1s --retries=120 --timeout=120s --start-period=120s CMD ["/healthcheck.sh"]
CMD ["/start.sh"]
### ----------------------------END ddev-webserver--------------------------------

### -------------------------ddev-webserver-prod-base-----------------------------
### Build ddev-webserver-prod-base from ddev-webserver-base
### This image is aimed at actual hardened production environments
FROM ddev-webserver-base AS ddev-webserver-prod-base
SHELL ["/bin/bash", "-eu", "-o", "pipefail", "-c"]

ENV PHP_DEFAULT_VERSION="8.4"

# Blackfire outputs errors about not having systemd, but they do no harm to our usage
RUN apt-get -qq update && \
    DEBIAN_FRONTEND=noninteractive apt-get -qq install -o Dpkg::Options::="--force-confold" --no-install-recommends --no-install-suggests -y \
        blackfire-php 2>/dev/null

COPY ddev-webserver-prod-files /
RUN phpdismod blackfire xhprof

RUN mkdir -p /etc/nginx/sites-enabled /var/lock/apache2 /var/log/apache2 /var/run/apache2 /var/lib/apache2/module/enabled_by_admin /var/lib/apache2/module/disabled_by_admin && \
    touch /var/log/php-fpm.log && \
    chmod ugo+rw /var/log/php-fpm.log && \
    chmod ugo+rwx /var/run && \
    touch /var/log/nginx/access.log && \
    touch /var/log/nginx/error.log && \
    chmod -R ugo+rw /var/log/nginx/ && \
    chmod ugo+rx /usr/local/bin/* && \
    update-alternatives --set php-fpm /usr/sbin/php-fpm${PHP_DEFAULT_VERSION}

RUN chmod -R 777 /var/log

# we need to create the /var/cache/linux and /var/lib/nginx manually for the arm64 image and chmod them, please don't remove them!
RUN mkdir -p /mnt/ddev-global-cache/mkcert /run/php /var/cache/nginx /var/lib/nginx && chmod -R ugo+rw /home /mnt/ddev-global-cache/

RUN chmod -fR ugo+w /etc/nginx /var/cache/nginx /var/lib/nginx /run /var/www /etc/php/*/*/conf.d/ /var/lib/php/modules /etc/alternatives /var/lib/dpkg/alternatives /usr/local/lib/node_modules /etc/php /etc/apache2 /var/lock/apache2 /var/log/apache2/ /var/run/apache2 /var/lib/apache2 /mnt/ddev-global-cache/*

RUN touch /var/log/nginx/error.log /var/log/nginx/access.log /var/log/php-fpm.log && \
    chmod 666 /var/log/nginx/error.log /var/log/nginx/access.log /var/log/php-fpm.log

RUN a2dismod mpm_event && a2enmod ssl headers expires

# scripts added last because they're most likely place to make changes, speeds up build
COPY ddev-webserver-prod-scripts /
RUN chmod ugo+x /start.sh /healthcheck.sh

RUN /sbin/mkhomedir_helper www-data

RUN chmod ugo+w /etc/ssl/certs /usr/local/share/ca-certificates

HEALTHCHECK --interval=1s --retries=120 --timeout=120s --start-period=120s CMD ["/healthcheck.sh"]
CMD ["/start.sh"]
RUN apt-get -qq clean -y && rm -rf /var/lib/apt/lists/* /tmp/*
RUN update-alternatives --set php /usr/bin/php${PHP_DEFAULT_VERSION}

### -----------------------END ddev-webserver-prod-base---------------------------

### ---------------------------ddev-webserver-prod--------------------------------
### Build ddev-webserver-prod, the hardened version of ddev-webserver-base
### (Withut dev features, single layer)
FROM scratch AS ddev-webserver-prod
ENV PHP_DEFAULT_VERSION="8.4"
ENV NGINX_SITE_TEMPLATE=/etc/nginx/nginx-site.conf
ENV APACHE_SITE_TEMPLATE=/etc/apache2/apache-site.conf
ENV TERMINUS_CACHE_DIR=/mnt/ddev-global-cache/terminus/cache
ENV TERMINUS_HIDE_UPDATE_MESSAGE=1
ENV CAROOT=/mnt/ddev-global-cache/mkcert
ENV COMPOSER_ALLOW_SUPERUSER=1
ENV COMPOSER_CACHE_DIR=/mnt/ddev-global-cache/composer
ENV COMPOSER_PROCESS_TIMEOUT=2000
ENV DEBIAN_FRONTEND=noninteractive
ENV LANG=C.UTF-8
ENV TERM=xterm
ENV BASH_ENV=/etc/bash.nointeractive.bashrc
ENV PLATFORMSH_CLI_UPDATES_CHECK=0

COPY --from=ddev-webserver-prod-base / /
HEALTHCHECK --interval=1s --retries=120 --timeout=120s --start-period=120s CMD ["/healthcheck.sh"]
CMD ["/start.sh"]
### -------------------------END ddev-webserver-prod------------------------------
