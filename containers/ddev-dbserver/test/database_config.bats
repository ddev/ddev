#!/usr/bin/env bats

# Run these tests from the repo root directory, for example
# ./test/bats tests

load functions.sh

function setup {
  basic_setup

  # echo "# Starting container using: docker run --rm -u "$MOUNTUID:$MOUNTGID" -v $VOLUME:/var/lib/mysql --name=$CONTAINER_NAME -p $HOSTPORT:3306 -d $IMAGE" >&3
  docker run -u "$MOUNTUID:$MOUNTGID" -v $VOLUME:/var/lib/mysql:nocopy --name=$CONTAINER_NAME -p $HOSTPORT:3306 -d $IMAGE
  containercheck
}

# Load configurations from file
function load_configs {
  local config_file="$PWD/test/testdata/database_config/${DB_TYPE}_${DB_VERSION}.ini"
  declare -gA configs

  if [[ ! -f "$config_file" ]]; then
    echo "Error: Configuration file $config_file not found!" >&2
    exit 1
  fi

  while IFS='=' read -r key value; do
    # Skip empty lines and comments
    [[ -z "$key" || "$key" =~ ^# ]] && continue
    # Trim surrounding whitespace
    key=$(echo "$key" | xargs)
    value=$(echo "$value" | xargs)
    configs["$key"]="$value"
  done < "$config_file"
}

@test "Check for expected configuration on ${DB_TYPE} ${DB_VERSION}" {
    load_configs

    # Iterate over each configuration and validate
  for config in "${!configs[@]}"; do
    expected_value="${configs[$config]}"
    value=$(mysql ${SKIP_SSL} --user=root --password=root --skip-column-names --host=127.0.0.1 --port=$HOSTPORT \
  -e "SHOW VARIABLES LIKE '${config}';" | awk -F'\t' '{print $2}')
    echo "# Checking ${config}: Expected=${expected_value}, Found=${value}"
    [ "${value}" = "${expected_value}" ]
  done
}

