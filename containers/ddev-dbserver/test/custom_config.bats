#!/usr/bin/env bats

# Run these tests from the repo root directory, for example
# bats tests

function setup {
    CONTAINER_NAME="testserver"
    HOSTPORT=33000
    MYTMPDIR="${HOME}/tmp/testserver-sh_${RANDOM}_$$"
    outdir="${HOME}/tmp/mariadb_testserver/output_${RANDOM}_$$"
    VOLUME="dbserver_test-${RANDOM}_$$"

    export MOUNTUID=33
    export MOUNTGID=33

    # Homebrew mysql client realy really wants /usr/local/etc/my.cnf.d
    if [ "${OS:-$(uname)}" != "Windows_NT" ] && [ ! -d /usr/local/etc/my.cnf.d ]; then
        mkdir -p /usr/local/etc/my.cnf.d || sudo mkdir -p /usr/local/etc/my.cnf.d
    fi
    docker rm -f ${CONTAINER_NAME} 2>/dev/null || true

    echo "# Starting image with database image $IMAGE"
    docker run -u "$MOUNTUID:$MOUNTGID" -v $VOLUME:/var/lib/mysql --mount "type=bind,src=$PWD/test/testdata,target=/mnt/ddev_config" --name=$CONTAINER_NAME -p $HOSTPORT:3306 -d $IMAGE
    containercheck
}

function teardown {
    docker rm -f $CONTAINER_NAME
    docker volume rm $VOLUME || true
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
	echo "# --- ddev-dbserver FAIL: information"
	docker ps -a
	docker logs $CONTAINER_NAME
	docker inspect $CONTAINER_NAME
	return 1
}

@test "test with mysql/utf.cnf override ${DB_TYPE} ${DB_VERSION}" {
    docker exec $CONTAINER_NAME sh -c 'grep collation-server /mnt/ddev_config/mysql/utf.cnf'
    mysql --user=root --password=root --skip-column-names --host=127.0.0.1 --port=$HOSTPORT -e "SHOW GLOBAL VARIABLES like \"collation_server\";" | grep "utf8_general_ci"
}



