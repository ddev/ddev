#!/usr/bin/env bash
# docker exec -w with a directory that does not exist in the container.
# Docker fails the exec (exit 127, "chdir to cwd ... no such file or directory");
# socktainer runs it there and leaves the directory created.
# Compare with Docker Engine: DOCKER_CONTEXT=orbstack handoff/execworkdirrepro/repro.sh
set -u
c=execworkdir-repro
trap 'docker rm -f "$c" >/dev/null 2>&1' EXIT
docker rm -f "$c" >/dev/null 2>&1
docker run -d --name "$c" alpine sleep 300 >/dev/null
echo "== before: ls -d /does/not/exist"
docker exec "$c" ls -d /does/not/exist; echo "exit $?"
echo "== docker exec -w /does/not/exist $c pwd"
docker exec -w /does/not/exist "$c" pwd; echo "exit $?"
echo "== after: ls -d /does/not/exist"
docker exec "$c" ls -d /does/not/exist; echo "exit $?"
