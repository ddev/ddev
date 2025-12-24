#!/usr/bin/env bats

# Run these tests from the repo root directory, in containers/ddev-traefik-router

setup() {

  bats_require_minimum_version 1.11.0
  set -eu -o pipefail
  brew_prefix=$(brew --prefix)
  load "${brew_prefix}/lib/bats-support/load.bash"
  load "${brew_prefix}/lib/bats-assert/load.bash"
  load "${brew_prefix}/lib/bats-file/load.bash"
  load functions.sh

  if [ "${OS:-$(uname)}" = "Windows_NT" ]; then
    skip "Skipping on Windows"
  fi

  basic_setup
  setup_test_data

  echo "# Starting ${IMAGE}" >&3
  docker run --rm --name ${CONTAINER_NAME} -p ${HOSTPORT_HTTP}:80 -p ${HOSTPORT_HTTPS}:443 -v ddev-global-cache:/mnt/ddev-global-cache -d ${IMAGE} --configFile=/mnt/ddev-global-cache/traefik/.static_config.yaml
  containercheck
}

@test "verify container is healthy" {
  run bash -c "docker inspect --format '{{json .State.Health }}' ${CONTAINER_NAME} | jq -r .Status"
  assert_success
  assert_output "healthy"
}

@test "verify traefik healthcheck ping works" {
  run docker exec ${CONTAINER_NAME} traefik healthcheck --ping
  assert_success
}

@test "verify /api/http/routers endpoint returns routers" {
  run docker exec ${CONTAINER_NAME} bash -c 'curl -sf http://127.0.0.1:${TRAEFIK_MONITOR_PORT}/api/http/routers | jq -e "length > 0"'
  assert_success
}

@test "verify all routers have a provider field" {
  run docker exec ${CONTAINER_NAME} bash -c 'curl -sf http://127.0.0.1:${TRAEFIK_MONITOR_PORT}/api/http/routers | jq -e "all(has(\"provider\"))"'
  assert_success
}

@test "verify at least one router has file provider" {
  run docker exec ${CONTAINER_NAME} bash -c 'curl -sf http://127.0.0.1:${TRAEFIK_MONITOR_PORT}/api/http/routers | jq -e "[.[] | select(.provider == \"file\")] | length > 0"'
  assert_success
}

@test "verify /api/overview endpoint has required structure" {
  run docker exec ${CONTAINER_NAME} bash -c 'curl -sf http://127.0.0.1:${TRAEFIK_MONITOR_PORT}/api/overview | jq -e "has(\"http\") and (.http | has(\"routers\") and has(\"services\") and has(\"middlewares\"))"'
  assert_success
}

@test "verify error fields exist in overview" {
  run docker exec ${CONTAINER_NAME} bash -c 'curl -sf http://127.0.0.1:${TRAEFIK_MONITOR_PORT}/api/overview | jq -e "(.http.routers | has(\"errors\")) and (.http.services | has(\"errors\")) and (.http.middlewares | has(\"errors\"))"'
  assert_success
}
