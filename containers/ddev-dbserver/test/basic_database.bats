#!/usr/bin/env bats

# Run these tests from the repo root directory, for example
# ./test/bats tests

load functions.sh

function setup {
  basic_setup

  echo "# Starting container using: docker run --rm -u "$MOUNTUID:$MOUNTGID" -v $VOLUME:/var/lib/mysql --name=$CONTAINER_NAME -p $HOSTPORT:3306 -d $IMAGE"
  docker run -u "$MOUNTUID:$MOUNTGID" -v $VOLUME:/var/lib/mysql:nocopy --name=$CONTAINER_NAME -p $HOSTPORT:3306 -d $IMAGE
  containercheck
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

