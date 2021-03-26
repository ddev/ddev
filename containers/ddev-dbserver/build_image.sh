#!/usr/bin/env bash

# Build a ddev-dbserver image for variety of mariadb/mysql
# and per architecture, optionally push
# By default loads to local docker

set -eu -o pipefail

OS=$(uname -s)
if [ ${OS} = "Darwin" ]; then
  if ! command -v brew >/dev/null; then
    echo "On macOS, homebrew is required to get gnu-getopt" && exit 1
  fi
  if ! brew info gnu-getopt >/dev/null; then
    echo "On macOS, brew install gnu-getopt"
    exit 1
  fi
  PATH="$(brew --prefix gnu-getopt)/bin:$PATH"
fi

! getopt --test >/dev/null
if [[ ${PIPESTATUS[0]} -ne 4 ]]; then
  echo 'getopt --test` failed in this environment.'
  exit 1
fi

OPTS=-h,-v:,-d:
LONGOPTS=archs:,db-type:,db-major-version:,db-pinned-version:,docker-args:,tag:,push,help

! PARSED=$(getopt --options=$OPTS --longoptions=$LONGOPTS --name "$0" -- "$@")
if [[ ${PIPESTATUS[0]} -ne 0 ]]; then
  # e.g. return value is 1
  #  then getopt has complained about wrong arguments to stdout
  printf "\n\nFailed parsing options:\n"
  getopt --longoptions=$LONGOPTS --name "$0" -- "$@"
  exit 3
fi

eval set -- "$PARSED"

ARCHS=linux/$(../get_arch.sh)
MYARCH=${ARCHS}
PUSH=""
NO_LOAD=""
DB_TYPE=mariadb
DB_MAJOR_VERSION=10.3
IMAGE_TAG=$(git describe --tags --always --dirty)
DOCKER_ARGS=""

while true; do
  case "$1" in
  --db-type | -d)
    DB_TYPE=$2
    shift 2
    ;;
  --db-major-version | -v)
    DB_MAJOR_VERSION=$2
    shift 2
    ;;
  --db-pinned-version)
    DB_PINNED_VERSION=$2
    shift 2
    ;;
  --archs)
    ARCHS=$2
    shift 2
    ;;
  --push)
    PUSH=true
    shift 1
    ;;
  --no-load)
    NO_LOAD=true
    shift 1
    ;;
  --docker-args)
    DOCKER_ARGS=$2
    shift 2
    ;;
  --tag)
    IMAGE_TAG=$2
    shift 2
    ;;
  -h | --help)
    echo "Usage: $0 --db-type [mariadb|mysql] --db-major-version <major> --tag <image_tag> --archs <comma-delimted_architectures> --push --no-load"
    printf "Examples: $0 ./build_image.sh --db-type mysql --db-major-version 8.0 --tag junker99 --archs linux/amd64 --push
  $0 --db-type mariadb --db-major-version 10.3 --tag junker99 --archs linux/amd64,linux/arm64"
    exit 0
    ;;
  --)
    break
    ;;
  esac
done

set -o nounset

if [ -z ${DB_PINNED_VERSION:-} ]; then
  DB_PINNED_VERSION=${DB_MAJOR_VERSION}
fi

printf "\n\n========== Building drud/ddev-dbserver-${DB_TYPE}-${DB_MAJOR_VERSION}:${IMAGE_TAG} for ${ARCHS} with pinned version ${DB_PINNED_VERSION} ==========\n"

if [ ! -z ${PUSH:-} ]; then
  echo "building/pushing drud/ddev-dbserver-${DB_TYPE}-${DB_MAJOR_VERSION}:${IMAGE_TAG}"
  set -x
  docker buildx build --push --platform ${ARCHS} ${DOCKER_ARGS} --build-arg="DB_TYPE=${DB_TYPE}" --build-arg="DB_PINNED_VERSION=${DB_PINNED_VERSION}" --build-arg="DB_VERSION=${DB_MAJOR_VERSION}" -t "drud/ddev-dbserver-${DB_TYPE}-${DB_MAJOR_VERSION}:${IMAGE_TAG}" .
  set +x
fi

# By default, load/import into local docker
if [ -z ${NO_LOAD:-} ]; then
  if [[ ${ARCHS} =~ ${MYARCH} ]]; then
    echo "Loading to local docker drud/ddev-dbserver-${DB_TYPE}-${DB_MAJOR_VERSION}:${IMAGE_TAG}"
    set -x
    docker buildx build --load --platform ${MYARCH} ${DOCKER_ARGS} --build-arg="DB_TYPE=${DB_TYPE}" --build-arg="DB_VERSION=${DB_MAJOR_VERSION}" --build-arg="DB_PINNED_VERSION=${DB_PINNED_VERSION}" -t "drud/ddev-dbserver-${DB_TYPE}-${DB_MAJOR_VERSION}:${IMAGE_TAG}" .
    set +x
  else
    echo "This architecture (${MYARCH}) was not built, so not loading"
    exit
  fi
fi
