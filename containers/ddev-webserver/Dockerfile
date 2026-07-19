### ---------------------------base-----------------------------------------
### Build the base Debian image that will be used in every other stage
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

### --------------------------ddev-keep-root-build--------------------------------
### Build the ddev-keep-root.so shim in a separate stage so gcc is not added to
### the shipped images. The shim lets a process keep running as root; see
### ddev-keep-root.c for details.
FROM base AS ddev-keep-root-build
SHELL ["/bin/bash", "-eu","-o", "pipefail",  "-c"]
RUN apt-get -qq update && \
    apt-get -qq install --no-install-recommends --no-install-suggests -y gcc libc6-dev && \
    gcc -shared -fPIC -O2 -Wall -o /ddev-keep-root.so /usr/local/src/ddev/ddev-keep-root.c -ldl
### ------------------------END ddev-keep-root-build------------------------------

### ------------------------ddev-php-extension-build------------------------------
### Uncomment the lines below to enable custom PHP extension builds:
#FROM base AS ddev-php-extension-build
#SHELL ["/bin/bash", "-eu","-o", "pipefail",  "-c"]
### Use this section to add PHP extensions that need to be built manually.
### Example: The following line installs Xdebug version 3.2.2 for PHP 8.1:
#RUN /usr/local/bin/build_php_extension.sh "php8.1" "xdebug" "3.2.2" "/usr/lib/php/20210902/xdebug.so"
### See more examples in the /usr/local/bin/build_php_extension.sh script
### -----------------------END ddev-php-extension-build---------------------------

### ---------------------------ddev-php-base--------------------------------------
### Build ddev-php-base, which is the base for ddev-webserver-*
### This combines the packages and features of DDEV's ddev-webserver and PHP image
FROM base AS ddev-php-base
SHELL ["/bin/bash", "-eu","-o", "pipefail",  "-c"]
ARG PHP_DEFAULT_VERSION="8.4"
ARG XDEBUG_MODE=off
ENV DDEV_PHP_VERSION=$PHP_DEFAULT_VERSION
ARG PHP_VERSIONS="php8.2 php8.3 php8.4 php8.5"
ENV PHP_INI=/etc/php/$PHP_DEFAULT_VERSION/fpm/php.ini
ARG DRUSH_VERSION=8.5.0
ARG NODE_VERSION=24
ENV PATH=$PATH:/usr/local/n/bin
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

# Add the ddev-keep-root.so shim (see ddev-keep-root.c). It does nothing unless
# DDEV_KEEP_ROOT is set, and DDEV deletes it from the web image unless it is
# needed (apache-fpm on Docker rootless + bind mounts); see WriteBuildDockerfile.
COPY --from=ddev-keep-root-build /ddev-keep-root.so /usr/local/lib/ddev/ddev-keep-root.so

RUN curl -L --fail -o /usr/local/bin/n -sSL https://raw.githubusercontent.com/tj/n/master/bin/n && chmod +x /usr/local/bin/n
# Install system Node.js, `--cleanup` is important, it removes downloaded files.
RUN N_PREFIX=/usr/local/n n install --cleanup "${NODE_VERSION}"
RUN npm install --unsafe-perm=true --global gulp-cli yarn

# Loop through $PHP_VERSIONS, selecting packages for the target architecture.
RUN for v in ${PHP_VERSIONS}; do \
    /usr/local/bin/install_php_extensions.sh "$v" "${TARGETPLATFORM#linux/}"; \
done
RUN update-alternatives --set php /usr/bin/php${PHP_DEFAULT_VERSION}
RUN mkdir -p /etc/nginx/sites-enabled /var/lock/apache2 /var/log/apache2 /var/run/apache2 /var/lib/apache2/module/enabled_by_admin /var/lib/apache2/module/disabled_by_admin && \
    touch /var/log/php-fpm.log && \
    chmod ugo+rw /var/log/php-fpm.log && \
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
### See more examples in the /usr/local/bin/build_php_extension.sh script
### -----------------------END ddev-php-extension-build---------------------------

RUN phpdismod xhprof xdebug
RUN apt-get -qq autoremove -y
RUN curl -L --fail -o /usr/local/bin/composer -sSL https://getcomposer.org/composer-stable.phar && chmod ugo+wx /usr/local/bin/composer
RUN <<ENDDRUSH
    set -eu -o pipefail
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
RUN mkdir -p /run/php && \
    chown www-data:www-data /run/php && chmod 755 /run/php && \
    ln -sf /run/php/php-fpm.sock /run/php-fpm.sock
RUN chmod 777 /var/lib/php/sessions
### -------------------------END ddev-php-base------------------------------------

