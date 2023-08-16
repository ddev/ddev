#!/bin/bash
set -x
set -eu
set -o pipefail

SOCKET=/var/tmp/mysql.sock
rm -f /tmp/healthy

# We can't just switch on database type here, because early versions
# of mariadb used xtrabackup
export BACKUPTOOL=mariabackup
export STREAMTOOL=mbstream
if command -v xtrabackup >/dev/null 2>&1 ; then
  BACKUPTOOL="xtrabackup"
  STREAMTOOL="xbstream"
fi


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

# There may be a snapshots volume mounted on /mnt/snapshots
# But if not, it means we can use snapshots from /mnt/ddev_config/snapshots
if [ ! -d /mnt/snapshots ]; then
  ln -s /mnt/ddev_config/ddev_snapshots /mnt/snapshots
fi
# If we have a restore_snapshot arg, get the snapshot file/directory
# otherwise, fail and abort startup
if [ $# = "2" ] && [ "${1:-}" = "restore_snapshot" ] ; then
  snapshot_basename=${2:-nothingthere}
  snapshot="/mnt/snapshots/${snapshot_basename}"
  # If a gzipped snapshot is passed in, unzip it
  if [ -f "$snapshot" ] && [ "${snapshot_basename##*.}" = "gz" ]; then
    echo "Restoring from snapshot file $snapshot"
    target="/var/tmp/${snapshot_basename}"
    mkdir -p "${target}"
    cd "${target}"
    gunzip -c ${snapshot} | ${STREAMTOOL} -x
    rm -rf /var/lib/mysql/*
  # Otherwise use it as is from the directory
  elif [ -d "$snapshot" ] ; then
    echo "Restoring from snapshot directory $snapshot"
    target="${snapshot}"
    # Ugly macOS .DS_Store in this directory can break the restore
    find ${snapshot} -name .DS_Store -print0 | xargs rm -f
    rm -rf /var/lib/mysql/*
  else
    echo "$snapshot does not exist, not attempting restore of snapshot"
    unset snapshot
    exit 101
  fi
fi

PATH=$PATH:/usr/sbin:/usr/local/bin:/usr/local/mysql/bin mysqld -V 2>/dev/null  | awk '{print $3}' > /tmp/raw_mysql_version.txt
# mysqld -V gives us the version in the form of 5.7.28-log for mysql or
# 5.5.64-MariaDB-1~trusty for MariaDB. Detect database type and version and output
# mysql-8.0 or mariadb-10.5.
server_db_version=$(awk -F- '{ sub( /\.[0-9]+(-.*)?$/, "", $1); server_type="mysql"; if ($2 ~ /^MariaDB/) { server_type="mariadb" }; print server_type "_" $1 }' /tmp/raw_mysql_version.txt)
rm -f /tmp/raw_mysql_version.txt

# If we have extra mariadb cnf files, copy them to where they go.
if [ -d /mnt/ddev_config/mysql ] && [ "$(echo /mnt/ddev_config/mysql/*.cnf)" != "/mnt/ddev_config/mysql/*.cnf" ] ; then
  cp /mnt/ddev_config/mysql/*.cnf /etc/mysql/conf.d
  # Ignore errors on files such as .gitmanaged
  chmod -f -R ugo-w /etc/mysql/conf.d/*
fi


# If mariadb has not been initialized, copy in the base image from either the default starter image (/mysqlbase/base_db.gz)
# or from a provided $snapshot_dir.
if [ ! -f "/var/lib/mysql/db_mariadb_version.txt" ]; then
    # If snapshot_dir is not set, this is a normal startup, so
    # tell healthcheck to wait by touching /tmp/initializing
    if [ -z "${snapshot:-}" ] ; then
      touch /tmp/initializing
    fi
    if [ "${target:-}" = "" ]; then
      target=${snapshot:-/var/tmp/base_db}
      mkdir -p ${target} && cd ${target}
      snapshot=/mysqlbase/base_db.gz
      gunzip -c ${snapshot} | ${STREAMTOOL} -x
    fi
    name=$(basename $target)

    rm -rf /var/lib/mysql/* /var/lib/mysql/.[a-z]*
    ${BACKUPTOOL} --prepare --skip-innodb-use-native-aio --target-dir "$target" --user=root --password=root 2>&1 | tee "/var/log/mariabackup_prepare_$name.log"
    ${BACKUPTOOL} --copy-back --skip-innodb-use-native-aio --force-non-empty-directories --target-dir "$target" --user=root --password=root 2>&1 | tee "/var/log/mariabackup_copy_back_$name.log"
    echo $server_db_version >/var/lib/mysql/db_mariadb_version.txt
    echo "Database initialized from ${target}"
    rm -f /tmp/initializing
fi

# db_mariadb_version.txt may be "mariadb_10.5" or "mysql_8.0" or old "10.0" or "8.0"
database_db_version=$(cat /var/lib/mysql/db_mariadb_version.txt)
# If we have an old-style reference, like "10.5", prefix it with the database type
if [ "${database_db_version#*_}" = "${database_db_version}" ]; then
  database_db_version="${server_db_version%_*}_${database_db_version}"
fi

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
mkdir -p /mnt/ddev-global-cache/{bashhistory,mysqlhistory}/${HOSTNAME} || true

echo
echo 'MySQL init process done. Ready for start up.'
echo

echo "Starting mysqld."
tail -f /var/log/mysqld.log &
exec mysqld --server-id=0
