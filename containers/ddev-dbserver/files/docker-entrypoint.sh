#!/bin/bash
set -x
set -eu
set -o pipefail

SOCKET=/var/tmp/mysql.sock
rm -f /tmp/healthy

# Wait for mysql server to be ready.
function serverwait {
	for i in {60..0};
	do
        if mysqladmin ping -uroot --socket=$SOCKET >/dev/null 2>&1; then
            return 0
        fi
        # Test to make sure we got it started in the first place. kill -s 0 just tests to see if process exists.
        if ! kill -s 0 $pid 2>/dev/null; then
            echo "MariaDB initialization startup failed"
            return 2
        fi
        sleep 1
	done
	return 1
}

# If we have a restore_snapshot arg, get the snapshot directory
# otherwise, fail and abort startup
if [ $# = "2" -a "${1:-}" = "restore_snapshot" ] ; then
  snapshot_dir="/mnt/ddev_config/db_snapshots/${2:-nothingthere}"
  if [ -d "$snapshot_dir" ] ; then
    echo "Restoring from snapshot directory $snapshot_dir"
    # Ugly macOS .DS_Store in this directory can break the restore
    find ${snapshot_dir} -name .DS_Store -print0 | xargs rm -f
    rm -rf /var/lib/mysql/*
  else
    echo "$snapshot_dir does not exist, not attempting restore of snapshot"
    unset snapshot_dir
    exit 101
  fi
fi

server_db_version=$(PATH=$PATH:/usr/sbin:/usr/local/bin:/usr/local/mysql/bin mysqld -V 2>/dev/null | awk '{sub( /\.[0-9]+(-.*)?$/, "", $3); print $3 }')

# If we have extra mariadb cnf files,, copy them to where they go.
if [ -d /mnt/ddev_config/mysql -a "$(echo /mnt/ddev_config/mysql/*.cnf)" != "/mnt/ddev_config/mysql/*.cnf" ] ; then
  echo "!includedir /mnt/ddev_config/mysql" >/etc/mysql/conf.d/ddev.cnf
fi

export BACKUPTOOL=mariabackup
if command -v xtrabackup; then BACKUPTOOL="xtrabackup"; fi

# If mariadb has not been initialized, copy in the base image from either the default starter image (/mysqlbase)
# or from a provided $snapshot_dir.
if [ ! -f "/var/lib/mysql/db_mariadb_version.txt" ]; then
    # If snapshot_dir is not set, this is a normal startup, so
    # tell healthcheck to wait by touching /tmp/initializing
    if [ -z "${snapshot_dir:-}" ] ; then
      touch /tmp/initializing
    fi
    target=${snapshot_dir:-/mysqlbase/}
    name=$(basename $target)
    rm -rf /var/lib/mysql/* /var/lib/mysql/.[a-z]*
    ${BACKUPTOOL} --prepare --skip-innodb-use-native-aio --target-dir "$target" --user=root --password=root --socket=$SOCKET 2>&1 | tee "/var/log/mariabackup_prepare_$name.log"
    ${BACKUPTOOL} --copy-back --skip-innodb-use-native-aio --force-non-empty-directories --target-dir "$target" --user=root --password=root --socket=$SOCKET 2>&1 | tee "/var/log/mariabackup_copy_back_$name.log"
    echo "Database initialized from ${target}"
    rm -f /tmp/initializing
fi

database_db_version=$(cat /var/lib/mysql/db_mariadb_version.txt)

if [ "${server_db_version}" != "${database_db_version}" ]; then
   echo "Starting with db server version=${server_db_version} but database was created with '${database_db_version}'."
   echo "Attempting upgrade, but it may not work, you may need to export your database, 'ddev delete --omit-snapshot', start, and reimport".

    PATH=$PATH:/usr/sbin:/usr/local/bin:/usr/local/mysql/bin mysqld --skip-networking --skip-grant-tables --socket=$SOCKET >/tmp/mysqld_temp_startup.log 2>&1 &
    pid=$!
    set +x
    if ! serverwait ; then
        echo "Failed to get mysqld running to run mysql_upgrade"
        exit 103
    fi
    set -x
    echo "Attempting mysql_upgrade because db server version ${server_db_version} is not the same as database db version ${database_db_version}"
    mysql_upgrade --socket=$SOCKET
    kill $pid
fi

# And update the server db version we have here.
echo $server_db_version >/var/lib/mysql/db_mariadb_version.txt

cp -r /home/{.my.cnf,.bashrc} ~/
mkdir -p /mnt/ddev-global-cache/bashhistory/${HOSTNAME} || true

echo
echo 'MySQL init process done. Ready for start up.'
echo

echo "Starting mysqld."
tail -f /var/log/mysqld.log &
exec mysqld --server-id=0
