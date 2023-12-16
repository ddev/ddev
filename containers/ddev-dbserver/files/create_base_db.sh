#!/bin/bash

set -eu
set -x
set -o pipefail

SOCKET=/var/tmp/mysql.sock
MYSQL_UNIX_PORT=$SOCKET
OUTDIR=/mysqlbase

mkdir -p ${OUTDIR}
chown -R "$(id -u):$(id -g)" $OUTDIR

chmod ugo+w /var/tmp
mkdir -p /var/lib/mysql /mnt/ddev_config/mysql && rm -f /var/lib/mysql/* && chmod -R ugo+w /var/lib/mysql

# On Github Actions, it seems that Apparmor prevents mysqld from having access to /etc/my.cnf, so
# copy to a simpler directory
cp /etc/my.cnf /var/tmp

echo 'Initializing mysql'
mysqld --version
mysqld_version=$(mysqld --version | awk '{ print $3 }')
mysqld_version=${mysqld_version%%-*}
mysqld_version=${mysqld_version%.*}
echo version=$mysqld_version
# Oracle mysql 5.7+ deprecates mysql_install_db
if [ "${mysqld_version}" = "5.7" ] || [  "${mysqld_version%%%.*}" = "8.0" ]; then
    mysqld --defaults-file=/var/tmp/my.cnf --initialize-insecure --datadir=/var/lib/mysql --server-id=0
else
    # mysql 5.5 requires running mysql_install_db in /usr/local/mysql
    if command -v mysqld | grep usr.local; then
        cd /usr/local/mysql
    fi
    mysql_install_db --defaults-file=/var/tmp/my.cnf --force --datadir=/var/lib/mysql
fi
echo "Starting mysqld --skip-networking --socket=${MYSQL_UNIX_PORT}"
mysqld --defaults-file=/var/tmp/my.cnf --user=root --socket=${MYSQL_UNIX_PORT} --innodb_log_file_size=48M --skip-networking --datadir=/var/lib/mysql --server-id=0 --skip-log-bin &
pid="$!"

# Wait for the server to respond to mysqladmin ping, or fail if it never does,
# or if the process dies.
for i in {90..0}; do
	if mysqladmin ping -uroot --socket=${MYSQL_UNIX_PORT} 2>/dev/null; then
		break
	fi
	# Test to make sure we got it started in the first place. kill -s 0 just tests to see if process exists.
	if ! kill -s 0 $pid 2>/dev/null; then
		echo "MariaDB initialization startup failed"
		exit 3
	fi
	sleep 1
done
if [ "$i" -eq 0 ]; then
	echo 'MariaDB initialization startup process timed out.'
	exit 4
fi


mysql_tzinfo_to_sql /usr/share/zoneinfo | mysql -uroot  --socket=${MYSQL_UNIX_PORT} mysql

mysql -uroot --socket=${MYSQL_UNIX_PORT} <<EOF
	CREATE DATABASE IF NOT EXISTS $MYSQL_DATABASE;
	CREATE USER '$MYSQL_USER'@'%' IDENTIFIED BY '$MYSQL_PASSWORD';
	CREATE USER '$MYSQL_USER'@'localhost' IDENTIFIED BY '$MYSQL_PASSWORD';

	GRANT ALL ON $MYSQL_DATABASE.* TO '$MYSQL_USER'@'%';
	GRANT ALL ON $MYSQL_DATABASE.* TO '$MYSQL_USER'@'localhost';

	CREATE USER 'root'@'%' IDENTIFIED BY '$MYSQL_ROOT_PASSWORD';
	GRANT ALL ON *.* TO 'root'@'%' WITH GRANT OPTION;
	FLUSH PRIVILEGES;
	FLUSH TABLES;
EOF

mysqladmin -uroot --socket=${MYSQL_UNIX_PORT} password root

if [  "${mysqld_version%%%.*}" = "8.0" ]; then
    mysql -uroot -proot --socket=${MYSQL_UNIX_PORT} <<EOF
        ALTER USER 'db'@'%' IDENTIFIED WITH mysql_native_password BY '$MYSQL_PASSWORD';
        ALTER USER 'root'@'%' IDENTIFIED WITH mysql_native_password BY '$MYSQL_ROOT_PASSWORD';
        ALTER USER 'root'@'localhost' IDENTIFIED WITH mysql_native_password BY '$MYSQL_ROOT_PASSWORD';
EOF
fi

mysql -uroot -proot --socket=${MYSQL_UNIX_PORT} -e "SELECT @@character_set_database, @@collation_database;"

rm -rf ${OUTDIR}/*

backuptool="mariabackup --defaults-file=/var/tmp/my.cnf"
streamtool=xbstream
if command -v xtrabackup; then
  backuptool="xtrabackup --defaults-file=/var/tmp/my.cnf --datadir=/var/lib/mysql";
  streamtool=xbstream
fi

# Initialize with current mariadb_version
PATH=$PATH:/usr/sbin:/usr/local/bin:/usr/local/mysql/bin mysqld -V 2>/dev/null  | awk '{print $3}' > /tmp/raw_mysql_version.txt
# mysqld -V gives us the version in the form of 5.7.28-log for mysql or
# 5.5.64-MariaDB-1~trusty for MariaDB. Detect database type and version and output
# mysql-8.0 or mariadb-10.5.
server_db_version=$(awk -F- '{ sub( /\.[0-9]+(-.*)?$/, "", $1); server_type="mysql"; if ($2 ~ /^MariaDB/) { server_type="mariadb" }; print server_type "_" $1 }' /tmp/raw_mysql_version.txt)
echo ${server_db_version} >/var/lib/mysql/db_mariadb_version.txt
${backuptool} --backup --stream=${streamtool} --user=root --password=root --socket=${MYSQL_UNIX_PORT}  | gzip >${OUTDIR}/base_db.gz
rm -f /tmp/raw_mysql_version.txt

if ! kill -s TERM "$pid" || ! wait "$pid"; then
	echo >&2 'Database initialization process failed.'
	exit 5
fi

echo "The startup database files (in mariabackup/xtradb format) are now in $OUTDIR"
