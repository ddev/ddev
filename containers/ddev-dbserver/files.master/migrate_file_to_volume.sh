#!/bin/bash

set -x
set -eu
set -o pipefail

# This script is used to migrate a ddev bind-mounted database to a docker-volume mounted database
# It is actually just for the initial migration of v1.0.0-era databases to (hopefully) v1.1.0
# docker-volume-mounted databases, around 2018-08-02. It should end up being not useful within a few
# months.
#
# Run this command in the project directory:
# docker run -t -u "$(id -u):$(id -g)" -e SNAPSHOT_NAME=<migration_snapshot_name -v "$PWD/.ddev:/mnt/ddev_config" -v "$HOME/.ddev/<projectname>/mysql:/mysqlmount" --rm --entrypoint=/migrate_file_to_volume.sh drud/ddev-dbserver:<your_version>

if [ -z "${SNAPSHOT_NAME:-}" ] ; then
    echo "SNAPSHOT_NAME environment variable must be set"
    exit 101
fi

OUTDIR="/mnt/ddev_config/db_snapshots/${SNAPSHOT_NAME}"
SOCKET=/var/tmp/mysql.sock

mkdir -p $OUTDIR

if [ ! -d "/mysqlmount/mysql" ]; then
	echo "No mysql bind-mount directory was found, aborting"
	exit 102
fi

sudo mkdir -p /var/lib/mysql && sudo chmod -R ugo+w /var/lib/mysql
cp -r /mysqlmount/* /var/lib/mysql


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
        echo "MariaDB initialization startup process in progress... Try# $i"
        sleep 1
	done
	return 1
}

# Using --skip-grant-tables here becasue some old projects may not have working
# --user root --password root
mysqld --skip-networking --skip-grant-tables --socket=$SOCKET 2>&1 &
pid=$!
if ! serverwait ; then
    echo "Failed to get mysqld running"
    exit 103
fi

mariabackup --backup --target-dir=$OUTDIR --user root --socket=$SOCKET 2>&1
if [ "$?" != 0 ] ; then
    echo "Failed mariabackup command.";
    exit $((200+$?));
fi

# Wait for mysqld to exit
kill -s TERM "$pid" && wait "$pid"

echo "migration in: $OUTDIR"
