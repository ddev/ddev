#!/bin/bash
# This script is used to build ddev/ddev using buildkite

set -eu -o pipefail

export PATH=$PATH:/home/linuxbrew/.linuxbrew/bin

# GOTEST_SHORT=12 means drupal10
export GOTEST_SHORT=12

export DOCKER_SCAN_SUGGEST=false
export DOCKER_SCOUT_SUGGEST=false

# On macOS, we can have several different docker providers, allow testing all
# In cleanup, stop everything we know of but leave either Orbstack or Docker Desktop running
if [ "${OSTYPE%%[0-9]*}" = "darwin" ]; then
  function cleanup {
    command -v orb 2>/dev/null && echo "Stopping orbstack" && (orb stop &)
    if [ -f /Applications/Docker.app ]; then echo "Stopping Docker Desktop" && (killall com.docker.backend || true); fi
    command -v colima 2>/dev/null && echo "Stopping colima" && (colima stop || true)
    command -v colima 2>/dev/null && echo "Stopping colima_vz" && (colima stop vz || true)
    command -v limactl 2>/dev/null && echo "Stopping lima" && (limactl stop lima-vz || true)
    if [ -f ~/.rd/bin/rdctl ]; then echo "Stopping Rancher Desktop" && (~/.rd/bin/rdctl shutdown || true); fi
    docker context use default
    # Leave orbstack running as the most likely to be reliable, otherwise Docker Desktop
    if command -v orb 2>/dev/null ; then
      echo "Starting orbstack" && (orb start &)
    else
      open -a /Applications/Docker.app
    fi
  }
  trap cleanup EXIT

  # Start with a predictable docker provider running
  cleanup

  echo "starting docker context situation:"
  docker context ls

  # Now start the docker provider we want
  case ${DOCKER_TYPE:=none} in
    "colima")
      colima start
      colima restart
      docker context use colima
      ;;
    "colima_vz")
      colima start vz
      colima restart vz
      docker context use colima-vz
      ;;

    "lima")
      limactl start lima-vz
      docker context use lima-lima-vz
      ;;

    "docker-desktop")
      open -a /Applications/Docker.app &
      docker context use desktop-linux
      ;;

    "orbstack")
      orb start &
      docker context use orbstack
      ;;

    "rancher-desktop")
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
echo "Waiting for docker provider to come up: $(date)"
date && ${TIMEOUT_CMD} 3m bash -c 'while ! docker ps >/dev/null 2>&1 ; do
  sleep 10
  echo "Waiting for docker to come up: $(date)"
done'
echo "Testing again to make sure docker came up: $(date)"
if ! docker ps >/dev/null 2>&1 ; then
  echo "Docker is not running, exiting"
  exit 1
fi

echo "buildkite building ${BUILDKITE_JOB_ID:-} at $(date) on $(hostname) as USER=${USER} for OS=${OSTYPE} DOCKER_TYPE=${DOCKER_TYPE:-notset} in ${PWD} with GOTEST_SHORT=${GOTEST_SHORT} golang=$(go version | awk '{print $3}') docker=$(docker version) ddev version=$(ddev --version | awk '{print $3}'))"

case ${DOCKER_TYPE:-none} in
  "docker-desktop")
    echo "docker-desktop=$(scripts/docker-desktop-version.sh)"
    ;;
  "colima*")
    echo "colima=$(colima version)"
    ;;
  "orbstack")
    echo "orbstack=$(orb --version)"
    ;;
  "rancher-desktop")
    echo "rancher-desktop=$(~/.rd/bin/rdctl --version)"
    ;;
  *)
    echo "$DOCKER_TYPE not found"
esac

ddev version

export DDEV_NONINTERACTIVE=true
export DDEV_DEBUG=true

# We can skip builds with commit message of [skip buildkite]
if [[ $BUILDKITE_MESSAGE == *"[skip buildkite]"* ]] || [[ $BUILDKITE_MESSAGE == *"[skip ci]"* ]]; then
  echo "Skipping build because message has '[skip buildkite]' or '[skip ci]'"
  exit 0
fi

# If this is a PR and the diff doesn't have code, skip it
if [ "${BUILDKITE_PULL_REQUEST:-false}" != "false" ] && ! git diff --name-only refs/remotes/origin/${BUILDKITE_PULL_REQUEST_BASE_BRANCH:-} | egrep "^(\.buildkite|Makefile|pkg|cmd|vendor|go\.)" >/dev/null; then
  echo "Skipping buildkite build since no code changes found"
  exit 0
fi

set -x

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
