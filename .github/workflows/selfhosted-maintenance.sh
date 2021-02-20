#!/bin/bash


set -o errexit
set -o pipefail
set -o nounset
set -x

# On macOS, restart docker to avoid bugs where containers can't be deleted
if [ "${OSTYPE%%[0-9]*}" = "darwin" ]; then
  killall Docker || true
  nohup /Applications/Docker.app/Contents/MacOS/Docker --unattended &
  sleep 10
fi

# Make sure docker is working
echo "Waiting for docker to come up: $(date)"
date && timeout -v 10m bash -c 'while ! docker ps >/dev/null 2>&1 ; do
  sleep 10
  echo "Waiting for docker to come up: $(date)"
done'
echo "Testing again to make sure docker came up: $(date)"
if ! docker ps >/dev/null 2>&1 ; then
  echo "Docker is not running, exiting"
  exit 1
fi

if [ ! -z "${DOCKERHUB_PULL_USERNAME:-}" ]; then
  set +x
  echo "${DOCKERHUB_PULL_PASSWORD:-}" | docker login --username "${DOCKERHUB_PULL_USERNAME}" --password-stdin
  set -x
fi

rm -rf ~/.ddev/Test* ~/.ddev/global_config.yaml ~/.ddev/homeadditions ~/.ddev/commands

# Run any testbot maintenance that may need to be done
echo "running selfhosted-upgrades.sh"
bash $(dirname $0)/selfhosted-upgrades.sh

echo "cleaning up docker and Test directories"
echo "Warning: deleting all docker containers and deleting ~/.ddev/Test*"
ddev poweroff || true
if [ "$(docker ps -aq | wc -l )" -gt 0 ] ; then
	docker rm -f $(docker ps -aq) >/dev/null 2>&1 || true
fi
docker system prune --volumes --force >/dev/null || true

# Our testbot should be sane, run the testbot checker to make sure.
./.github/workflows/sanetestbot.sh

# Update any images that could have changed
( docker images | awk '/drud/ {print $1":"$2 }' | xargs -L1 docker pull ) || true

# homebrew sometimes removes /usr/local/etc/my.cnf.d
mkdir -p "$(brew --prefix)/etc/my.cnf.d"
