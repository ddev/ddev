#!/bin/bash
set -x
set -euo pipefail

IMAGE="$1"  # Full image name with tag
MYSQL_VERSION="$2"
CONTAINER_NAME="testserver"
HOSTPORT=33000
MYTMPDIR=~/tmp/testserver-sh_$$

# Always clean up the container on exit.
function cleanup {
	echo "Removing ${CONTAINER_NAME}"
	docker rm -f $CONTAINER_NAME 2>/dev/null || true
}

# Wait for container to be ready.
function containercheck {
	for i in {60..0};
	do
		# status contains uptime and health in parenthesis, sed to return health
		status="$(docker ps --format "{{.Status}}" --filter "name=$CONTAINER_NAME" | sed  's/.*(\(.*\)).*/\1/')"
		if [[ "$status" == "healthy" ]]
		then
			return 0
		fi
		sleep 1
	done
	return 1
}


# Just to make sure we're starting with a clean environment.
cleanup

# We use MYTMPDIR for a bogus temp dir since mktemp -d creates a dir
# outside a docker-mountable directory on macOS
mkdir -p $MYTMPDIR
rm -rf $MYTMPDIR/*

echo "Starting image with database image $IMAGE"
if ! docker run -u "$(id -u):$(id -g)" -v $MYTMPDIR:/var/lib/mysql --name=$CONTAINER_NAME -p $HOSTPORT:3306 -d $IMAGE; then
	echo "MySQL server start failed with error code $?"
	exit 2
fi

# Now that we've got a container running, we need to make sure to clean up
# at the end of the test run, even if something fails.
trap cleanup EXIT

echo "Waiting for database server to become ready..."
if ! containercheck; then
	echo "Container did not become ready"
	exit 1
fi
echo "Connected to mysql server."

# Try basic connection using root user/password.
if ! mysql --user=root --password=root --database=mysql --host=127.0.0.1 --port=$HOSTPORT -e "SELECT 1;"; then
	exit 2;
fi

# Test to make sure the db user and database are installed properly
if ! mysql -udb -pdb --database=db --host=127.0.0.1 --port=$HOSTPORT -e "SHOW TABLES;"; then
	exit 3
fi

# Make sure we have the right mysql version and can query it (and have root user setup)
OUTPUT=$(mysql --user=root --password=root --skip-column-names --host=127.0.0.1 --port=$HOSTPORT -e "SHOW VARIABLES like \"version\";")
RES=$?
if [ $RES -eq 0 ]; then
	echo "Successful mysql show variables, output=$OUTPUT"
fi
versionregex="version	$MYSQL_VERSION"
if [[ $OUTPUT =~ $versionregex ]];
then
	echo "Version check ok - found '$MYSQL_VERSION'"
else
	echo "Expected to see $versionregex. Actual output: $OUTPUT"
	exit 4
fi

# With the standard config, our collation should be utf8mb4_bin
mysql --user=root --password=root --skip-column-names --host=127.0.0.1 --port=$HOSTPORT -e "SHOW GLOBAL VARIABLES like \"collation_server\";" | grep "utf8mb4_bin"

cleanup

# Run with alternate configuration my.cnf mounted
if ! docker run -u "$(id -u):$(id -g)" -v $MYTMPDIR:/var/lib/mysql -v $PWD/test/testdata:/mnt/ddev_config --name=$CONTAINER_NAME -p $HOSTPORT:3306 -d $IMAGE; then
	echo "MySQL server start failed with error code $?"
	exit 3
fi

if ! containercheck; then
	echo "Container did not become ready"
	exit 5
fi

# Make sure the custom config is present in the container.
docker exec -it $CONTAINER_NAME grep "collation-server" /mnt/ddev_config/mysql/utf.cnf

# With the custom config, our collation should be utf8_general_ci, not utf8mb4
mysql --user=root --password=root --skip-column-names --host=127.0.0.1 --port=$HOSTPORT -e "SHOW GLOBAL VARIABLES like \"collation_server\";" | grep "utf8_general_ci"

cleanup

# Test that the create_base_db.sh script can create a starter tarball.
# This one runs as root, and ruins the underlying host mount on linux (makes it owned by root)
outdir=/tmp/output_$$
mkdir /tmp/output_$$
docker run -it -v "$outdir:/mysqlbase" --rm --entrypoint=/create_base_db.sh $IMAGE
if [ ! -f $outdir/mariadb_10.1_base_db.tgz ] ; then
  echo "Failed to build test starter tarball for mariadb."
  exit 4
fi
rm -rf $outdir $MYTMPDIR

echo "Tests passed"
exit 0
