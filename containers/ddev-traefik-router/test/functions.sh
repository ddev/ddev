#!/usr/bin/env bash

function basic_setup {
    export CONTAINER_NAME="ddev-traefik-router-test"
    export HOSTPORT_HTTP=31080
    export HOSTPORT_HTTPS=31443
    export TEST_VOLUME_NAME="ddev-traefik-router-test-cache"

    docker rm -f ${CONTAINER_NAME} 2>/dev/null || true
}

function teardown {
  docker rm -f ${CONTAINER_NAME} >/dev/null 2>&1 || true
  docker volume rm ${TEST_VOLUME_NAME} 2>/dev/null || true
}

# Wait for container to be ready.
function containercheck {
  for i in {60..0}; do
    # fail if we can't find the container
    if ! docker inspect ${CONTAINER_NAME} >/dev/null 2>&1; then
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
  echo "# --- ddev-traefik-router FAIL -----"
  return 1
}

# Setup test data in the test volume (separate from ddev-global-cache to avoid conflicts)
# Must use --entrypoint to override the traefik entrypoint
function setup_test_data {
  # Make sure rootCA is created and installed on the ddev-global-cache/mkcert
  mkcert -install

  # Copy test data using --entrypoint to override default entrypoint
  docker run --rm --entrypoint /bin/bash \
    -v "$(mkcert -CAROOT):/mnt/mkcert" \
    -v "${TEST_SCRIPT_DIR}/testdata:/mnt/testdata" \
    -v ${TEST_VOLUME_NAME}:/mnt/ddev-global-cache \
    "${IMAGE}" \
    -c "mkdir -p /mnt/ddev-global-cache/{mkcert,traefik} && chmod -R ugo+w /mnt/ddev-global-cache/* && cp -R /mnt/mkcert /mnt/ddev-global-cache && cp -rT /mnt/testdata/ /mnt/ddev-global-cache/traefik/"
}