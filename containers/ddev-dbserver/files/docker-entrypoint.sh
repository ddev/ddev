#!/bin/bash
set -x
set -eu
set -o pipefail

# Normally /mnt/ddev_config will be mounted; config file requires it,
# so create it if it doesn't exist.
if [ ! -d /mnt/ddev_config/mysql ] ; then
  mkdir -p /mnt/ddev_config/mysql
  chmod ugo+rx /mnt/ddev_config /mnt/ddev_config/mysql
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
