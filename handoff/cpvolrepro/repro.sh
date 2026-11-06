#!/usr/bin/env bash
# docker cp of a directory into a running container, at a named-volume mount and
# on the rootfs. socktainer creates the directories under the volume but drops
# every file; Docker Engine copies everything to both.
set -u
src=$(mktemp -d)
mkdir -p "$src/sub" && echo hello > "$src/a.txt" && echo sub > "$src/sub/b.txt"
docker rm -f cpc >/dev/null 2>&1; docker volume rm cpvol >/dev/null 2>&1
docker volume create cpvol >/dev/null
docker run -d --name cpc -v cpvol:/mnt/v alpine sleep 300 >/dev/null
docker exec cpc mkdir -p /root/r
docker cp "$src/." cpc:/mnt/v/ && echo "cp to volume: ok"
docker cp "$src/." cpc:/root/r/ && echo "cp to rootfs: ok"
echo "volume:"; docker exec cpc find /mnt/v -mindepth 1 | sort
echo "rootfs:"; docker exec cpc find /root/r -mindepth 1 | sort
docker rm -f cpc >/dev/null; docker volume rm cpvol >/dev/null; rm -rf "$src"
