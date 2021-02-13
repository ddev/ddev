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
RUN mkdir -p /var/lib/mysql && chmod 777 /var/lib/mysql

ADD files /

# Build a starter base db
RUN /create_base_db.sh

RUN chmod ugo+x /healthcheck.sh /backuptool.sh

# But make sure these are right
RUN chmod ugo+wx /mnt /var/tmp

RUN mkdir -p /var/log /var/tmp/mysqlbase /etc/mysql/conf.d && chmod -R ugo+wx /var/log /var/tmp/mysqlbase /etc/mysql/conf.d
RUN ln -s -f /dev/stderr /var/log/mysqld.err

RUN /sbin/mkhomedir_helper www-data

ENTRYPOINT ["/docker-entrypoint.sh"]

EXPOSE 3306
# The following line overrides any cmd entry
CMD []
HEALTHCHECK --interval=1s --retries=30 --timeout=120s CMD ["/healthcheck.sh"]
