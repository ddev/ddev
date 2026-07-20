#!/usr/bin/env bats

# Run these tests from the repo root directory, for example
# ./test/bats tests
#
# Exercises the base_db seed lookup in docker-entrypoint.sh, which is
# checked in priority order:
#   1. /mnt/snapshots/initializer-<db_type>_<db_version>.*  - project-supplied,
#      living alongside regular snapshots in .ddev/db_snapshots
#   2. /mysqlbase/custom/base_db.*                          - baked into a derived image
#   3. /mysqlbase/base_db.*                                 - the stock DDEV starter database
# A .zst seed is preferred over .gz at each location; .gz is exercised too,
# since db versions without zstd (e.g. MariaDB 5.5) still produce it.

load functions.sh

function setup_file {
  export SEED_DIR="${BATS_FILE_TMPDIR}/base_db_seeds"
  mkdir -p "${SEED_DIR}"

  # Match whatever compressor create_base_db.sh actually used to build this
  # image's stock seed, so the override seeds we build here use the same
  # extension the entrypoint expects for this ${DB_TYPE} ${DB_VERSION}.
  if docker run --rm --entrypoint bash "${IMAGE}" -c 'command -v zstdmt || command -v zstd' >/dev/null 2>&1; then
    export SEED_EXT="zst"
    export SEED_COMPRESS="zstdmt --quiet"
  else
    export SEED_EXT="gz"
    export SEED_COMPRESS="gzip"
  fi

  # Matches the server_db_version computed by docker-entrypoint.sh, e.g. mariadb_11.8
  export INITIALIZER_NAME="initializer-${DB_TYPE}_${DB_VERSION}.${SEED_EXT}"

  make_seed "marker_seed_a" "${SEED_DIR}/seed_a.${SEED_EXT}"
  make_seed "marker_seed_b" "${SEED_DIR}/seed_b.${SEED_EXT}"
}

# Build a base_db seed containing a single marker table, using the same
# backup tool (mariabackup/xtrabackup via xbstream) the entrypoint expects.
function make_seed {
  local marker_table=$1
  local outfile=$2
  local name="seedbuilder-$$-${RANDOM}"
  local vol="seedbuilder-vol-$$-${RANDOM}"

  docker run --rm -v "${vol}:/var/lib/mysql:nocopy" busybox:stable chown -R 33:33 /var/lib/mysql
  docker run -d --user=33:33 -v "${vol}:/var/lib/mysql:nocopy" --name="${name}" "${IMAGE}"

  local health=""
  for i in {60..0}; do
    health="$(docker inspect --format '{{json .State.Health }}' "${name}" | jq -r .Status)"
    if [ "${health}" = "healthy" ]; then
      break
    fi
    sleep 1
  done
  [ "${health}" = "healthy" ]

  docker exec "${name}" mysql -udb -pdb --database=db -e "CREATE TABLE ${marker_table} (id INT);"
  # mariabackup/mbstream on MariaDB, xtrabackup/xbstream on MySQL (and some
  # older MariaDB) -- same detection docker-entrypoint.sh itself uses.
  docker exec "${name}" bash -c "
    backuptool=mariabackup; streamtool=mbstream
    if command -v xtrabackup >/dev/null 2>&1; then backuptool=xtrabackup; streamtool=xbstream; fi
    \${backuptool} --backup --stream=\${streamtool} --user=root --password=root --socket=/var/tmp/mysql.sock 2>/tmp/seed.log | ${SEED_COMPRESS}
  " >"${outfile}"

  docker rm -f "${name}" >/dev/null
  docker volume rm "${vol}" >/dev/null
}

function setup {
  basic_setup
}

@test "stock base_db seed is used when no override is present for ${DB_TYPE} ${DB_VERSION}" {
  docker run -u "$MOUNTUID:$MOUNTGID" -v $VOLUME:/var/lib/mysql:nocopy --name=$CONTAINER_NAME -p $HOSTPORT:3306 -d $IMAGE
  containercheck
  run docker logs $CONTAINER_NAME
  [[ "$output" == *"snapshot=/mysqlbase/base_db.${SEED_EXT}"* ]]
  mysql ${SKIP_SSL:-} -udb -pdb --database=db --host=127.0.0.1 --port=$HOSTPORT -e "SHOW TABLES;"
}

@test "project-mounted initializer seed overrides the stock seed for ${DB_TYPE} ${DB_VERSION}" {
  docker run -u "$MOUNTUID:$MOUNTGID" -v $VOLUME:/var/lib/mysql:nocopy \
    -v "${SEED_DIR}/seed_a.${SEED_EXT}:/mnt/snapshots/${INITIALIZER_NAME}:ro" \
    --name=$CONTAINER_NAME -p $HOSTPORT:3306 -d $IMAGE
  containercheck
  run docker logs $CONTAINER_NAME
  [[ "$output" == *"snapshot=/mnt/snapshots/${INITIALIZER_NAME}"* ]]
  run mysql ${SKIP_SSL:-} -udb -pdb --database=db --host=127.0.0.1 --port=$HOSTPORT -e "SHOW TABLES;"
  [[ "$output" == *"marker_seed_a"* ]]
}

@test "derived-image custom base_db seed overrides the stock seed for ${DB_TYPE} ${DB_VERSION}" {
  docker run -u "$MOUNTUID:$MOUNTGID" -v $VOLUME:/var/lib/mysql:nocopy \
    -v "${SEED_DIR}/seed_b.${SEED_EXT}:/mysqlbase/custom/base_db.${SEED_EXT}:ro" \
    --name=$CONTAINER_NAME -p $HOSTPORT:3306 -d $IMAGE
  containercheck
  run docker logs $CONTAINER_NAME
  [[ "$output" == *"snapshot=/mysqlbase/custom/base_db.${SEED_EXT}"* ]]
  run mysql ${SKIP_SSL:-} -udb -pdb --database=db --host=127.0.0.1 --port=$HOSTPORT -e "SHOW TABLES;"
  [[ "$output" == *"marker_seed_b"* ]]
}

@test "project-mounted initializer seed takes precedence over derived-image seed for ${DB_TYPE} ${DB_VERSION}" {
  docker run -u "$MOUNTUID:$MOUNTGID" -v $VOLUME:/var/lib/mysql:nocopy \
    -v "${SEED_DIR}/seed_a.${SEED_EXT}:/mnt/snapshots/${INITIALIZER_NAME}:ro" \
    -v "${SEED_DIR}/seed_b.${SEED_EXT}:/mysqlbase/custom/base_db.${SEED_EXT}:ro" \
    --name=$CONTAINER_NAME -p $HOSTPORT:3306 -d $IMAGE
  containercheck
  run docker logs $CONTAINER_NAME
  [[ "$output" == *"snapshot=/mnt/snapshots/${INITIALIZER_NAME}"* ]]
  run mysql ${SKIP_SSL:-} -udb -pdb --database=db --host=127.0.0.1 --port=$HOSTPORT -e "SHOW TABLES;"
  [[ "$output" == *"marker_seed_a"* ]]
}
