#!/bin/bash
set -x
set -eu
set -o pipefail

# If we have a restore_snapshot arg, get the snapshot directory
# otherwise, fail and abort startup
if [ $# = "2" -a "${1:-}" = "restore_snapshot" ] ; then
  snapshot_dir="/mnt/ddev_config/db_snapshots/${2:-nothingthere}"
  if [ -d $snapshot_dir ] ; then
    echo "Restoring from snapshot directory $snapshot_dir"
    sudo rm -rf /var/lib/mysql/*
  else
    echo "$snapshot_dir does not exist, not attempting restore of snapshot"
    unset snapshot_dir
    exit 3
  fi
fi

sudo chown -R $UID:$(id -g) /var/lib/mysql

# If we have extra mariadb cnf files,, copy them to where they go.
if [ -d /mnt/ddev_config/mysql -a "$(echo /mnt/ddev_config/mysql/*.cnf)" != "/mnt/ddev_config/mysql/*.cnf" ] ; then
  cp /mnt/ddev_config/mysql/*.cnf /etc/mysql/conf.d
  chmod ugo-w /etc/mysql/conf.d/*
fi

# If mariadb has not been initialized, copy in the base image.
if [ ! -d "/var/lib/mysql/mysql" ]; then
    target=${snapshot_dir:-/var/tmp/mysqlbase/}
    name=$(basename $target)
    sudo rm -rf /var/lib/mysql && sudo mkdir -p /var/lib/mysql && sudo chown $USER /var/lib/mysql
	mariabackup --prepare --target-dir "$target" --user root --password root --socket=/var/tmp/mysql.sock 2>"/var/log/mariabackup_prepare_$name.log"
	mariabackup --copy-back --target-dir "$target" --user root --password root --socket=/var/tmp/mysql.sock 2>"/var/log/mariabackup_copy_back_$name.log"
	echo 'Database initialized from $target'
fi


echo
echo 'MySQL init process done. Ready for start up.'
echo

echo "Starting mysqld."
tail -f /var/log/mysqld.log &
exec mysqld
