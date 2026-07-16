#!/usr/bin/env bash

## traefik health check
set -u -o pipefail

# Configuration
sleeptime=59
error_file="/tmp/ddev-traefik-errors.txt"
healthy_marker="/tmp/healthy"
config_dir="/mnt/ddev-global-cache/traefik/config"

# --- Helper Functions ---

# Clear healthcheck warnings from error file (preserves traefik ERR/WRN logs)
clear_warnings() {
    if [ -f "${error_file}" ]; then
        sed -i '/^WARNING:/d' "${error_file}"
        # Remove file if now empty
        [ ! -s "${error_file}" ] && rm -f "${error_file}"
    fi
}

# Get count of routers from Traefik API (all are from file provider)
get_traefiks_router_count() {
    curl -sf "http://127.0.0.1:${TRAEFIK_MONITOR_PORT}/api/http/routers?per_page=10000" 2>/dev/null \
        | jq '[.[] | select(.provider == "file")] | length' 2>/dev/null || echo 0
}

# Get total config error count from Traefik API (routers + services + middlewares)
get_error_count() {
    curl -sf "http://127.0.0.1:${TRAEFIK_MONITOR_PORT}/api/overview" 2>/dev/null \
        | jq '(.http.routers.errors // 0) + (.http.services.errors // 0) + (.http.middlewares.errors // 0)' 2>/dev/null || echo 0
}

# Calculate expected router count by parsing config files
# that we've given to traefik
get_expected_router_count() {
    local count=0
    if [ -d "${config_dir}" ]; then
        local config_files
        config_files=$(find "${config_dir}" -name "*.yaml" -o -name "*.yml" 2>/dev/null)
        if [ -n "$config_files" ]; then
            for config_file in $config_files; do
                local routers_in_file
                routers_in_file=$(yq eval '.http.routers | length' "$config_file" 2>/dev/null || echo 0)
                count=$((count + routers_in_file))
            done
        fi
    fi
    echo "$count"
}

# Mark container as healthy and exit successfully
mark_healthy() {
    local message="$1"
    printf "%s" "${message}"
    touch "${healthy_marker}"
    exit 0
}

# Write warning to error file (avoids duplicates)
write_warning() {
    local warning="$1"
    if ! grep -qF "${warning}" "${error_file}" 2>/dev/null; then
        echo "${warning}" >> "${error_file}"
    fi
}

# Wait for traefik to reflect file-provider routers (startup and after config pushes)
# Only waits when we expect routers, there are no errors yet, and counts don't match
wait_for_routers() {
    local file_router_count="$1"
    local expected_router_count="$2"
    local error_count="$3"

    # How long to wait (seconds) for Traefik to reflect new file-provider config.
    # This covers both initial startup and later config pushes.
    local max_retries=60

    # Only retry if: we expect some routers AND there are no errors yet AND counts don't match yet.
    if [ "$expected_router_count" -gt 0 ] && [ "$error_count" -eq 0 ] && [ "$file_router_count" -ne "$expected_router_count" ]; then
        for _ in $(seq 1 $max_retries); do
            sleep 1
            file_router_count=$(get_traefiks_router_count)
            error_count=$(get_error_count)

            # Success: counts match and still no errors.
            if [ "$file_router_count" -eq "$expected_router_count" ] && [ "$error_count" -eq 0 ]; then
                break
            fi

            # Stop early if errors appeared (surface them immediately).
            if [ "$error_count" -gt 0 ]; then
                break
            fi
        done
    fi

    # Return values via global variables (bash limitation)
    ROUTER_COUNT=$file_router_count
    ERROR_COUNT=$error_count
}

# Generate appropriate warning message based on state
# Pass to it
# $1=expected_router_count
# $2=actual_router_count
# #3=error count
generate_warning_message() {
    local expected_router_count="$1"
    local actual_router_count="$2"
    local errors="$3"

    if [ "$errors" -gt 0 ]; then
        echo "WARNING: Detected ${errors} configuration error(s)"
    elif [ "${expected_router_count}" -gt 0 ] && [ "${actual_router_count}" -ne "${expected_router_count}" ]; then
        echo "WARNING: Router count mismatch: ${actual_router_count} loaded, ${expected_router_count} expected"
    else
        echo "WARNING: Unknown issue detected"
    fi
}

# --- Main Script ---

# Since docker doesn't provide a lazy period for startup,
# we track health. If the last check showed healthy
# as determined by existence of /tmp/healthy, then
# sleep at startup. This requires the timeout to be set
# higher than the sleeptime used here.
if [ -f "${healthy_marker}" ]; then
    printf "container was previously healthy, so sleeping %s seconds before continuing healthcheck... " ${sleeptime}
    sleep ${sleeptime}
fi

# Check traefik ping endpoint
# Technique from https://doc.traefik.io/traefik/operations/ping/#entrypoint
check=$(traefik healthcheck --ping 2>&1)
exit_code=$?

# If ping fails, traefik is unhealthy
if [ $exit_code -ne 0 ]; then
    printf "Traefik healthcheck failed: %s" "${check}"
    rm -f "${healthy_marker}"
    exit $exit_code
fi

# Ping succeeded - now inspect additional health indicators
# Traefik API endpoints https://doc.traefik.io/traefik/operations/api/#endpoints

# If no dynamic config directory, we're done
if [ ! -d "${config_dir}" ]; then
    clear_warnings
    mark_healthy "${check}"
fi

# Get expected and actual router counts
expected_router_count=$(get_expected_router_count)
file_router_count=$(get_traefiks_router_count)
error_count=$(get_error_count)

# Wait for traefik to finish loading if needed (avoids false warnings during startup)
wait_for_routers "$file_router_count" "$expected_router_count" "$error_count"
file_router_count=$ROUTER_COUNT
error_count=$ERROR_COUNT

# Check if configuration is healthy:
# 1. No config errors
# 2. Either:
#    a. No config files expected (all projects stopped) OR
#    b. Expected routers > 0 AND actual router count matches expected count
if [ "$error_count" -eq 0 ] && \
   { [ "$expected_router_count" -eq 0 ] || \
     { [ "$expected_router_count" -gt 0 ] && [ "$file_router_count" -eq "$expected_router_count" ]; }; }; then
    clear_warnings
    mark_healthy "${check}"
fi

# Configuration has issues - generate and record warning
# Return success (exit 0) so router is "healthy" but user sees warnings
# via GetRouterConfigErrors() in router.go
warning=$(generate_warning_message "$expected_router_count" "$file_router_count" "$error_count")
write_warning "${warning}"
mark_healthy "${warning}"
