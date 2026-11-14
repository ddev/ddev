#!/usr/bin/env bash
# Two `docker load`s at once, each retagging an existing name to a new image.
# Docker Engine moves both tags every round. Run with DOCKER_CONTEXT=socktainer
# or orbstack. Pass the number of rounds (default 6); SERIAL=1 loads one at a time.
set -u
d="${HOME}/tmp/concurrent-load"
rounds="${1:-6}"
rm -rf "${d}" && mkdir -p "${d}" && cd "${d}" || exit 1
# Distinct images for each name and round, saved as tarballs up front
for n in $(seq 0 "${rounds}"); do
  for name in cla clb; do
    printf 'FROM busybox:stable\nRUN echo %s-%s > /v.txt\n' "${name}" "${n}" > Dockerfile
    docker build -q --load -t "${name}:latest" . >/dev/null 2>&1
    docker save -o "${name}-${n}.tar" "${name}:latest"
    docker image inspect "${name}:latest" -f '{{.Id}}' > "${name}-${n}.id"
  done
done
docker load -i cla-0.tar >/dev/null && docker load -i clb-0.tar >/dev/null
lost=0
for n in $(seq 1 "${rounds}"); do
  if [ -n "${SERIAL:-}" ]; then
    docker load -i "cla-${n}.tar" >/dev/null; docker load -i "clb-${n}.tar" >/dev/null
  else
    docker load -i "cla-${n}.tar" >/dev/null &
    docker load -i "clb-${n}.tar" >/dev/null &
    wait
  fi
  for name in cla clb; do
    want=$(cat "${name}-${n}.id")
    got=$(docker image inspect "${name}:latest" -f '{{.Id}}')
    if [ "${want}" = "${got}" ]; then echo "round ${n} ${name}: ok"; else echo "round ${n} ${name}: STALE (still ${got})"; lost=$((lost+1)); fi
  done
done
echo "stale tags: ${lost} of $((rounds*2))"
docker rmi cla:latest clb:latest >/dev/null 2>&1