### ---------------------------ddev-webserver-base--------------------------------
### Build ddev-webserver-base from ddev-php-base
FROM ddev-php-base AS ddev-webserver-base
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
RUN sed -i 's#listen = /run/php-fpm.sock#listen = /run/php/php-fpm.sock#' /etc/php/*/fpm/pool.d/www.conf
# Debian's nginx (since 1.26.3-3+deb13u4, a workaround for Debian bug #1126960)
# passes the host to backends as the port-stripped $host nginx variable instead of
# the client's Host header, so PHP sees HTTP_HOST without the port and apps
# build wrong absolute URLs whenever router_http(s)_port is not 80/443.
# Rewrite the *_params files to use $http_host (the Host header as sent by the
# client, port included):
#   fastcgi_param HTTP_HOST $host;  -> fastcgi_param HTTP_HOST $http_host;
#     (same for scgi_param and uwsgi_param, matched by \w+_param)
#   proxy_set_header Host $host;    -> proxy_set_header Host $http_host;
# If Debian ever drops these lines, the seds are no-ops and nginx's default
# pass-through of the Host header is correct again. Debian's own comment says
# nginx 1.30.0+ will preserve the port via $host$is_request_port$request_port;
# once that lands, our "$host;" match no longer fires and we inherit the correct
# upstream behavior automatically (see ddev/ddev#8555).
# The last two seds rewrite Debian's now-contradictory "Do not use $http_host"
# comment so `cat`ing the file in the container is not misleading.
RUN sed -i 's#^\(\w\+_param\s\+HTTP_HOST\s\+\)$host;#\1$http_host;#' /etc/nginx/fastcgi_params /etc/nginx/scgi_params /etc/nginx/uwsgi_params && \
    sed -i 's#^\(proxy_set_header\s\+Host\s\+\)$host;#\1$http_host;#' /etc/nginx/proxy_params && \
    sed -i 's@^# Do not use HTTP_HOST as "$http_host"\.@# DDEV: the HTTP_HOST line below intentionally uses $http_host to preserve a nonstandard router port (ddev/ddev#8555); Debian set $host as a workaround for bug #1126960, which strips the port.@' /etc/nginx/fastcgi_params /etc/nginx/scgi_params /etc/nginx/uwsgi_params && \
    sed -i 's@^# Do not set the `Host` header as "$http_host"\.@# DDEV: the Host line below intentionally uses $http_host to preserve a nonstandard router port (ddev/ddev#8555); Debian set $host as a workaround for bug #1126960, which strips the port.@' /etc/nginx/proxy_params
# Guard: fail the build if the HTTP_HOST/Host rewrite above did not land as
# $http_host. This deliberately breaks the build in two cases, both of which a
# maintainer must look at:
#   - the sed silently failed to match because Debian reformatted its $host
#     workaround while still stripping the port -> our fix has regressed; or
#   - Debian shipped the proper fix (nginx 1.30.0+ builds HTTP_HOST from
#     $host$is_request_port$request_port) -> our workaround is now dead code and
#     should be removed.
# Without this guard the second case is invisible: the functional test stays
# green because the port is preserved upstream, and the dead seds linger. See
# ddev/ddev#8555.
RUN <<'EOF'
set -eu -o pipefail
for f in /etc/nginx/fastcgi_params /etc/nginx/scgi_params /etc/nginx/uwsgi_params; do
  if ! grep -Eq '^[a-z]+_param[[:space:]]+HTTP_HOST[[:space:]]+\$http_host;' "$f"; then
    echo "ERROR: ${f}: HTTP_HOST is not set to \$http_host after the rewrite." >&2
    echo "Debian's nginx-common line changed (bug #1126960 / ddev/ddev#8555)." >&2
    echo "If upstream now preserves the port (nginx 1.30.0 \$host\$is_request_port\$request_port), remove this workaround; otherwise fix the seds above." >&2
    exit 1
  fi
done
if ! grep -Eq '^proxy_set_header[[:space:]]+Host[[:space:]]+\$http_host;' /etc/nginx/proxy_params; then
  echo "ERROR: /etc/nginx/proxy_params: Host is not set to \$http_host after the rewrite (bug #1126960 / ddev/ddev#8555)." >&2
  exit 1
fi
EOF
RUN locale-gen

# Composer may need to be updated by composer self-update
RUN chmod -f ugo+rx /usr/local/bin && \
    chgrp www-data /usr/local/bin/composer && \
    chmod g+rwx,o-w /usr/local/bin/composer

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
    touch /var/log/supervisord.log && \
    chmod ugo+rw /var/log/supervisord.log && \
    touch /var/log/nginx/access.log && \
    touch /var/log/nginx/error.log && \
    chmod -R ugo+rwX /var/log/apache2 /var/log/nginx && \
    chmod ugo+rx /usr/local/bin && \
    chmod ugo+rx /usr/local/bin/* && \
    update-alternatives --set php-fpm /usr/sbin/php-fpm${PHP_DEFAULT_VERSION}

# we need to create the /var/cache/nginx and /var/lib/nginx manually for the arm64 image and chmod them, please don't remove them!
RUN mkdir -p /mnt/ddev-global-cache/mkcert /run/php /run/blackfire /var/run/nginx /var/run/supervisor /var/run/apache2 /var/lock/apache2 /var/cache/nginx /var/lib/nginx && \
    chown www-data:www-data /run/php /run/blackfire /var/run/nginx /var/run/supervisor /var/run/apache2 /var/lock/apache2 && \
    chmod 755 /run/php /run/blackfire /var/run/nginx /var/run/supervisor /var/run/apache2 /var/lock/apache2 && \
    ln -sf /run/php/php-fpm.sock /run/php-fpm.sock && \
    chmod -R ugo+rw /mnt/ddev-global-cache/

RUN chgrp -R www-data /etc/alternatives /var/lib/dpkg/alternatives && \
    chmod -R g+rwX,o-w /etc/alternatives /var/lib/dpkg/alternatives

RUN chmod -fR ugo+w /etc/nginx /var/cache/nginx /var/lib/nginx /var/www /etc/php/*/*/conf.d/ /var/lib/php/modules /etc/php /etc/apache2 /var/lib/apache2 /mnt/ddev-global-cache/*

RUN mkdir -p /var/xhprof && curl --fail -o /tmp/xhprof.tgz -sSL https://pecl.php.net/get/xhprof && tar -zxf /tmp/xhprof.tgz --strip-components=1 -C /var/xhprof && chmod ugo+rwX /var/xhprof/xhprof_html && rm /tmp/xhprof.tgz

# Install https://github.com/perftools/php-profiler for XHGui
RUN COMPOSER_HOME=/tmp/composer composer global require perftools/php-profiler --dev && \
    mv /tmp/composer/vendor/perftools/php-profiler /usr/local/xhgui.collector/php-profiler && \
    rm -rf /tmp/composer && \
    chmod -R ugo+rw /usr/local/xhgui.collector

RUN touch /var/log/nginx/error.log /var/log/nginx/access.log /var/log/php-fpm.log && \
    chmod ugo+rw /var/log/php-fpm.log && \
    chmod -R ugo+rwX /var/log/apache2 /var/log/nginx

RUN a2dismod mpm_event && a2enmod ssl headers expires

# scripts added last because they're most likely place to make changes, speeds up build
COPY ddev-webserver-base-scripts /
RUN chmod ugo+x /start.sh /healthcheck.sh

# Composer, etc may need to be updated by composer self-update
RUN chmod -f ugo+rx /usr/local/bin /usr/local/bin/* && \
    chgrp www-data /usr/local/bin/composer && \
    chmod g+rwx,o-w /usr/local/bin/composer

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
ENV PATH=$PATH:/usr/local/n/bin

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
    touch /var/log/supervisord.log && \
    chmod ugo+rw /var/log/supervisord.log && \
    touch /var/log/nginx/access.log && \
    touch /var/log/nginx/error.log && \
    chmod -R ugo+rwX /var/log/apache2 /var/log/nginx && \
    chmod ugo+rx /usr/local/bin/* && \
    update-alternatives --set php-fpm /usr/sbin/php-fpm${PHP_DEFAULT_VERSION}

# we need to create the /var/cache/linux and /var/lib/nginx manually for the arm64 image and chmod them, please don't remove them!
RUN mkdir -p /mnt/ddev-global-cache/mkcert /run/php /var/run/nginx /var/run/supervisor /var/run/apache2 /var/lock/apache2 /var/cache/nginx /var/lib/nginx && \
    chown www-data:www-data /run/php /var/run/nginx /var/run/supervisor /var/run/apache2 /var/lock/apache2 && \
    chmod 755 /run/php /var/run/nginx /var/run/supervisor /var/run/apache2 /var/lock/apache2 && \
    ln -sf /run/php/php-fpm.sock /run/php-fpm.sock && \
    chmod -R ugo+rw /home /mnt/ddev-global-cache/

RUN chgrp -R www-data /etc/alternatives /var/lib/dpkg/alternatives && \
    chmod -R g+rwX,o-w /etc/alternatives /var/lib/dpkg/alternatives

RUN chmod -fR ugo+w /etc/nginx /var/cache/nginx /var/lib/nginx /var/www /etc/php/*/*/conf.d/ /var/lib/php/modules /etc/php /etc/apache2 /var/lib/apache2 /mnt/ddev-global-cache/*

RUN touch /var/log/nginx/error.log /var/log/nginx/access.log /var/log/php-fpm.log && \
    chmod ugo+rw /var/log/php-fpm.log && \
    chmod -R ugo+rwX /var/log/apache2 /var/log/nginx

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
ENV PATH=$PATH:/usr/local/n/bin

COPY --from=ddev-webserver-prod-base / /
HEALTHCHECK --interval=1s --retries=120 --timeout=120s --start-period=120s CMD ["/healthcheck.sh"]
CMD ["/start.sh"]
### -------------------------END ddev-webserver-prod------------------------------
