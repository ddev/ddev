# syntax=docker/dockerfile:1
# check=skip=SecretsUsedInArgOrEnv

ARG BASE_IMAGE="scratch"
ARG DB_PINNED_VERSION="latest"
FROM ${BASE_IMAGE}:${DB_PINNED_VERSION}
SHELL ["/bin/bash", "-eu","-o", "pipefail",  "-c"]

# This must be reiterated because everything is emptied on FROM
ARG BASE_IMAGE
ARG DB_TYPE
ARG DB_MAJOR_VERSION

ARG PERCONA_SETUP_URL=https://repo.percona.com/apt/percona-release_latest.generic_all.deb

ARG LANG=C.UTF-8
ARG MYSQL_DATABASE=db
ARG MYSQL_USER=db
ARG MYSQL_PASSWORD=db
ARG MYSQL_ROOT_PASSWORD=root

SHELL ["/bin/bash", "-c"]

# Fix APT for Debian Stretch (EOL; upstream mirrors disabled)
# Based on: https://serverfault.com/a/1131653
RUN <<EOF
    set -eu -o pipefail
    if grep "Debian GNU/Linux 9" /etc/issue >/dev/null; then
        rm -f /etc/apt/sources.list.d/mysql.list
        echo "deb http://archive.debian.org/debian/ stretch main contrib non-free" >/etc/apt/sources.list
        echo "deb http://archive.debian.org/debian-security/ stretch/updates main contrib non-free" >>/etc/apt/sources.list
        apt-get -qq update -o Acquire::AllowInsecureRepositories=true -o Acquire::AllowDowngradeToInsecureRepositories=true -o APT::Get::AllowUnauthenticated=true
        DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends --no-install-suggests -o APT::Get::AllowUnauthenticated=true debian-archive-keyring
    fi
EOF

# MariaDB 11.x moved MySQL symlinks into separate packages
RUN set -x; if ( command -v mariadbd && ! command -v mysqld ); then \
    apt-get -qq update && DEBIAN_FRONTEND=noninteractive apt-get install -y -o Dpkg::Options::="--force-confold" mariadb-server-compat mariadb-client-compat; \
    fi

# Older versions of mariadb have been removed from
# the mariadb apt repository, so we don't want to
# look there when doing apt-get update. And we don't use new packages from there.
# And we're going to install our own percona.list if needed, so get that if needed
# and remove it here
USER root
RUN rm -f /etc/apt/sources.list.d/mariadb.list /etc/apt/sources.list.d/percona.list
RUN mkdir -p /var/lib/apt/lists/partial
RUN apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y -o Dpkg::Options::="--force-confold" bzip2 curl gnupg2 gpgv less lsb-release pv tzdata vim-tiny wget
RUN update-alternatives --install /usr/bin/vim vim /usr/bin/vim.tiny 10

# Configure APT to use gpgv for signature verification to avoid SHA1 deprecation issues
RUN echo 'APT::Key::gpgvcommand "/usr/bin/gpgv";' > /etc/apt/apt.conf.d/99-use-gpgv

RUN <<EOF
    set -eu -o pipefail
    arch=$(dpkg --print-architecture)
    set -x;
    if ( ! command -v xtrabackup && ! command -v mariabackup ); then
        curl -o /tmp/percona_release_latest.deb --fail -sSL ${PERCONA_SETUP_URL}
        apt-get install -y /tmp/percona_release_latest.deb
        rm /tmp/percona_release_latest.deb

        # Use http when connecting. This only affects MariaDB 5.5
        if [ "$(lsb_release -i -s)" = "Ubuntu" ] && "$(lsb_release -r -s)" <= "16.04" ]; then sed -i s/https:/http:/g /etc/apt/sources.list.d/percona.list; fi

        case ${DB_MAJOR_VERSION} in
            "8.0")
                percona-release setup pxb80
                apt-get update && apt-get -qq install -y percona-xtrabackup-80
                ;;

            "8.4")
                percona-release setup pxb84lts
                apt-get update && apt-get -qq install -y percona-xtrabackup-84
                ;;
            # Older versions of mysql and a few mariadb use pxb24
            *)
                percona-release setup pxb24
                apt-get update && apt-get install -qq -y percona-xtrabackup-24
                ;;
        esac
    fi
EOF

# Install zstd for database snapshots.
RUN <<EOF
    set -eu -o pipefail
    if ! grep "Ubuntu 14.04" /etc/issue >/dev/null; then
        apt-get -qq update && DEBIAN_FRONTEND=noninteractive apt-get install -y -o Dpkg::Options::="--force-confold" zstd
    fi
EOF

RUN apt-get -qq autoclean

RUN rm -rf /var/lib/mysql /etc/mysql
RUN mkdir -p /var/lib/mysql

ADD files /

# On bitnami-derived images, remove their default config so we get ours
RUN rm -rf /opt/bitnami/mysql/conf

ARG MYSQL_UNIX_PORT=/var/tmp/mysql.sock
RUN mkdir -p /var/log /var/tmp/mysqlbase /etc/mysql/conf.d && chmod -R ugo+wx /var/log /var/tmp/mysqlbase /etc/mysql/conf.d
RUN  ln -sf /tmp/mysqlx.sock ${MYSQL_UNIX_PORT}
RUN if ! id mysql &>/dev/null ; then useradd -u 112 mysql; fi

# Build a starter base db
RUN /create_base_db.sh

RUN chmod ugo+x /healthcheck.sh

# But make sure these are right
RUN chmod ugo+wx /mnt /var/tmp

RUN rm -rf /var/lib/mysql/*
RUN chmod -R ugo+rw /var/lib/mysql /etc/mysql/conf.d /mysqlbase && find /mysqlbase -type d | xargs chmod ugo+rwx
# version-conf.d should be writable so we can symlink, but the actual config files should not be
RUN chmod ugo+w /etc/mysql/version-conf.d && chmod -R ugo-w /etc/mysql/version-conf.d/*

RUN mkdir -p /var/run/mysqld && chmod 755 /var/run/mysqld

RUN /sbin/mkhomedir_helper www-data

# Normal upstream image doesn't actually have /home/mysql created
# Make sure it's there in case user 999 (mysql) is using this image.
RUN mkdir /home/mysql && chown mysql:mysql /home/mysql

ENTRYPOINT ["/docker-entrypoint.sh"]

EXPOSE 3306
# The following line overrides any cmd entry
CMD []
HEALTHCHECK --interval=1s --retries=30 --timeout=120s CMD ["/healthcheck.sh"]
