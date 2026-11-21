#!/usr/bin/env bash
# docker cp into a running container: are the tar entries' mtime and owner kept?
# Run with DOCKER_CONTEXT=orbstack (or any Docker Engine) to compare.
set -eu -o pipefail

work=$(mktemp -d)
trap 'docker rm -f cpmeta >/dev/null 2>&1 || true; rm -rf "$work"' EXIT

mkdir -p "$work/src"
for i in 1 2 3; do
  echo "$i" >"$work/src/f$i"
  touch -t "202601011200.0$i" "$work/src/f$i"
done
# The tar records owner 33:33 and mtimes 12:00:01-03 UTC-local.
tar -C "$work/src" --uid 33 --gid 33 -cf "$work/a.tar" f1 f2 f3

docker run -d --name cpmeta alpine sleep 300 >/dev/null
docker cp - cpmeta:/tmp <"$work/a.tar"

printf '%-4s %-12s %-12s %s\n' file tar_mtime ctr_mtime ctr_owner
for i in 1 2 3; do
  want=$(stat -f %m "$work/src/f$i" 2>/dev/null || stat -c %Y "$work/src/f$i")
  got=$(docker exec cpmeta stat -c '%Y %u:%g' "/tmp/f$i")
  printf '%-4s %-12s %-12s %s\n' "f$i" "$want" "${got% *}" "${got#* }"
done
