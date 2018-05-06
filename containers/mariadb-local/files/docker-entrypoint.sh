#!/bin/bash
set -x
set -eu
set -o pipefail

SOCKET=/var/tmp/mysql.sock

# Change  to UID/GID of the docker user
# We use the default assignment to zero to prevent triggering
# unbound variable exit. Since we chown all files to mysql, this
# must be done at the beginning of the script here.
if [ "${DDEV_UID:=0}" -gt "0" ] ; then
        echo "changing mysql user to uid: $DDEV_UID"
        usermod -o -u $DDEV_UID mysql
fi
if [ "${DDEV_GID:=0}" -gt 0 ] ; then
        echo "changing mysql group to gid: $DDEV_GID"
        groupmod -o -g $DDEV_GID mysql
fi

# If mariadb has not been initialized, initialize it.
# Then create our 'db', database, 'db' user, and permissions.
if [ ! -d "/var/lib/mysql/mysql" ]; then
	mkdir -p /var/lib/mysql
	chown -R mysql:mysql /var/lib/mysql /var/log/mysql*

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
		exit 1
	fi


	mysql_tzinfo_to_sql /usr/share/zoneinfo | mysql -uroot  mysql

	mysql -uroot <<EOF
		CREATE DATABASE IF NOT EXISTS $MYSQL_DATABASE;
		CREATE USER IF NOT EXISTS '$MYSQL_USER'@'%' IDENTIFIED BY '$MYSQL_PASSWORD';
		CREATE USER IF NOT EXISTS '$MYSQL_USER'@'localhost' IDENTIFIED BY '$MYSQL_PASSWORD';

		GRANT ALL ON $MYSQL_DATABASE.* TO '$MYSQL_USER'@'%';
		GRANT ALL ON $MYSQL_DATABASE.* TO '$MYSQL_USER'@'localhost';

		CREATE USER IF NOT EXISTS 'root'@'%' IDENTIFIED BY '$MYSQL_ROOT_PASSWORD';
		CREATE USER IF NOT EXISTS 'root'@'localhost' IDENTIFIED BY '$MYSQL_ROOT_PASSWORD';

		GRANT ALL ON *.* TO 'root'@'%' WITH GRANT OPTION;
		GRANT ALL ON *.* TO 'root'@'localhost' WITH GRANT OPTION;
EOF

	if ! kill -s TERM "$pid" || ! wait "$pid"; then
		echo >&2 'Mariadb initialization process failed.'
		exit 1
	fi

	echo 'Database initialized'
fi


echo
echo 'MySQL init process done. Ready for start up.'
echo

cp /root/mysqlclient.cnf /root/.my.cnf
chown -R mysql:mysql /var/lib/mysql /var/log/mysql*

echo "Starting mysqld."
exec mysqld --max-allowed-packet=${MYSQL_MAX_ALLOWED_PACKET:-16m}
