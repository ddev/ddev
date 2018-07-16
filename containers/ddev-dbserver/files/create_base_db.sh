#!/bin/bash

set -x
set -eu
set -o pipefail

# This script can be used to create a bare database directory for use by
# ddev startup. It can be run from the host with:
# docker run -it -v "$PWD/files/var/tmp/mysqlbase:/mysqlbase" --rm --entrypoint=/create_base_db.sh drud/ddev-dbserver:<your_version>

SOCKET=/var/tmp/mysql.sock
OUTDIR=/mysqlbase

if [ ! -d $OUTDIR ] ; then
  echo "The required output directory $OUTDIR does not seem to exist."
  exit 1
fi

# For this script we don't want the defaults in .my.cnf
# However, this script is never run on a normal startup, so we can just throw it away.
rm -f /home/.my.cnf

chgrp mysql /var/tmp
chmod ug+rw /var/tmp

if [ -d "/var/lib/mysql/mysql" ]; then
	echo "A mysql installation already exists, aborting"
	exit 2
fi
mkdir -p /var/lib/mysql /mnt/ddev_config/mysql
chown -R mysql:mysql /var/lib/mysql /mnt/ddev_config/mysql /var/log/mysql*

echo 'Initializing mysql'
mysql_install_db
echo 'Starting mysqld --skip-networking'
mysqld --skip-networking &
pid="$!"

# Wait for the server to respond to mysqladmin ping, or fail if it never does,
# or if the process dies.
for i in {60..0}; do
	if mysqladmin ping -uroot --socket=$SOCKET; then
		break
	fi
	# Test to make sure we got it started in the first place. kill -s 0 just tests to see if process exists.
	if ! kill -s 0 $pid 2>/dev/null; then
		echo "MariaDB initialization startup failed"
		exit 3
	fi
	echo "MariaDB initialization startup process in progress... Try# $i"
	sleep 1
done
if [ "$i" -eq 0 ]; then
	echo 'MariaDB initialization startup process timed out.'
	exit 4
fi


mysql_tzinfo_to_sql /usr/share/zoneinfo | mysql -uroot  mysql

mysql -uroot <<EOF
	CREATE DATABASE IF NOT EXISTS $MYSQL_DATABASE;
	CREATE USER IF NOT EXISTS '$MYSQL_USER'@'%' IDENTIFIED BY '$MYSQL_PASSWORD';
	CREATE USER IF NOT EXISTS '$MYSQL_USER'@'localhost' IDENTIFIED BY '$MYSQL_PASSWORD';

	GRANT ALL ON $MYSQL_DATABASE.* TO '$MYSQL_USER'@'%';
	GRANT ALL ON $MYSQL_DATABASE.* TO '$MYSQL_USER'@'localhost';

	CREATE USER IF NOT EXISTS 'root'@'%' IDENTIFIED BY '$MYSQL_ROOT_PASSWORD';
	GRANT ALL ON *.* TO 'root'@'%' WITH GRANT OPTION;
	GRANT ALL ON *.* to 'root'@'localhost' IDENTIFIED BY '$MYSQL_ROOT_PASSWORD';
	FLUSH PRIVILEGES;
	FLUSH TABLES;
EOF

mariabackup --backup --target-dir=$OUTDIR --user root --password root --socket=$SOCKET

if ! kill -s TERM "$pid" || ! wait "$pid"; then
	echo >&2 'Mariadb initialization process failed.'
	exit 5
fi

echo "The startup database files (in mariabackup format) are now in $OUTDIR"
