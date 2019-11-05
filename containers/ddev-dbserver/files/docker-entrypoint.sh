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
#        echo "MariaDB initialization startup process in progress... Try# $i"
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
    sudo rm -rf /var/lib/mysql/*
  else
    echo "$snapshot_dir does not exist, not attempting restore of snapshot"
    unset snapshot_dir
    exit 101
  fi
fi

sudo chown -R "$(id -u):$(id -g)" /mysqlbase /var/lib/mysql

server_db_version=$(PATH=$PATH:/usr/sbin:/usr/local/bin:/usr/local/mysql/bin mysqld -V 2>/dev/null | awk '{sub( /\.[0-9]+(-.*)?$/, "", $3); print $3 }')

sudo chown -R "$UID:$(id -g)" /var/lib/mysql

# If we have extra mariadb cnf files,, copy them to where they go.
if [ -d /mnt/ddev_config/mysql -a "$(echo /mnt/ddev_config/mysql/*.cnf)" != "/mnt/ddev_config/mysql/*.cnf" ] ; then
  sudo cp /mnt/ddev_config/mysql/*.cnf /etc/mysql/conf.d
  sudo chmod -R ugo-w /etc/mysql/conf.d
fi

export BACKUPTOOL=mariabackup
if command -v xtrabackup; then BACKUPTOOL="xtrabackup"; fi

# If mariadb has not been initialized, copy in the base image from either the default starter image (/var/tmp/mysqlbase)
# or from a provided $snapshot_dir.
if [ ! -d "/var/lib/mysql/mysql" ]; then
    target=${snapshot_dir:-/mysqlbase/}
    name=$(basename $target)
    sudo rm -rf /var/lib/mysql/* /var/lib/mysql/.[a-z]* && sudo chmod -R ugo+w /var/lib/mysql
    sudo chmod -R ugo+r $target
    ${BACKUPTOOL} --prepare --skip-innodb-use-native-aio --target-dir "$target" --user=root --password=root --socket=$SOCKET 2>&1 | tee "/var/log/mariabackup_prepare_$name.log"
    ${BACKUPTOOL} --copy-back --skip-innodb-use-native-aio --force-non-empty-directories --target-dir "$target" --user=root --password=root --socket=$SOCKET 2>&1 | tee "/var/log/mariabackup_copy_back_$name.log"
    echo 'Database initialized from $target'
fi

if [ -f /var/lib/mysql/db_mariadb_version.txt ]; then
   database_db_version=$(cat /var/lib/mysql/db_mariadb_version.txt)
else
    database_db_version="unknown"
 fi

if [ "${server_db_version}" != "${database_db_version}" ]; then
   echo "Starting with db server version=${server_db_version} but database was created with '${database_db_version}'."
   echo "Attempting upgrade, but it may not work, you may need to export your database, 'ddev stop -RO', start, and reimport".

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
sudo mkdir -p /mnt/ddev-global-cache/bashhistory/${HOSTNAME}
sudo chown -R "$(id -u):$(id -g)" /mnt/ddev-global-cache/ ~/.my.cnf

echo
echo 'MySQL init process done. Ready for start up.'
echo

echo "Starting mysqld."
tail -f /var/log/mysqld.log &
exec mysqld --server-id=0
