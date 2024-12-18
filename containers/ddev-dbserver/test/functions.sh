#!/usr/bin/env bash

SKIP_SSL="--skip-ssl"
if mysql --help | grep -q -v -- '--skip-ssl'; then SKIP_SSL=""; fi

function basic_setup {
    export CONTAINER_NAME="testserver"
    export HOSTPORT=31000
    export MYTMPDIR="${HOME}/tmp/testserver-sh_${RANDOM}_$$"
    export outdir="${HOME}/tmp/mariadb_testserver/output_${RANDOM}_$$"
    export VOLUME="dbserver_test-${RANDOM}_$$"

    export MOUNTUID=33
    export MOUNTGID=33

    # Homebrew mysql client really really wants /usr/local/etc/my.cnf.d
    if [ "${OS:-$(uname)}" != "Windows_NT" ] && [ ! -d "$(brew --prefix)/etc/my.cnf.d" ]; then
        mkdir -p "$(brew --prefix)/etc/my.cnf.d" || sudo mkdir -p "$(brew --prefix)/etc/my.cnf.d"
    fi
    docker rm -f ${CONTAINER_NAME} 2>/dev/null || true

    # Initialize the volume with the correct ownership
    docker run --rm -v "${VOLUME}:/var/lib/mysql:nocopy" busybox:stable chown -R ${MOUNTUID}:${MOUNTGID} /var/lib/mysql
}

function teardown {
  docker rm -f ${CONTAINER_NAME} >/dev/null
  docker volume rm $VOLUME >/dev/null || true
}

# Wait for container to be ready.
function containercheck {
  for i in {15..0}; do
    # fail if we can't find the container
    if ! docker inspect ${CONTAINER_NAME} >/dev/null; then
      break
    fi

    status="$(docker inspect ${CONTAINER_NAME} | jq -r '.[0].State.Status')"
    if [ "${status}" != "running" ]; then
      break
    fi
    health="$(docker inspect --format '{{json .State.Health }}' ${CONTAINER_NAME} | jq -r .Status)"
    case ${health} in
    healthy)
      return 0
      ;;
    *)
      sleep 1
      ;;
    esac
  done
  echo "# --- ddev-dbserver FAIL -----"
  return 1
}
