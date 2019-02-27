FROM mariadb:10.2.22

ENV MYSQL_DATABASE db
ENV MYSQL_USER db
ENV MYSQL_PASSWORD db
ENV MYSQL_ROOT_PASSWORD root

# Install mariadb and other packages
RUN apt-get update && apt-get install -y tzdata sudo pv

RUN rm -rf /var/lib/mysql/* /etc/mysql
RUN mkdir -p /var/lib/mysql && chmod 777 /var/lib/mysql

ADD files /

RUN chmod ugo+x /healthcheck.sh

# Security-sensitive changes: Make sure our start script can do what is needed
# But make sure these are right
RUN chmod ugo+wx /mnt /var/tmp
RUN chmod -R ugo+wx /var/log /var/tmp/mysqlbase /etc/mysql/conf.d
RUN ln -s /dev/stderr /var/log/mysqld.err

ENTRYPOINT ["/docker-entrypoint.sh"]

EXPOSE 3306
# The following line overrides any cmd entry
CMD []
HEALTHCHECK --interval=2s --retries=30 CMD ["/healthcheck.sh"]
