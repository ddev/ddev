#!/usr/bin/env bash
# GET /images/json with a label filter nothing matches. Docker Engine lists no
# images; socktainer ignores the filter and lists them all.
set -u
docker pull -q alpine >/dev/null
echo "images with label repro.none=1 (expect none):"
docker images --filter label=repro.none=1 --format '  {{.Repository}}:{{.Tag}}'
echo "total images: $(docker images -q | wc -l | tr -d ' ')"
