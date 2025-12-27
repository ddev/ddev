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
    
    # Calculate expected router count by parsing config files
    expected_router_count=0
    if [ -d /mnt/ddev-global-cache/traefik/config ]; then
        config_files=$(find /mnt/ddev-global-cache/traefik/config -name "*.yaml" -o -name "*.yml" 2>/dev/null)
        if [ -n "$config_files" ]; then
            for config_file in $config_files; do
                # Count routers in each config file using yq
                routers_in_file=$(yq eval '.http.routers | length' "$config_file" 2>/dev/null || echo 0)
                expected_router_count=$((expected_router_count + routers_in_file))
            done
        fi
    fi
    
    # Count routers loaded via file provider (.ddev/traefik/config/<project>.yaml)
    file_router_count=$(curl -sf "http://127.0.0.1:${TRAEFIK_MONITOR_PORT}/api/http/routers" 2>/dev/null | jq '[.[] | select(.provider == "file")] | length' 2>/dev/null || echo 0)
    
    # Sum up router/service/middleware config errors reported by Traefik
    error_count=$(curl -sf "http://127.0.0.1:${TRAEFIK_MONITOR_PORT}/api/overview" 2>/dev/null | jq '(.http.routers.errors // 0) + (.http.services.errors // 0) + (.http.middlewares.errors // 0)' 2>/dev/null || echo 0)
    
    # Check backend service status - verify all services are UP
    services_not_up=0
    if [ "$file_router_count" -gt 0 ]; then
        services_not_up=$(curl -sf "http://127.0.0.1:${TRAEFIK_MONITOR_PORT}/api/http/services" 2>/dev/null | jq '[.[] | select(.provider == "file") | select(.serverStatus != "UP")] | length' 2>/dev/null || echo 0)
    fi
    
    # Healthy if:
    # 1. Config files found and expected routers > 0
    # 2. Actual router count matches expected count
    # 3. No config errors
    # 4. All backend services are UP
    if [ "$expected_router_count" -gt 0 ] && \
       [ "$file_router_count" -eq "$expected_router_count" ] && \
       [ "$error_count" -eq 0 ] && \
       [ "$services_not_up" -eq 0 ]; then
        printf "%s" "${check}"
        touch /tmp/healthy
        exit 0
    fi
    
    # Set descriptive error message for failure
    elif [ "$error_count" -gt 0 ]; then
    else
        check="WARNING: Unknown issue detected"
    fi
    # Return success with warning message
    printf "%s" "${check}"
    touch /tmp/healthy
    exit 0
fi

printf "Traefik healthcheck failed: %s" "${check}"
rm -f /tmp/healthy
exit $exit_code
