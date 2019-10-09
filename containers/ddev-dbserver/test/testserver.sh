#!/bin/bash
set -x
set -euo pipefail

IMAGE="$1"  # Full image name with tag
MARIADB_VERSION="$2"
CONTAINER_NAME="testserver"
HOSTPORT=33000
MYTMPDIR="${HOME}/tmp/testserver-sh_${RANDOM}_$$"
outdir="${HOME}/tmp/mariadb_testserver/output_${RANDOM}_$$"
VOLUME="mariadbtest-${RANDOM}_$$"

export MOUNTUID=98
export MOUNTGID=98

mkdir -p /usr/local/etc/my.cnf.d || sudo mkdir -p /usr/local/etc/my.cnf.d

# Always clean up the container on exit.
function cleanup {
	echo "Removing ${CONTAINER_NAME}"
	docker rm -f $CONTAINER_NAME 2>/dev/null || true
	docker volume rm $VOLUME 2>/dev/null || true
	# We use MYTMPDIR for a bogus temp dir since mktemp -d creates a dir
    # outside a docker-mountable directory on macOS
    rm -rf $outdir/* $outdir/.git*
}

# Wait for container to be ready.
function containercheck {
    set +x
	for i in {60..0};
	do
		# status contains uptime and health in parenthesis, sed to return health
		status="$(docker ps --format "{{.Status}}" --filter "name=$CONTAINER_NAME" | sed  's/.*(\(.*\)).*/\1/')"
		if [[ "$status" == "healthy" ]]
		then
		    set -x
			return 0
		fi
		sleep 1
	done
	echo "--- ddev-dbserver FAIL: information"
	docker ps -a
	docker logs $CONTAINER_NAME
	docker inspect $CONTAINER_NAME
	return 1
}


cleanup

echo "Starting image with database image $IMAGE"
if ! docker run -u "$MOUNTUID:$MOUNTGID" -v $VOLUME:/var/lib/mysql --name=$CONTAINER_NAME -p $HOSTPORT:3306 -d $IMAGE; then
	echo "MySQL server start failed with error code $?"
	exit 101
fi

# Now that we've got a container running, we need to make sure to clean up
# at the end of the test run, even if something fails.
trap cleanup EXIT

echo "Waiting for database server to become ready..."
if ! containercheck; then
	exit 102
fi
echo "DB Server container started up successfully"

# Try basic connection using root user/password.
if ! mysql --user=root --password=root --database=mysql --host=127.0.0.1 --port=$HOSTPORT -e "SELECT 1;"; then
	exit 103
fi

# Test to make sure the db user and database are installed properly
if ! mysql -udb -pdb --database=db --host=127.0.0.1 --port=$HOSTPORT -e "SHOW TABLES;"; then
	exit 104
fi

# Test to make sure trigger capability works for 'db' user
# Triggers are needed occasionally, as for CiviCRM
if ! mysql -udb -pdb --database=db --host=127.0.0.1 --port=$HOSTPORT -e 'CREATE TABLE account (acct_num INT, amount DECIMAL(10,2)); CREATE TRIGGER ins_sum BEFORE INSERT ON account
       FOR EACH ROW SET @sum = @sum + NEW.amount;'; then
   echo "Unable to create trigger"
   exit 105
fi

# Make sure we have the right mysql version and can query it (and have root user setup)
OUTPUT=$(mysql --user=root --password=root --skip-column-names --host=127.0.0.1 --port=$HOSTPORT -e "SHOW VARIABLES like \"version\";")
RES=$?
if [ $RES -eq 0 ]; then
	echo "Successful mysql show variables, output=$OUTPUT"
fi
versionregex="version	$MARIADB_VERSION"
if [[ $OUTPUT =~ $versionregex ]];
then
	echo "Version check ok - found '$MARIADB_VERSION'"
else
	echo "Expected to see $versionregex. Actual output: $OUTPUT"
	exit 105
fi

# With the standard config, our collation should be utf8mb4_bin
mysql --user=root --password=root --skip-column-names --host=127.0.0.1 --port=$HOSTPORT -e "SHOW GLOBAL VARIABLES like \"collation_server\";" | grep "utf8mb4_bin"

cleanup

# Run with alternate configuration my.cnf mounted
# mysqld will ignore world-writeable config file, so we make it ro for sure
if ! docker run -u "$MOUNTUID:$MOUNTGID" -v $VOLUME:/var/lib/mysql -v $PWD/test/testdata:/mnt/ddev_config:ro --name=$CONTAINER_NAME -p $HOSTPORT:3306 -d $IMAGE; then
	echo "MySQL server start failed with error code $?"
	exit 106
fi

if ! containercheck; then
	exit 107
fi

# Make sure the custom config is present in the container.
docker exec -t $CONTAINER_NAME grep "collation-server" /mnt/ddev_config/mysql/utf.cnf

# With the custom config, our collation should be utf8_general_ci, not utf8mb4
mysql --user=root --password=root --skip-column-names --host=127.0.0.1 --port=$HOSTPORT -e "SHOW GLOBAL VARIABLES like \"collation_server\";" | grep "utf8_general_ci"

cleanup

#if [ "$MARIADB_VERSION" != "10.1" ] ; then
#    # Start up with a Mariadb 10.1 database and verify that it gets upgraded successfully to 10.2
#    docker volume rm $VOLUME && docker volume create $VOLUME
#    # Populate the volume with the contents of our 10.1 tarball. Here it doesn't matter that
#    # we're putting it in /var/lib/mysql, but it's put there just for clarity of purpose.
#    docker run -i --rm -v "$VOLUME:/var/lib/mysql" busybox tar -C /var/lib/mysql -zxf - <test/testdata/d6git_basic_mariadb_10_1.tgz
#    # Now start up the container with the populated volume
#    if ! docker run -u "$MOUNTUID:$MOUNTGID" -v "$VOLUME:/var/lib/mysql" --name=$CONTAINER_NAME -d $IMAGE; then
#        echo "MySQL server start failed with error code $?"
#        exit 109
#    fi
#    containercheck
#    # Here we should show an upgrade happening because this was a 10.1 database
#    (docker logs $CONTAINER_NAME 2>&1 | grep "Running mysql_upgrade because my_mariadb_version=10.2 is not the same as db_mariadb_version=10.1" >/dev/null 2>&1) || (echo "Failed to find mysql_upgrade clause in docker logs" && exit 4)
#
#    docker stop $CONTAINER_NAME && docker rm $CONTAINER_NAME
#    # Now run the container again with the same volume (now upgraded) and make sure we don't have the upgrade action
#    if ! docker run -u "$MOUNTUID:$MOUNTGID" -v "$VOLUME:/var/lib/mysql" --name=$CONTAINER_NAME -d $IMAGE; then
#        echo "MySQL server start failed with error code $?"
#        exit 110
#    fi
#    containercheck
#    (docker logs $CONTAINER_NAME 2>&1 | grep -v "Running mysql_upgrade" >/dev/null 2>&1) || (echo "Found  mysql_upgrade clause in docker logs when upgrade should not have happened" && exit 6)
#fi

echo "Tests passed"
exit 0
