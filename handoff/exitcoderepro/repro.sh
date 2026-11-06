#!/usr/bin/env bash
# docker inspect reports State.ExitCode 0 for every stopped container on
# socktainer, though docker wait returns the real code. Inspect runs first,
# without a wait, the way a client polling for exit sees it. Compare with
# DOCKER_CONTEXT=orbstack (or any Docker Engine).
set -u
name=exitcoderepro-$$
docker create --name "$name" busybox sh -c 'exit 5' >/dev/null
docker start "$name" >/dev/null
sleep 3
echo "inspect State.Status:     $(docker inspect -f '{{.State.Status}}' "$name")"
echo "inspect State.ExitCode:   $(docker inspect -f '{{.State.ExitCode}}' "$name")"
echo "docker wait:              $(docker wait "$name")"
docker rm "$name" >/dev/null
