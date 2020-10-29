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
    docker run -u "$MOUNTUID:$MOUNTGID" -v $VOLUME:/var/lib/mysql --name=$CONTAINER_NAME -p $HOSTPORT:3306 -d $IMAGE
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

@test "test user root and db access for ${DB_TYPE} ${DB_VERSION}" {
    mysql --user=root --password=root --database=mysql --host=127.0.0.1 --port=$HOSTPORT -e "SELECT 1;"
    mysql -udb -pdb --database=db --host=127.0.0.1 --port=$HOSTPORT -e "SHOW TABLES;"
}

@test "make sure trigger capability works correctly on ${DB_TYPE} ${DB_VERSION}" {
    mysql -udb -pdb --database=db --host=127.0.0.1 --port=$HOSTPORT -e 'CREATE TABLE account (acct_num INT, amount DECIMAL(10,2)); CREATE TRIGGER ins_sum BEFORE INSERT ON account
           FOR EACH ROW SET @sum = @sum + NEW.amount;'
}

@test "check correct mysql/mariadb version for ${DB_TYPE} ${DB_VERSION}" {
    reported_version=$(mysql --user=root --password=root --skip-column-names --host=127.0.0.1 --port=$HOSTPORT -e "SHOW VARIABLES like \"version\";" | awk '{sub( /\.[0-9]+(-.*)?$/, "", $2); print $2 }')
    echo "# Reported mysql/mariadb version=$reported_version and DB_VERSION=${DB_VERSION}"
    [ "${reported_version}" = ${DB_VERSION} ]
}

@test "look for utf8mb4_general_ci configured on ${DB_TYPE} ${DB_VERSION}" {
    mysql --user=root --password=root --skip-column-names --host=127.0.0.1 --port=$HOSTPORT -e "SHOW GLOBAL VARIABLES like \"collation_server\";" | grep "utf8mb4_general_ci"
}

