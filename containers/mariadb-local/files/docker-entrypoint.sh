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
	# The tarball should include only the contents of the db and mysql directories.
	tar --no-same-owner -C /var/lib/mysql -zxf /var/tmp/mariadb_10.1_base_db.tgz
	echo 'Database initialized'
fi


echo
echo 'MySQL init process done. Ready for start up.'
echo

echo "Starting mysqld."
tail -f /var/log/mysqld.log &
exec mysqld --max-allowed-packet=${MYSQL_MAX_ALLOWED_PACKET:-16m}
