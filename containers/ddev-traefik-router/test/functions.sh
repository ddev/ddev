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

# Wait for Traefik to reload config by running the healthcheck.
# This is more reliable than sleep as the healthcheck has built-in
# wait_for_routers() logic that polls the API for up to 10 seconds.
function wait_for_config_reload {
  docker exec ${CONTAINER_NAME} bash -c 'rm -f /tmp/healthy && /healthcheck.sh' >/dev/null
  sleep 1
}

# Wait for a specific router to appear in the Traefik API.
# Args: router_name (without @provider suffix)
# Returns: 0 if router found, 1 if timeout
function wait_for_router {
  local router_name="$1"
  local max_attempts=20
  local attempt=0

  while [ $attempt -lt $max_attempts ]; do
    if docker exec ${CONTAINER_NAME} bash -c "curl -sf http://127.0.0.1:\${TRAEFIK_MONITOR_PORT}/api/http/routers/${router_name}@file" >/dev/null 2>&1; then
      return 0
    fi
    sleep 0.5
    attempt=$((attempt + 1))
  done
  return 1
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
  mkcert -install >/dev/null 2>&1

  # Copy test data using --entrypoint to override default entrypoint
  docker run --rm --entrypoint /bin/bash \
    -v "$(mkcert -CAROOT):/mnt/mkcert" \
    -v "${TEST_SCRIPT_DIR}/testdata:/mnt/testdata" \
    -v ${TEST_VOLUME_NAME}:/mnt/ddev-global-cache \
    "${IMAGE}" \
    -c "mkdir -p /mnt/ddev-global-cache/{mkcert,traefik} && chmod -R ugo+w /mnt/ddev-global-cache/* && cp -R /mnt/mkcert /mnt/ddev-global-cache && cp -rT /mnt/testdata/ /mnt/ddev-global-cache/traefik/" >/dev/null
}
