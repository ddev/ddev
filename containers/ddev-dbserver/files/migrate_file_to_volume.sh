#!/bin/bash

# set -x
set -eu
set -o pipefail

# This script is used to migrate a ddev bind-mounted database to a docker-volume mounted database
# It is actually just for the initial migration of v1.0.0-era databases to (hopefully) v1.1.0
# docker-volume-mounted databases, around 2018-08-02. It should end up being not useful within a few
# months.
#
# Run this command in the project directory:
# docker run -t -e SNAPSHOT_NAME=<migration_snapshot_name -v "$PWD/.ddev:/mnt/ddev_config" -v "$HOME/.ddev/<projectname>/mysql:/var/lib/mysql" --rm --entrypoint=/migrate_file_to_volume.sh drud/ddev-dbserver:<your_version>

if [ -z "${SNAPSHOT_NAME:-}" ] ; then
    echo "SNAPSHOT_NAME environment variable must be set"
    exit 1
fi

OUTDIR="/mnt/ddev_config/db_snapshots/${SNAPSHOT_NAME}"
SOCKET=/var/tmp/mysql.sock

mkdir -p $OUTDIR
chgrp mysql /var/tmp
chmod ug+rw /var/tmp

if [ ! -d "/var/lib/mysql/mysql" ]; then
	echo "No mysql bind-mount directory was found, aborting"
	exit 2
fi
chown -R mysql:mysql /var/lib/mysql /var/log/mysql*

mysqld --skip-networking &
pid="$!"

# Wait for the server to respond to mysqladmin ping, or fail if it never does,
# or if the process dies.
for i in {60..0}; do
	if mysqladmin ping -uroot --socket=$SOCKET >/dev/null 2>&1; then
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

mariabackup --backup --target-dir=$OUTDIR --user root --password root --socket=$SOCKET 2>/dev/null

# Wait for mysqld to exit
kill -s TERM "$pid"&& wait "$pid"

echo "migration in: $OUTDIR"
