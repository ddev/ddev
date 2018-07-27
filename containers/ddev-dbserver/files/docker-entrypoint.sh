#!/bin/bash
set -x
set -eu
set -o pipefail

# If we have extra mariadb cnf files,, copy them to where they go.
if [ -d /mnt/ddev_config/mysql -a "$(echo /mnt/ddev_config/mysql/*.cnf)" != "/mnt/ddev_config/mysql/*.cnf" ] ; then
  cp /mnt/ddev_config/mysql/*.cnf /etc/mysql/conf.d
  chmod ugo-w /etc/mysql/conf.d/*
fi

# If mariadb has not been initialized, copy in the base image.
if [ ! -d "/var/lib/mysql/mysql" ]; then
	mkdir -p /var/lib/mysql
	mariabackup --prepare --target-dir /var/tmp/mysqlbase/ --user root --password root --socket=/var/tmp/mysql.sock
	mariabackup --copy-back --target-dir /var/tmp/mysqlbase/ --user root --password root --socket=/var/tmp/mysql.sock
	echo 'Database initialized'
fi


echo
echo 'MySQL init process done. Ready for start up.'
echo

echo "Starting mysqld."
tail -f /var/log/mysqld.log &
exec mysqld
