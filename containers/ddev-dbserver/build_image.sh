#!/usr/bin/env bash

# Build a ddev-dbserver image for variety of mariadb/mysql
# and per architecture, optionally push
# By default loads to local docker
# Example:
# ./build_image.sh --db-type=mysql --db-major-version=8.4 --archs=linux/arm64 --tag=v1.23.4-22-gfe969a5bb-dirty --docker-args=

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
LONGOPTS=archs:,db-type:,db-major-version:,db-pinned-version:,docker-args:,docker-org:,tag:,push,help

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
  --docker-org)
    DOCKER_ORG=$2
    shift 2
    ;;
  -h | --help)
    echo "Usage: $0 --db-type [mariadb|mysql] --db-major-version <major> --tag <image_tag> --archs <comma-delimited_architectures> --push --no-load"
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

BASE_IMAGE=${DB_TYPE}

set -x

if [ ${DB_TYPE} = "mysql" ]; then
    # For mysql 5.7, we have to use our own base images at ddev/mysql (due to arm64)
    if [ ${DB_MAJOR_VERSION} = "5.7" ]; then
      BASE_IMAGE=ddev/mysql
    elif [ "${DB_MAJOR_VERSION:-}" = "8.0" ] || [ "${DB_MAJOR_VERSION}" = "8.4" ]; then
      BASE_IMAGE=bitnamilegacy/mysql
    fi
fi
printf "\n\n========== Building ddev/ddev-dbserver-${DB_TYPE}-${DB_MAJOR_VERSION}:${IMAGE_TAG} from ${BASE_IMAGE} for ${ARCHS} with pinned version ${DB_PINNED_VERSION} ==========\n"

# Build up the -t section with optionally both tags
tag_directive="-t ${DOCKER_ORG}/ddev-dbserver-${DB_TYPE}-${DB_MAJOR_VERSION}:${IMAGE_TAG}"
if [[ ${IMAGE_TAG} =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  tag_directive="$tag_directive -t ${DOCKER_ORG}/ddev-dbserver-${DB_TYPE}-${DB_MAJOR_VERSION}:latest"
fi
if [ ! -z ${PUSH:-} ]; then
  echo "building/pushing ddev/ddev-dbserver-${DB_TYPE}-${DB_MAJOR_VERSION}:${IMAGE_TAG}"
  set -x
  docker buildx build --push --platform ${ARCHS} ${DOCKER_ARGS} --build-arg="BASE_IMAGE=${BASE_IMAGE}" --build-arg="DB_PINNED_VERSION=${DB_PINNED_VERSION}" --build-arg="DB_MAJOR_VERSION=${DB_MAJOR_VERSION}" ${tag_directive}  .
  set +x
fi

# By default, load/import into local docker
set -x
if [ -z "${PUSH:-}" ]; then
    echo "Loading to local docker ddev/ddev-dbserver-${DB_TYPE}-${DB_MAJOR_VERSION}:${IMAGE_TAG}"
    docker buildx build --load ${DOCKER_ARGS} --build-arg="DB_TYPE=${DB_TYPE}" --build-arg="DB_MAJOR_VERSION=${DB_MAJOR_VERSION}" --build-arg="BASE_IMAGE=${BASE_IMAGE}" --build-arg="DB_PINNED_VERSION=${DB_PINNED_VERSION}" ${tag_directive} .
fi
