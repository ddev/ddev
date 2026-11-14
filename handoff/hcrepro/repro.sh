#!/bin/bash
# Healthcheck probe user repro: container runs as uid 1000; its healthcheck records its uid
# and touches /tmp/healthy. Then the container's own user tries to remove that file.
docker rm -f hcuser >/dev/null 2>&1
docker run -d --name hcuser --user 1000:1000 \
  --health-cmd 'id -u > /tmp/probe-uid; touch /tmp/healthy' --health-interval 2s \
  alpine sleep 600 >/dev/null || { echo "run failed"; exit 1; }
sleep 10
echo "-- Config.User: $(docker inspect hcuser --format '{{.Config.User}}')  health: $(docker inspect hcuser --format '{{.State.Health.Status}}')"
echo "-- docker exec id -u: $(docker exec hcuser id -u)"
echo "-- uid the probe ran as: $(docker exec hcuser cat /tmp/probe-uid)"
echo "-- ls -ln /tmp/healthy:"; docker exec hcuser ls -ln /tmp/healthy
echo "-- rm -f /tmp/healthy as the container's user:"; docker exec hcuser rm -f /tmp/healthy && echo OK || echo "FAILED rc=$?"
docker rm -f hcuser >/dev/null
