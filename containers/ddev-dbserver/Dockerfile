ARG DB_TYPE
ARG DB_PINNED_VERSION
FROM ${DB_TYPE}:${DB_PINNED_VERSION}

# This must be reiterated because everything is emptied on FROM
ARG DB_TYPE
ARG DB_VERSION

ENV MYSQL_DATABASE db
ENV MYSQL_USER db
ENV MYSQL_PASSWORD db
ENV MYSQL_ROOT_PASSWORD root

SHELL ["/bin/bash", "-c"]

# Remove obsolete MySQL 5.5/5.6 Jessie and before keys so they don't make expiration key test stumble
RUN for item in "75DD C3C4 A499 F1A1 8CB5  F3C8 CBF8 D6FD 518E 17E1" "126C 0D24 BD8A 2942 CC7D  F8AC 7638 D044 2B90 D010" "D211 6914 1CEC D440 F2EB  8DDA 9D6D 8F6B C857 C906" "A1BD 8E9D 78F7 FE5C 3E65  D8AF 8B48 AD62 4692 5553" "ED6D 6527 1AAC F0FF 15D1  2303 6FB2 A1C2 65FF B764"; do \
    apt-key remove "${item}" || true; \
  done;


RUN apt-get -qq update && apt-get -qq install -y tzdata gnupg2 pv less vim wget curl lsb-release >/dev/null

RUN set -x; if ( ! command -v xtrabackup && ! command -v mariabackup ); then \
    curl -sSL https://repo.percona.com/apt/percona-release_latest.$(lsb_release -sc)_all.deb -o percona-release_latest.$(lsb_release -sc)_all.deb; \
    dpkg -i percona-release_latest.$(lsb_release -sc)_all.deb >/dev/null; \
    rm percona-release*.deb ; \
    xtrabackup_version=percona-xtrabackup-24 ; \
    if [ "$DB_VERSION" = "8.0" ]; then xtrabackup_version=percona-xtrabackup-80; fi ; \
    apt-get -qq update && apt-get -qq install -y ${xtrabackup_version} >/dev/null ; \
fi
RUN apt-get -qq autoclean

RUN rm -rf /var/lib/mysql/* /etc/mysql
RUN mkdir -p /var/lib/mysql

ADD files /

# Build a starter base db
RUN /create_base_db.sh

RUN chmod ugo+x /healthcheck.sh /backuptool.sh

# But make sure these are right
RUN chmod ugo+wx /mnt /var/tmp

RUN mkdir -p /var/log /var/tmp/mysqlbase /etc/mysql/conf.d && chmod -R ugo+wx /var/log /var/tmp/mysqlbase /etc/mysql/conf.d
RUN ln -s -f /dev/stderr /var/log/mysqld.err

RUN rm -rf /var/lib/mysql/*
RUN chmod -R ugo+rw /var/lib/mysql /etc/mysql/conf.d /mysqlbase && find /mysqlbase -type d | xargs chmod ugo+rwx

RUN /sbin/mkhomedir_helper www-data

ENTRYPOINT ["/docker-entrypoint.sh"]

EXPOSE 3306
# The following line overrides any cmd entry
CMD []
HEALTHCHECK --interval=1s --retries=30 --timeout=120s CMD ["/healthcheck.sh"]
