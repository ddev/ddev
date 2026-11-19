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
# the responder once for each connection. The responder reads the request
# before answering: closing a socket with unread data sends a reset, which
# made every request fail even when the forward worked.
read -r -d '' RESPOND <<'EOF' || true
#!/bin/sh
cr=$(printf '\r')
while IFS= read -r line; do
  line=${line%"$cr"}
  [ -z "$line" ] && break
done
printf 'HTTP/1.0 200 OK\r\nContent-Type: text/plain\r\n\r\n%s\n' "$EXPECT"
EOF
docker run -d --name "${NAME}" -p "127.0.0.1:${PORT}:8000" -e EXPECT="${EXPECT}" -e RESPOND="${RESPOND}" \
  ddev/ddev-utilities sh -c 'printf "%s\n" "$RESPOND" >/respond && chmod +x /respond && exec nc -lk -p 8000 -e /respond' >/dev/null

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
  # macOS Local Network privacy asks once for each new container-runtime-linux
  # binary, so each install or upgrade of Apple Container needs an Allow on the
  # testbot's desktop. Until someone answers the prompt, the forwarder can't
  # connect to containers and logs "No route to host" (EHOSTUNREACH), and
  # UserEventAgent logs a Local Network lookup for each attempt. Neither this job
  # nor SSH can show or answer the prompt. The log process is excluded because
  # it logs its own arguments, which contain the search text.
  localnet=$(log show --last 5m --style compact \
    --predicate 'process != "log" AND eventMessage CONTAINS "LocalNetwork: found bundle id com.apple.container.container-runtime-linux"' 2>/dev/null |
    grep -c 'LocalNetwork: found bundle id' || true)
  if [ "${localnet:-0}" -gt 0 ]; then
    echo "+++ Local Network permission needed on $(hostname)"
    echo "macOS checked container-runtime-linux's Local Network permission ${localnet} time(s) in the last 5 minutes,"
    echo "so a Local Network prompt is probably waiting on this testbot's desktop. On the desktop (screen sharing,"
    echo "not SSH), allow container-runtime-linux, or switch it on in System Settings > Privacy & Security >"
    echo "Local Network, then rerun the build."
  fi
  echo "--- container ls"
  container ls 2>&1 || true
  # Apple's port forwarder connects from the host to the container's address.
  # Try the same connection from this job's shell: if it works while the
  # forwarder fails, the problem is the forwarder's own permission (above).
  ip=$(container ls 2>/dev/null | awk -v n="${NAME}" '$1 == n {print $6}' | cut -d/ -f1)
  if [ -n "${ip}" ]; then
    echo "--- route to the container address ${ip}"
    route -n get "${ip}" 2>&1 || true
    echo "--- ping ${ip} from the job's shell"
    ping -c2 -t2 "${ip}" 2>&1 || true
    echo "--- curl http://${ip}:8000/ from the job's shell, bypassing the port forwarder"
    curl -sv --max-time 5 "http://${ip}:8000/" 2>&1 || true
  fi
  # A job's container system start registers container-apiserver and its
  # helpers in gui/<uid>; one started over SSH lands in user/<uid> and a later
  # job reuses it. Show which.
  for domain in "gui/$(id -u)" "user/$(id -u)"; do
    echo "--- container services in ${domain}"
    launchctl print "${domain}" 2>/dev/null | grep -E '^[[:space:]]+[0-9-]+[[:space:]]+[-0-9]+[[:space:]]+com\.apple\.container' || echo "(none)"
    launchctl print "${domain}/com.apple.container.apiserver" 2>&1 | grep -E 'state =|domain =|session =|spawn type' || true
  done
  echo "--- the job's launchd session: $(launchctl managername 2>&1)"
  echo "--- apple container log, last 10 minutes"
  log show --last 10m --style compact --predicate 'subsystem BEGINSWITH "com.apple.container"' 2>&1 | tail -60 || true
  if [ -f ~/tmp/socktainer.log ]; then
    echo "--- ~/tmp/socktainer.log (tail)"
    tail -40 ~/tmp/socktainer.log
  fi
fi
exit 1
