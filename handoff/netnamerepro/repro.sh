#!/usr/bin/env bash
# Create networks whose names Docker Engine accepts: 63 and 64 lowercase
# characters, and a name with uppercase letters. socktainer rejects the last
# two with "Something went wrong." Compose names networks <project>_default.
# Compare with Docker Engine: DOCKER_CONTEXT=orbstack handoff/netnamerepro/repro.sh
set -u
n63=netname-repro-$(printf 'a%.0s' $(seq 49))
n64=${n63}b
names=("$n63" "$n64" "NetName-Repro" "netname-repro_default")
trap 'for n in "${names[@]}"; do docker network rm "$n" >/dev/null 2>&1; done' EXIT
for n in "${names[@]}"; do
  echo "== docker network create $n (${#n} chars)"
  docker network create "$n" >/dev/null && echo "ok" || echo "exit $?"
done
