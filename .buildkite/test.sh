#!/bin/bash

# This script is used to build drud/ddev using buildkite



echo "--- buildkite building ${BUILDKITE_JOB_ID:-} at $(date) on $(hostname) for OS=$(go env GOOS) in $PWD with golang=$(go version) docker=$(docker version --format '{{.Server.Version}}') and docker-compose $(docker-compose version --short) ddev version=$(ddev --version)"

export GOTEST_SHORT=1
export DRUD_NONINTERACTIVE=true

set -o errexit
set -o pipefail
set -o nounset
set -x

rm -rf ~/.ddev/Test* ~/.ddev/global_config.yaml ~/.ddev/homeadditions ~/.ddev/commands

# There are discrepancies in golang hash checking in 1.11+, so kill off modcache to solve.
# See https://github.com/golang/go/issues/27925
# This can probably be removed when current work is merged 2018-12-27
# go clean -modcache  (Doesn't work due to current bug in golang)
chmod -R u+w ~/go/pkg && rm -rf ~/go/pkg/*

# Kill off any running containers before sanetestbot.
ddev poweroff
ddev stop --unlist --all
docker rm -f $(docker ps -aq) >/dev/null 2>&1 || true

# Run any testbot maintenance that may need to be done
echo "--- running testbot_maintenance.sh"
bash $(dirname $0)/testbot_maintenance.sh

# Our testbot should be sane, run the testbot checker to make sure.
echo "--- running sanetestbot.sh"
./.buildkite/sanetestbot.sh

echo "--- cleaning up docker and Test directories"
echo "Warning: deleting all docker containers and deleting ~/.ddev/Test*"
if [ "$(docker ps -aq | wc -l)" -gt 0 ] ; then
	docker rm -f $(docker ps -aq) >/dev/null 2>&1 || true
fi
docker system prune --volumes --force >/dev/null || true

# Update all images that could have changed
( docker images | awk '/drud/ {print $1":"$2 }' | xargs -L1 docker pull ) || true

# homebrew sometimes removes /usr/local/etc/my.cnf.d
mkdir -p /usr/local/etc/my.cnf.d

echo "Running tests..."
time make test
RV=$?
echo "build.sh completed with status=$RV"
exit $RV
