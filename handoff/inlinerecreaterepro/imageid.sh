#!/usr/bin/env bash
# Is the image ID socktainer reports stable, and does it change when the
# build's content changes? Compose recreates a container only when its
# com.docker.compose.image label differs from the image's current ID.
set -u
d="${HOME}/tmp/imageid-check"
img=imageidcheck:latest
rm -rf "${d}" && mkdir -p "${d}" && cd "${d}" || exit 1
build() {
  printf 'FROM busybox:stable\nRUN echo %s > /inline.txt\n' "$1" > Dockerfile
  docker build -q --load -t "${img}" . >/dev/null 2>&1 || docker build -q -t "${img}" . >/dev/null
  for i in 1 2 3; do
    echo "$1 inspect $i: $(docker image inspect "${img}" -f '{{.Id}}')"
  done
  echo "$1 list: $(docker images --no-trunc -q "${img}")"
}
build inline-1
build inline-2
build inline-1
docker rmi "${img}" >/dev/null 2>&1
