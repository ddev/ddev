#!/bin/bash
# Exec behavior suite against whatever socktainer daemon is running.
# Usage: handoff/exec-suite.sh <label>
label=$1
export DOCKER_HOST=unix://$HOME/.socktainer/container.sock
cd "$(dirname "$0")/.." || exit 1
repro="go run ./handoff/testdata/execrepro"
docker rm -f exectest >/dev/null 2>&1
docker run -d --name exectest alpine sleep 1800 >/dev/null
sleep 2
echo "===== $label"
echo "-- go small, no half-close";       timeout 30 $repro exectest 'echo hi; exit 42' || echo "TIMEOUT/rc=$?"
echo "-- go 15MB stdout, no half-close"; timeout 90 $repro exectest 'yes 0123456789abcdef | head -c 15000000' || echo "TIMEOUT/rc=$?"
echo "-- go 15MB+5MB stderr";            timeout 90 $repro exectest 'yes 0123456789abcdef | head -c 15000000; yes e | head -c 5000000 >&2; exit 7' || echo "TIMEOUT/rc=$?"
echo "-- go 15MB, half-close";           CLOSEWRITE=1 timeout 90 $repro exectest 'yes 0123456789abcdef | head -c 15000000' || echo "TIMEOUT/rc=$?"
echo "-- cli 15MB byte count";           timeout 90 docker exec exectest sh -c 'yes 0123456789abcdef | head -c 15000000' | wc -c
echo "-- cli exit codes";                docker exec exectest sh -c 'exit 3'; echo "rc=$?"; docker exec exectest sh -c 'echo out; echo err >&2; exit 5' 2>/dev/null; echo "rc=$?"
echo "-- cli stdin piped";               echo piped-in | timeout 20 docker exec -i exectest cat
echo "-- cli 20s exec";                  s=$(date +%s); timeout 60 docker exec exectest sh -c 'sleep 20; echo done-20'; echo "elapsed=$(( $(date +%s) - s ))s"
echo "-- cli detached";                  timeout 30 docker exec -d exectest sleep 1; echo "rc=$?"
docker rm -f exectest >/dev/null
