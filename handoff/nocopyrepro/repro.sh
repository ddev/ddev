#!/usr/bin/env bash
# Mount a new, empty named volume over an image directory that has content,
# with and without volume-nocopy. Docker leaves the nocopy volume empty;
# socktainer copies the image content into it either way.
# Compare with Docker Engine: DOCKER_CONTEXT=orbstack handoff/nocopyrepro/repro.sh
set -u
trap 'docker volume rm -f nocopy-repro-copy nocopy-repro-nocopy >/dev/null 2>&1' EXIT
docker volume create nocopy-repro-copy >/dev/null
docker volume create nocopy-repro-nocopy >/dev/null
echo "== image /usr/share, no volume:"
docker run --rm alpine ls /usr/share
echo "== new volume, default (copy):"
docker run --rm --mount type=volume,src=nocopy-repro-copy,dst=/usr/share alpine ls /usr/share
echo "== new volume, volume-nocopy:"
docker run --rm --mount type=volume,src=nocopy-repro-nocopy,dst=/usr/share,volume-nocopy alpine ls /usr/share
echo "== nocopy volume afterwards, seen from another mount point:"
docker run --rm -v nocopy-repro-nocopy:/v alpine ls -A /v
echo "(end)"
