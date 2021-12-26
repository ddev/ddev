#!/bin/bash
# This script is used to build drud/ddev using buildkite

# Use docker-compose v2 on WSL2 to make sure we get testing on it
if [ ! -z "${WSL_DISTRO_NAME:-}" ]; then
  docker-compose enable-v2
else
  docker-compose disable-v2
fi

export PATH=$PATH:/home/linuxbrew/.linuxbrew/bin

echo "buildkite building ${BUILDKITE_JOB_ID:-} at $(date) on $(hostname) as USER=${USER} for OS=${OSTYPE} in ${PWD} with golang=$(go version | awk '{print $3}') docker-desktop=$(scripts/docker-desktop-version.sh) docker=$(docker --version | awk '{print $3}') and $(docker-compose --version) ddev version=$(ddev --version | awk '{print $3}'))"

export GOTEST_SHORT=1
export DDEV_NONINTERACTIVE=true
export DDEV_DEBUG=true

set -o errexit
set -o pipefail
set -o nounset
set -x

# On macOS, restart docker to avoid bugs where containers can't be deleted
#if [ "${OSTYPE%%[0-9]*}" = "darwin" ]; then
#  killall Docker || true
#  nohup /Applications/Docker.app/Contents/MacOS/Docker --unattended &
#  sleep 10
#fi

export TIMEOUT_CMD="timeout -v"
if [ ${OSTYPE%%-*} = "linux" ]; then
  TIMEOUT_CMD="timeout"
fi
# Make sure docker is working
echo "Waiting for docker to come up: $(date)"
date && ${TIMEOUT_CMD} 10m bash -c 'while ! docker ps >/dev/null 2>&1 ; do
  sleep 10
  echo "Waiting for docker to come up: $(date)"
done'
echo "Testing again to make sure docker came up: $(date)"
if ! docker ps >/dev/null 2>&1 ; then
  echo "Docker is not running, exiting"
  exit 1
fi

# Make sure we have a reasonable mutagen setup
if command -v mutagen >/dev/null ; then
  mutagen daemon stop || true
fi
~/.ddev/.mutagen/bin/mutagen daemon stop || true
if command -v killall >/dev/null ; then
  killall mutagen || true
fi

# Run any testbot maintenance that may need to be done
echo "--- running testbot_maintenance.sh"
bash "$(dirname $0)/testbot_maintenance.sh" || true

echo "--- cleaning up docker and Test directories"
echo "Warning: deleting all docker containers and deleting ~/.ddev/Test*"
ddev poweroff || true
if [ "$(docker ps -aq | wc -l )" -gt 0 ] ; then
	docker rm -f $(docker ps -aq) >/dev/null 2>&1 || true
fi
docker system prune --volumes --force >/dev/null || true

# Our testbot should be sane, run the testbot checker to make sure.
echo "--- running sanetestbot.sh"
./.buildkite/sanetestbot.sh

# Update all images that could have changed
( docker images | awk '/drud|phpmyadmin/ {print $1":"$2 }' | xargs -L1 docker pull ) || true

# homebrew sometimes removes /usr/local/etc/my.cnf.d
if command -v brew >/dev/null; then
  mkdir -p "$(brew --prefix)/etc/my.cnf.d"
fi

echo "--- Running tests..."
make test
RV=$?
echo "test.sh completed with status=$RV"
ddev poweroff || true

docker-compose disable-v2
exit $RV
