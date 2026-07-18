#!/usr/bin/env bash

# Container entrypoint for ddev-router.
#
# Starts a minimal fallback responder that always answers with a real HTTP
# 404 status plus an informative body. Traefik's global dynamic config (see
# traefik_global_config_template.yaml) adds a low-priority catch-all router
# that forwards any request not matched by a real project router to this
# local responder, so users get an explanation instead of Traefik's bare
# "404 page not found" - and the response is still a genuine 404, not a 200,
# so browsers/scripts/monitoring correctly see it as "not found".
#
# socat's `fork` mode spawns a fresh handler per connection and keeps
# listening for the life of the container, so there's no separate daemon
# process or pidfile to manage or go stale across a container restart.
#
# Then hands off to the normal traefik startup (wrapped by
# monitor-traefik-stderr.sh for error-log capture).

set -eu -o pipefail

# A container restart reuses the writable layer, so a /tmp/healthy marker
# from before the restart would otherwise survive and trigger
# healthcheck.sh's steady-state "already healthy, sleep before rechecking"
# fast path on the very first check of the new boot - needlessly delaying
# detection of readiness right when it matters most.
rm -f /tmp/healthy

DDEV_ROUTER_FALLBACK_PORT="${DDEV_ROUTER_FALLBACK_PORT:-8999}"

socat -T 5 TCP-LISTEN:"${DDEV_ROUTER_FALLBACK_PORT}",bind=127.0.0.1,reuseaddr,fork \
  SYSTEM:/usr/local/bin/ddev-router-fallback-responder.sh &

exec /usr/local/bin/monitor-traefik-stderr.sh "$@"
