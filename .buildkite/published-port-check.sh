#!/usr/bin/env bash
# Check that a port a container publishes on the host answers from the host,
# the way ddev-router's ports must. Without this, a provider whose port
# forwarding is broken passes sanetestbot.sh and then fails every HTTP test,
# as Apple Container did in macos-socktainer builds 31 and 32 (connection
# reset by peer on every published port).
#
# Usage: published-port-check.sh [host port]
# Binds 127.0.0.1, like DDEV's default router ports. The default port isn't one
# that DDEV or the providers' idle listeners use.

set -o errexit
set -o pipefail
set -o nounset

PORT=${1:-18765}
NAME=ddev-published-port-check
EXPECT=ddev-published-port-check-ok

docker rm -f "${NAME}" >/dev/null 2>&1 || true
trap 'docker rm -f "${NAME}" >/dev/null 2>&1 || true' EXIT

# ddev-utilities is Alpine, whose busybox has nc but no httpd. nc -lk -e runs
# the responder once for each connection.
docker run -d --name "${NAME}" -p "127.0.0.1:${PORT}:8000" ddev/ddev-utilities sh -c \
  "printf '#!/bin/sh\nprintf \"HTTP/1.0 200 OK\\\\r\\\\nContent-Type: text/plain\\\\r\\\\n\\\\r\\\\n${EXPECT}\\\\n\"\n' >/respond && chmod +x /respond && exec nc -lk -p 8000 -e /respond" >/dev/null

# The forward can take a few seconds to appear after the container starts.
body=""
for i in $(seq 1 15); do
  body=$(curl -sS --max-time 5 "http://127.0.0.1:${PORT}/" 2>&1) || true
  if [[ "${body}" == *"${EXPECT}"* ]]; then
    echo "Published port 127.0.0.1:${PORT} answered after ${i} attempt(s)"
    exit 0
  fi
  sleep 1
done

echo "Published port 127.0.0.1:${PORT} did not answer from the host; last response:"
echo "${body:-(empty)}"
echo "Every DDEV test that uses the router's host ports would fail the same way."
echo "--- curl -v from the host"
curl -v --max-time 5 "http://127.0.0.1:${PORT}/" 2>&1 || true
echo "--- the same request from inside the container"
docker exec "${NAME}" sh -c "printf 'GET / HTTP/1.0\r\n\r\n' | nc -w 5 127.0.0.1 8000" 2>&1 || true
echo "--- docker ps for ${NAME}"
docker ps -a --filter "name=${NAME}" --format '{{.Names}} {{.Status}} {{.Ports}}' 2>&1 || true
docker logs "${NAME}" 2>&1 | tail -20 || true
if command -v lsof >/dev/null 2>&1; then
  echo "--- listeners on ${PORT}"
  lsof -nP -iTCP:"${PORT}" -sTCP:LISTEN 2>&1 || true
  # Since #9114 other providers stay running during a job; list their listeners.
  echo "--- all TCP listeners"
  lsof -nP -iTCP -sTCP:LISTEN 2>&1 | awk 'NR == 1 || $1 !~ /^(rapportd|ControlCe|sharingd)$/' || true
fi
if [ "${DOCKER_TYPE:-}" = "socktainer" ]; then
  echo "--- container ls"
  container ls 2>&1 || true
  echo "--- apple container log, last 10 minutes"
  log show --last 10m --style compact --predicate 'subsystem BEGINSWITH "com.apple.container"' 2>&1 | tail -60 || true
  if [ -f ~/tmp/socktainer.log ]; then
    echo "--- ~/tmp/socktainer.log (tail)"
    tail -40 ~/tmp/socktainer.log
  fi
fi
exit 1
