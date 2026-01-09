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

  # Remove container first, then volume (container must be stopped before volume can be removed)
  basic_setup
  docker volume rm ${TEST_VOLUME_NAME} 2>/dev/null || true

  setup_test_data

  #echo "# Starting ${IMAGE}" >&3
  docker run --rm --name ${CONTAINER_NAME} -p ${HOSTPORT_HTTP}:80 -p ${HOSTPORT_HTTPS}:443 -v ${TEST_VOLUME_NAME}:/mnt/ddev-global-cache -d ${IMAGE} --configFile=/mnt/ddev-global-cache/traefik/.static_config.yaml >/dev/null
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

@test "verify healthcheck captures router count mismatch warning" {
  # Use unique filename for this test
  local config_file="/mnt/ddev-global-cache/traefik/config/test8_invalid.yaml"

  # Clean up any leftover state
  docker exec ${CONTAINER_NAME} rm -f /tmp/ddev-traefik-errors.txt "${config_file}"
  wait_for_config_reload

  # Add an invalid config file with duplicate router name
  docker exec ${CONTAINER_NAME} bash -c "cat > ${config_file} << 'EOF'
http:
  routers:
    d11-web-80-http:
      entrypoints:
        - http-80
      rule: HostRegexp(\`^invalid\.ddev\.site$\`)
      service: nonexistent-service
      tls: false
EOF"
  # Wait for traefik to reload config
  wait_for_config_reload

  # Force healthcheck to run
  run docker exec ${CONTAINER_NAME} bash -c 'rm -f /tmp/healthy && /healthcheck.sh'
  assert_success
  assert_output --partial "WARNING: Router count mismatch"

  # Verify warning was written to error file
  run docker exec ${CONTAINER_NAME} cat /tmp/ddev-traefik-errors.txt
  assert_success
  assert_output --partial "WARNING: Router count mismatch"

  # Clean up
  docker exec ${CONTAINER_NAME} rm -f "${config_file}"
}

@test "verify healthcheck clears warnings when issue resolved" {
  # Create an issue, capture warning, then resolve and verify warning is cleared
  # All within a single fresh container to avoid volume state issues

  # Clean up any leftover state
  docker exec ${CONTAINER_NAME} rm -f /tmp/ddev-traefik-errors.txt
  docker exec ${CONTAINER_NAME} bash -c 'rm -f /mnt/ddev-global-cache/traefik/config/test*.yaml'
  wait_for_config_reload

  # First verify we start with no warnings (router count should match)
  run docker exec ${CONTAINER_NAME} bash -c 'rm -f /tmp/healthy && /healthcheck.sh'
  assert_success
  refute_output --partial "WARNING:"

  # Add an invalid config file with duplicate router name
  docker exec ${CONTAINER_NAME} bash -c "cat > /mnt/ddev-global-cache/traefik/config/test_invalid.yaml << 'EOF'
http:
  routers:
    d11-web-80-http:
      entrypoints:
        - http-80
      rule: HostRegexp(\`^invalid\.ddev\.site$\`)
      service: nonexistent-service
      tls: false
EOF"
  wait_for_config_reload

  # Force healthcheck to capture the warning
  run docker exec ${CONTAINER_NAME} bash -c 'rm -f /tmp/healthy && /healthcheck.sh'
  assert_success
  assert_output --partial "WARNING: Router count mismatch"

  # Verify warning was written to error file
  run docker exec ${CONTAINER_NAME} cat /tmp/ddev-traefik-errors.txt
  assert_success
  assert_output --partial "WARNING: Router count mismatch"

  # Remove the invalid config to resolve the issue
  docker exec ${CONTAINER_NAME} rm -f /mnt/ddev-global-cache/traefik/config/test_invalid.yaml
  wait_for_config_reload

  # Force healthcheck to run again - should clear warnings
  run docker exec ${CONTAINER_NAME} bash -c 'rm -f /tmp/healthy && /healthcheck.sh'
  assert_success
  refute_output --partial "WARNING:"

  # Verify warning file is cleared or doesn't exist
  run docker exec ${CONTAINER_NAME} bash -c 'cat /tmp/ddev-traefik-errors.txt 2>/dev/null || echo "FILE_DOES_NOT_EXIST"'
  assert_success
  refute_output --partial "WARNING:"
}

@test "verify API detects router with missing service reference" {
  # Clean up any leftover state
  docker exec ${CONTAINER_NAME} rm -f /tmp/ddev-traefik-errors.txt
  docker exec ${CONTAINER_NAME} bash -c 'rm -f /mnt/ddev-global-cache/traefik/config/test*.yaml'
  wait_for_config_reload

  # Add a config with a router referencing a non-existent service
  docker exec ${CONTAINER_NAME} bash -c "cat > /mnt/ddev-global-cache/traefik/config/test_missing_service.yaml << 'EOF'
http:
  routers:
    test-missing-svc:
      entrypoints:
        - http-80
      rule: Host(\`test-missing-svc.ddev.site\`)
      service: this-service-does-not-exist
EOF"
  # Wait for traefik to reload config
  wait_for_config_reload
  wait_for_router "test-missing-svc"

  # Verify router is disabled with error
  run docker exec ${CONTAINER_NAME} bash -c 'curl -sf http://127.0.0.1:${TRAEFIK_MONITOR_PORT}/api/http/routers/test-missing-svc@file | jq -r .status'
  assert_success
  assert_output "disabled"

  # Verify error message mentions the missing service
  run docker exec ${CONTAINER_NAME} bash -c 'curl -sf http://127.0.0.1:${TRAEFIK_MONITOR_PORT}/api/http/routers/test-missing-svc@file | jq -r ".error[]"'
  assert_success
  assert_output --partial "this-service-does-not-exist"
  assert_output --partial "does not exist"

  # Verify overview shows error count > 0
  run docker exec ${CONTAINER_NAME} bash -c 'curl -sf http://127.0.0.1:${TRAEFIK_MONITOR_PORT}/api/overview | jq ".http.routers.errors > 0"'
  assert_success
  assert_output "true"

  # Clean up
  docker exec ${CONTAINER_NAME} rm -f /mnt/ddev-global-cache/traefik/config/test_missing_service.yaml
}

@test "verify API detects router with missing middleware reference" {
  # Clean up any leftover state
  docker exec ${CONTAINER_NAME} rm -f /tmp/ddev-traefik-errors.txt
  docker exec ${CONTAINER_NAME} bash -c 'rm -f /mnt/ddev-global-cache/traefik/config/test*.yaml'
  wait_for_config_reload

  # Add a config with a router referencing a non-existent middleware
  docker exec ${CONTAINER_NAME} bash -c "cat > /mnt/ddev-global-cache/traefik/config/test_missing_middleware.yaml << 'EOF'
http:
  routers:
    test-missing-mw:
      entrypoints:
        - http-80
      rule: Host(\`test-missing-mw.ddev.site\`)
      service: d11-web-80
      middlewares:
        - nonexistent-middleware
EOF"
  wait_for_config_reload
  wait_for_router "test-missing-mw"

  # Verify router is disabled with error
  run docker exec ${CONTAINER_NAME} bash -c 'curl -sf http://127.0.0.1:${TRAEFIK_MONITOR_PORT}/api/http/routers/test-missing-mw@file | jq -r .status'
  assert_success
  assert_output "disabled"

  # Verify error message mentions the missing middleware
  run docker exec ${CONTAINER_NAME} bash -c 'curl -sf http://127.0.0.1:${TRAEFIK_MONITOR_PORT}/api/http/routers/test-missing-mw@file | jq -r ".error[]"'
  assert_success
  assert_output --partial "nonexistent-middleware"
  assert_output --partial "does not exist"

  # Clean up
  docker exec ${CONTAINER_NAME} rm -f /mnt/ddev-global-cache/traefik/config/test_missing_middleware.yaml
}

@test "verify API detects router with invalid entrypoint" {
  # Clean up any leftover state
  docker exec ${CONTAINER_NAME} rm -f /tmp/ddev-traefik-errors.txt
  docker exec ${CONTAINER_NAME} bash -c 'rm -f /mnt/ddev-global-cache/traefik/config/test*.yaml'
  wait_for_config_reload

  # Add a config with a router using a non-existent entrypoint
  docker exec ${CONTAINER_NAME} bash -c "cat > /mnt/ddev-global-cache/traefik/config/test_bad_entrypoint.yaml << 'EOF'
http:
  routers:
    test-bad-ep:
      entrypoints:
        - nonexistent-entrypoint
      rule: Host(\`test-bad-ep.ddev.site\`)
      service: d11-web-80
EOF"
  # Wait for traefik to reload config
  wait_for_config_reload
  wait_for_router "test-bad-ep"

  # Verify router is disabled with error
  run docker exec ${CONTAINER_NAME} bash -c 'curl -sf http://127.0.0.1:${TRAEFIK_MONITOR_PORT}/api/http/routers/test-bad-ep@file | jq -r .status'
  assert_success
  assert_output "disabled"

  # Verify error message mentions the entrypoint issue
  run docker exec ${CONTAINER_NAME} bash -c 'curl -sf http://127.0.0.1:${TRAEFIK_MONITOR_PORT}/api/http/routers/test-bad-ep@file | jq -r ".error[]" | head -1'
  assert_success
  assert_output --partial "nonexistent-entrypoint"

  # Clean up
  docker exec ${CONTAINER_NAME} rm -f /mnt/ddev-global-cache/traefik/config/test_bad_entrypoint.yaml
}

@test "verify API detects router with invalid rule syntax" {
  # Clean up any leftover state
  docker exec ${CONTAINER_NAME} rm -f /tmp/ddev-traefik-errors.txt
  docker exec ${CONTAINER_NAME} bash -c 'rm -f /mnt/ddev-global-cache/traefik/config/test*.yaml'
  wait_for_config_reload

  # Add a config with a router having invalid rule syntax
  docker exec ${CONTAINER_NAME} bash -c "cat > /mnt/ddev-global-cache/traefik/config/test_bad_rule.yaml << 'EOF'
http:
  routers:
    test-bad-rule:
      entrypoints:
        - http-80
      rule: InvalidSyntax(((broken
      service: d11-web-80
EOF"
  wait_for_config_reload
  wait_for_router "test-bad-rule"

  # Verify router is disabled with error
  run docker exec ${CONTAINER_NAME} bash -c 'curl -sf http://127.0.0.1:${TRAEFIK_MONITOR_PORT}/api/http/routers/test-bad-rule@file | jq -r .status'
  assert_success
  assert_output "disabled"

  # Verify error message mentions parsing error with the specific rule
  run docker exec ${CONTAINER_NAME} bash -c 'curl -sf http://127.0.0.1:${TRAEFIK_MONITOR_PORT}/api/http/routers/test-bad-rule@file | jq -r ".error[]"'
  assert_success
  assert_output --partial "InvalidSyntax"
  assert_output --partial "parsing rule"

  # Clean up
  docker exec ${CONTAINER_NAME} rm -f /mnt/ddev-global-cache/traefik/config/test_bad_rule.yaml
}

