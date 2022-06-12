#!/bin/bash
# This script is used to build drud/ddev using buildkite

export PATH=$PATH:/home/linuxbrew/.linuxbrew/bin

echo "buildkite building ${BUILDKITE_JOB_ID:-} at $(date) on $(hostname) as USER=${USER} for OS=${OSTYPE} in ${PWD} with golang=$(go version | awk '{print $3}') docker-desktop=$(scripts/docker-desktop-version.sh) docker=$(docker --version | awk '{print $3}') ddev version=$(ddev --version | awk '{print $3}'))"

export GOTEST_SHORT=1
export DDEV_NONINTERACTIVE=true
export DDEV_DEBUG=true

set -o errexit
set -o pipefail
set -o nounset
set -x

# If this is a PR and the diff doesn't have code, skip it
if [ "${BUILDKITE_PULL_REQUEST}" != "false" ] && ! git diff --name-only refs/remotes/origin/${BUILDKITE_PULL_REQUEST_BASE_BRANCH:-} | egrep "^(Makefile|pkg|cmd|vendor|go\.)"; then
  echo "Skipping build since no code changes found"
  exit 0
fi

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
if [ -f ~/.ddev/.mutagen/bin/mutagen ]; then
  ~/.ddev/.mutagen/bin/mutagen daemon stop || true
fi
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

BUILDKITE_ANALYTICS_OUTPUT=~/tmp/ddevtest
rm -f ${BUILDKITE_ANALYTICS_OUTPUT}/junit.*.xml

echo "--- Running tests..."
make test TESTARGS="-failfast"
RV=$?

echo "test.sh completed with status=$RV"
ddev poweroff || true

set +x
if [ ! -z "${BUILDKITE_ANALYTICS_TOKEN}" ]; then
  for item in ${BUILDKITE_ANALYTICS_OUTPUT}/junit.*.xml; do
    curl \
    -X POST \
    --fail-with-body \
    -H "Authorization: Token token=\"$BUILDKITE_ANALYTICS_TOKEN\"" \
    -F "data=@${item}" \
    -F "format=junit" \
    -F "run_env[CI]=buildkite" \
    -F "run_env[key]=$BUILDKITE_BUILD_ID" \
    -F "run_env[number]=$BUILDKITE_BUILD_NUMBER" \
    -F "run_env[job_id]=$BUILDKITE_JOB_ID" \
    -F "run_env[branch]=$BUILDKITE_BRANCH" \
    -F "run_env[commit_sha]=$BUILDKITE_COMMIT" \
    -F "run_env[message]=$BUILDKITE_MESSAGE" \
    -F "run_env[url]=$BUILDKITE_BUILD_URL" \
    https://analytics-api.buildkite.com/v1/uploads
  done
fi

exit $RV
