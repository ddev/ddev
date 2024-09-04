ARG BASE_IMAGE="scratch"
ARG DB_PINNED_VERSION="latest"
FROM ${BASE_IMAGE}:${DB_PINNED_VERSION}

# This must be reiterated because everything is emptied on FROM
ARG BASE_IMAGE
ARG DB_VERSION

ENV LANG=C.UTF-8
ENV MYSQL_DATABASE=db
ENV MYSQL_USER=db
ENV MYSQL_PASSWORD=db
ENV MYSQL_ROOT_PASSWORD=root

SHELL ["/bin/bash", "-c"]

# Debian Stretch archives have been turned off
RUN if grep "Debian GNU/Linux 9" /etc/issue >/dev/null ; then \
    echo "deb http://archive.debian.org/debian/ stretch main contrib non-free" >/etc/apt/sources.list; \
    fi
# Remove obsolete MySQL 5.5/5.6 Jessie and before keys so they don't make expiration key test stumble
RUN for item in "75DD C3C4 A499 F1A1 8CB5  F3C8 CBF8 D6FD 518E 17E1" "126C 0D24 BD8A 2942 CC7D  F8AC 7638 D044 2B90 D010" "D211 6914 1CEC D440 F2EB  8DDA 9D6D 8F6B C857 C906" "A1BD 8E9D 78F7 FE5C 3E65  D8AF 8B48 AD62 4692 5553" "ED6D 6527 1AAC F0FF 15D1  2303 6FB2 A1C2 65FF B764"; do \
    apt-key remove "${item}" || true; \
  done;

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
RUN apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y -o Dpkg::Options::="--force-confold" bzip2 curl gnupg2 less lsb-release pv tzdata vim wget

RUN <<EOF
    arch=$(dpkg --print-architecture)
    set -x;
    if ( ! command -v xtrabackup && ! command -v mariabackup ); then
        curl --fail -sSL https://repo.percona.com/apt/percona-release_latest.$(lsb_release -sc)_all.deb -o /tmp/percona-release_latest.deb
        apt-get install -y /tmp/percona-release_latest.deb
        rm /tmp/percona-release*.deb

        # Use http when connecting. This only affects MariaDB 5.5
        if [ "$(lsb_release -i -s)" = "Ubuntu" ] && "$(lsb_release -r -s)" <= "16.04" ]; then sed -i s/https:/http:/g /etc/apt/sources.list.d/percona.list; fi

        case ${DB_VERSION} in
            "8.0")
                xtrabackup_version=percona-xtrabackup-80; percona_setup=pxb80;
                percona-release setup pxb80
                apt-get -qq update && apt-get -qq install -y percona-xtrabackup-80
                ;;
            # TODO: This is just temp hack to get xtrabackup here, it should end up using their `percona-release setup pxb84-lts` tool
            # Currently broken due to https://perconadev.atlassian.net/issues/PXB-3361
            "8.4")
                curl -L -s --fail -o /tmp/xtrabackup.deb https://repo.percona.com/pxb-84-lts/apt/pool/main/p/percona-xtrabackup-84/percona-xtrabackup-84_8.4.0-1-1.bookworm_${arch}.deb
                apt-get update && apt-get -qq install -y /tmp/xtrabackup.deb
                rm /tmp/xtrabackup.deb
                ;;
            *)
                percona-release setup pxb24
                apt-get update && apt-get install -qq -y percona-xtrabackup-24
                ;;
        esac
    fi
EOF

RUN apt-get -qq autoclean

RUN rm -rf /var/lib/mysql /etc/mysql
RUN mkdir -p /var/lib/mysql

ADD files /
# TODO: Do this only if needed, or perhaps put these files elsewhere
RUN mkdir -p /opt/bitnami/mysql/tmp && chmod 777 /opt/bitnami/mysql/tmp

ENV MYSQL_UNIX_PORT=/var/tmp/mysql.sock
RUN mkdir -p /var/log /var/tmp/mysqlbase /etc/mysql/conf.d && chmod -R ugo+wx /var/log /var/tmp/mysqlbase /etc/mysql/conf.d
RUN ln -s -f /dev/stderr /var/log/mysqld.err
RUN  ln -sf /tmp/mysqlx.sock ${MYSQL_UNIX_PORT}
RUN useradd -u 1001 mysql
# Build a starter base db
RUN /create_base_db.sh

RUN chmod ugo+x /healthcheck.sh

# But make sure these are right
RUN chmod ugo+wx /mnt /var/tmp

RUN rm -rf /var/lib/mysql/*
RUN chmod -R ugo+rw /var/lib/mysql /etc/mysql/conf.d /mysqlbase && find /mysqlbase -type d | xargs chmod ugo+rwx

RUN mkdir -p /var/run/mysqld && chmod 777 /var/run/mysqld

RUN /sbin/mkhomedir_helper www-data

# Remove the /etc/apt entry so that if they don't renew the key
# apt-get update will continue to function
# Remove MySQL published key that expires 2022-02-15
# Have requested them to update this in https://bugs.mysql.com/bug.php?id=105632
# But they haven't done it, and it will break things when it expires.
RUN rm -f /etc/apt/sources.list.d/mysql.list && \
    for item in "A4A9 4068 76FC BD3C 4567  70C8 8C71 8D3B 5072 E1F5" ; do \
        apt-key remove "${item}" || true; \
    done;

# Normal upstream image doesn't actually have /home/mysql created
# Make sure it's there in case user 999 (mysql) is using this image.
RUN mkdir /home/mysql && chown mysql:mysql /home/mysql

ENTRYPOINT ["/docker-entrypoint.sh"]

EXPOSE 3306
# The following line overrides any cmd entry
CMD []
HEALTHCHECK --interval=1s --retries=30 --timeout=120s CMD ["/healthcheck.sh"]
