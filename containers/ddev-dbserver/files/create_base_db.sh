#!/bin/bash

set -eu
set -o pipefail

SOCKET=/var/tmp/mysql.sock
OUTDIR=/mysqlbase

mkdir -p ${OUTDIR}
chown -R "$(id -u):$(id -g)" $OUTDIR

chmod ugo+w /var/tmp
mkdir -p /var/lib/mysql /mnt/ddev_config/mysql && rm -f /var/lib/mysql/* && chmod -R ugo+w /var/lib/mysql

echo 'Initializing mysql'
mysqld --version
mysqld_version=$(mysqld --version | awk '{ gsub(/-log/, ""); gsub(/\.[0-9]+$/, "", $3);  print $3}')
echo version=$mysqld_version
# Oracle mysql 5.7+ deprecates mysql_install_db
if [ "${mysqld_version}" = "5.7" ] || [  "${mysqld_version%%%.*}" = "8.0" ]; then
    mysqld --initialize-insecure --datadir=/var/lib/mysql --server-id=0
else
    # mysql 5.5 requires running mysql_install_db in /usr/local/mysql
    if command -v mysqld | grep usr.local; then
        cd /usr/local/mysql
    fi
    mysql_install_db --force --datadir=/var/lib/mysql
fi
echo "Starting mysqld --skip-networking --socket=${SOCKET}"
mysqld --user=root --socket=$SOCKET --innodb_log_file_size=48M --skip-networking --datadir=/var/lib/mysql --server-id=0 --skip-log-bin &
pid="$!"

# Wait for the server to respond to mysqladmin ping, or fail if it never does,
# or if the process dies.
for i in {90..0}; do
	if mysqladmin ping -uroot --socket=$SOCKET 2>/dev/null; then
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


mysql_tzinfo_to_sql /usr/share/zoneinfo | mysql -uroot  mysql

mysql -uroot <<EOF
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

mysqladmin -uroot password root

if [  "${mysqld_version%%%.*}" = "8.0" ]; then
    mysql -uroot -proot <<EOF
        ALTER USER 'db'@'%' IDENTIFIED WITH mysql_native_password BY '$MYSQL_PASSWORD';
        ALTER USER 'root'@'%' IDENTIFIED WITH mysql_native_password BY '$MYSQL_ROOT_PASSWORD';
        ALTER USER 'root'@'localhost' IDENTIFIED WITH mysql_native_password BY '$MYSQL_ROOT_PASSWORD';
EOF
fi


rm -rf $OUTDIR/*

backuptool=mariabackup
if command -v xtrabackup; then backuptool="xtrabackup --datadir=/var/lib/mysql"; fi
${backuptool} --backup --target-dir=$OUTDIR --user=root --password=root --socket=$SOCKET

# Initialize with current mariadb_version
my_mariadb_version=$(PATH=$PATH:/usr/sbin:/usr/local/bin:/usr/local/mysql/bin mysqld -V 2>/dev/null | awk '{sub( /\.[0-9]+(-.*)?$/, "", $3); print $3 }')
echo $my_mariadb_version >$OUTDIR/db_mariadb_version.txt

if ! kill -s TERM "$pid" || ! wait "$pid"; then
	echo >&2 'Database initialization process failed.'
	exit 5
fi

echo "The startup database files (in mariabackup/xtradb format) are now in $OUTDIR"
