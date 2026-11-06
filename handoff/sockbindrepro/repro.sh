#!/usr/bin/env bash
# Bind-mount two host Unix sockets that differ only in name and connect to each
# from a container. socktainer drops the one named *.sock.
# Compare with Docker Engine: DOCKER_CONTEXT=orbstack handoff/sockbindrepro/repro.sh
set -u
mkdir -p ~/tmp
dir=$(mktemp -d ~/tmp/sockbind.XXXX)
trap 'kill $(jobs -p) 2>/dev/null; rm -rf "$dir"' EXIT
for name in echo echo.sock; do
  socat UNIX-LISTEN:"$dir/$name",fork SYSTEM:"echo hello from host via $name" &
done
sleep 1
for name in echo echo.sock; do
  printf '%-10s ' "$name:"
  docker run --rm -v "$dir/$name:/mnt/s" alpine/socat -u UNIX-CONNECT:/mnt/s - 2>&1 | tail -1
done
