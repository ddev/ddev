#!/usr/bin/env bash

## traefik health check
set -u -o pipefail
sleeptime=59

# Since docker doesn't provide a lazy period for startup,
# we track health. If the last check showed healthy
# as determined by existence of /tmp/healthy, then
# sleep at startup. This requires the timeout to be set
# higher than the sleeptime used here.
if [ -f /tmp/healthy ]; then
    printf "container was previously healthy, so sleeping %s seconds before continuing healthcheck...  " ${sleeptime}
    sleep ${sleeptime}
fi

# If we can now access the traefik ping endpoint, then we're healthy
# Technique from https://doc.traefik.io/traefik/operations/ping/#entrypoint
check=$(traefik healthcheck --ping 2>&1)
exit_code=$?

# If ping is successful, inspect additional health indicators
# Traefik API endpoints https://doc.traefik.io/traefik/operations/api/#endpoints
if [ $exit_code -eq 0 ]; then
    # Count routers loaded via file provider (.ddev/traefik/config/<project>.yaml)
    file_router_count=$(curl -sf "http://127.0.0.1:${TRAEFIK_MONITOR_PORT}/api/http/routers" 2>/dev/null | jq '[.[] | select(.provider == "file")] | length' 2>/dev/null || echo 0)
    # Sum up router/service/middleware config errors reported by Traefik
    error_count=$(curl -sf "http://127.0.0.1:${TRAEFIK_MONITOR_PORT}/api/overview" 2>/dev/null | jq '(.http.routers.errors // 0) + (.http.services.errors // 0) + (.http.middlewares.errors // 0)' 2>/dev/null || echo 0)
    # Healthy if file-based routers are present and no config errors exist
    if [ "$file_router_count" -gt 0 ] && [ "$error_count" -eq 0 ]; then
        printf "%s" "${check}"
        touch /tmp/healthy
        exit 0
    fi
    # Set descriptive error message for failure
    if [ "$file_router_count" -eq 0 ]; then
        check="No file-based routers found"
        exit_code=1
    elif [ "$error_count" -gt 0 ]; then
        check="Detected ${error_count} configuration error(s) in project"
        exit_code=2
    else
        check="Unknown failure"
        exit_code=255
    fi
fi

printf "Traefik healthcheck failed: %s" "${check}"
rm -f /tmp/healthy
exit $exit_code
