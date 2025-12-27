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
    # If there is no dynamic config, don't check for additional endpoints
    if [ ! -d /mnt/ddev-global-cache/traefik/config ]; then
        printf "%s" "${check}"
        touch /tmp/healthy
        exit 0
    fi

    # Get expected project count from environment (set by DDEV when starting router)
    expected_project_count=${EXPECTED_PROJECT_COUNT:-0}

    # If no projects expected, we're healthy as soon as traefik ping works
    if [ "$expected_project_count" -eq 0 ]; then
        printf "%s" "${check}"
        touch /tmp/healthy
        exit 0
    fi

    # Count unique projects that have loaded routers via file provider
    # Router names follow pattern: <projectname>-<service>-<port>-<http|https>@file
    # where service is a simple word (web, db, etc) and port is a number
    # We use a regex to extract the project name, handling dashed project names correctly
    routers_json=$(curl -sf "http://127.0.0.1:${TRAEFIK_MONITOR_PORT}/api/http/routers" 2>/dev/null || echo "[]")
    # Extract unique project names using regex pattern matching
    # Pattern: everything before -<word>-<number>-<http|https>@file
    loaded_project_count=$(echo "$routers_json" | jq -r '
      [.[] | select(.provider == "file") | .name |
       capture("^(?<project>.+)-[a-zA-Z]+-[0-9]+-(http|https)(@file)?$") |
       .project] | unique | length' 2>/dev/null || echo 0)

    # Sum up router/service/middleware config errors reported by Traefik
    error_count=$(curl -sf "http://127.0.0.1:${TRAEFIK_MONITOR_PORT}/api/overview" 2>/dev/null | jq '(.http.routers.errors // 0) + (.http.services.errors // 0) + (.http.middlewares.errors // 0)' 2>/dev/null || echo 0)

    # Healthy if all expected projects have loaded routers and no config errors exist
    if [ "$loaded_project_count" -ge "$expected_project_count" ] && [ "$error_count" -eq 0 ]; then
        printf "%s" "${check}"
        touch /tmp/healthy
        exit 0
    fi

    # Set descriptive error message for failure
    if [ "$loaded_project_count" -lt "$expected_project_count" ]; then
        check="Waiting for projects: loaded ${loaded_project_count}/${expected_project_count}"
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
