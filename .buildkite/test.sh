#!/bin/bash
# This script is used to build ddev/ddev using buildkite

export PATH=$PATH:/home/linuxbrew/.linuxbrew/bin

# GOTEST_SHORT=8 means drupal9
export GOTEST_SHORT=8

export DOCKER_SCAN_SUGGEST=false
export DOCKER_SCOUT_SUGGEST=false

# On macOS, we can have several different docker providers, allow testing all
if [ "${OSTYPE%%[0-9]*}" = "darwin" ]; then
  function cleanup {
#      docker context use default
    true
  }
  trap cleanup EXIT

  echo "original docker context situation:"
  docker context ls
  case ${DOCKER_TYPE} in
    "colima")
      colima stop vz || true
      ~/.rd/bin/rdctl shutdown || true
      orb stop &
      killall com.docker.backend || true
      colima start
      colima restart
      docker context use colima
      ;;
    "colima_vz")
      ~/.rd/bin/rdctl shutdown || true
      colima stop || true
      orb stop &
      killall com.docker.backend || true
      colima start vz
      colima restart vz
      docker context use colima-vz
      ;;

    "docker-desktop")
      orb stop &
      ~/.rd/bin/rdctl shutdown || true
      colima stop || true
      colima stop vz || true
      open -a Docker &
      docker context use desktop-linux
      ;;
    "orbstack")
      ~/.rd/bin/rdctl shutdown || true
      colima stop || true
      colima stop vz || true
      killall com.docker.backend || true
      orb start &
      docker context use orbstack
      ;;
    "rancher-desktop")
      killall com.docker.backend || true
      colima stop || true
      colima stop vz || true
      orb stop &
      ~/.rd/bin/rdctl start
      for i in {1..120}; do
        if docker context use rancher-desktop >/dev/null 2>&1 ; then
          break
        fi
        echo "$(date): Waiting for rancher-desktop context to be available"
        sleep 1
      done
      docker context use rancher-desktop
      ;;

    *)
      echo "no DOCKER_TYPE specified, exiting" && exit 10
      ;;
  esac
fi

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

echo "buildkite building ${BUILDKITE_JOB_ID:-} at $(date) on $(hostname) as USER=${USER} for OS=${OSTYPE} DOCKER_TYPE=${DOCKER_TYPE:notset} in ${PWD} with GOTEST_SHORT=${GOTEST_SHORT} golang=$(go version | awk '{print $3}') docker-desktop=$(scripts/docker-desktop-version.sh) docker=$(docker --version | awk '{print $3}') ddev version=$(ddev --version | awk '{print $3}'))"

ddev version

export DDEV_NONINTERACTIVE=true
export DDEV_DEBUG=true

set -o errexit
set -o pipefail
set -o nounset
set -x

# Broken docker context list from https://github.com/docker/for-win/issues/13180
# When this is solved this can be removed.
# The only place we care about non-default context is macOS Colima
if ! docker context list >/dev/null; then
  rm -rf ~/.docker/contexts && docker context list >/dev/null
fi

# If this is a PR and the diff doesn't have code, skip it
if [ "${BUILDKITE_PULL_REQUEST:-false}" != "false" ] && ! git diff --name-only refs/remotes/origin/${BUILDKITE_PULL_REQUEST_BASE_BRANCH:-} | egrep "^(\.buildkite|Makefile|pkg|cmd|vendor|go\.)" >/dev/null; then
  echo "Skipping build since no code changes found"
  exit 0
fi

# We can skip builds with commit message of [skip buildkite]
if [[ $BUILDKITE_MESSAGE == *"[skip buildkite]"* ]] || [[ $BUILDKITE_MESSAGE == *"[skip ci]"* ]]; then
  echo "Skipping build because message has '[skip buildkite]' or '[skip ci]'"
  exit 0
fi

# On macOS, restart docker to avoid bugs where containers can't be deleted
#if [ "${OSTYPE%%[0-9]*}" = "darwin" ]; then
#  killall Docker || true
#  nohup /Applications/Docker.app/Contents/MacOS/Docker --unattended &
#  sleep 10
#fi

# We don't want any docker volumes to be existing and changing behavior
docker volume prune -a -f >/dev/null 2>&1 || true

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
( docker images | awk '/ddev|traefik|postgres/ {print $1":"$2 }' | xargs -L1 docker pull ) || true

# homebrew sometimes removes /usr/local/etc/my.cnf.d
if command -v brew >/dev/null; then
  mkdir -p "$(brew --prefix)/etc/my.cnf.d"
fi

# Make sure we start with mutagen daemon off.
unset MUTAGEN_DATA_DIRECTORY
if [ -f ~/.ddev/bin/mutagen -o -f ~/.ddev/bin/mutagen.exe ]; then
  MUTAGEN_DATA_DIRECTORY=~/.ddev_mutagen_data_directory/ ~/.ddev/bin/mutagen sync terminate -a || true
  MUTAGEN_DATA_DIRECTORY=~/.mutagen ~/.ddev/bin/mutagen daemon stop || true
  MUTAGEN_DATA_DIRECTORY=~/.ddev_mutagen_data_directory/ ~/.ddev/bin/mutagen daemon stop || true
fi
if command -v killall >/dev/null ; then
  killall mutagen || true
fi

echo "--- Running tests..."
make test TESTARGS="-failfast"
RV=$?
echo "test.sh completed with status=$RV"
ddev poweroff || true

exit $RV
