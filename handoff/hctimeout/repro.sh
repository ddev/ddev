#!/bin/bash
# Healthcheck timeout repro: each probe logs its start, sleeps 30s, then logs its finish.
# With --health-timeout 2s the probe should be killed after 2s and counted as a failure.
docker rm -f hctimeout >/dev/null 2>&1
docker run -d --name hctimeout \
  --health-cmd 'echo start >> /tmp/probe.log; sleep 30; echo finished >> /tmp/probe.log' \
  --health-interval 2s --health-timeout 2s --health-retries 1 \
  alpine sleep 600 >/dev/null || { echo "run failed"; exit 1; }
for t in 12 40; do
  sleep $((t - ${prev:-0})); prev=$t
  echo "== t=${t}s"
  docker inspect hctimeout --format 'status={{.State.Health.Status}} failingStreak={{.State.Health.FailingStreak}} logEntries={{len .State.Health.Log}}'
  docker inspect hctimeout --format '{{range .State.Health.Log}}  exit={{.ExitCode}} start={{.Start}} end={{.End}}{{"\n"}}{{end}}'
  echo "-- /tmp/probe.log: $(docker exec hctimeout cat /tmp/probe.log 2>/dev/null | sort | uniq -c | xargs)"
done
docker rm -f hctimeout >/dev/null
