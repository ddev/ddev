#!/bin/bash

# .circleci/trigger_release.sh x --release-tag="v1.12.0-20" --circleci-token=circletoken  --build-image-tarballs=false --windows-signing-password=winsignpasswd --macos-signing-password=macsigningpwd --macos-app-password="macapppwd"

# .circleci/trigger_release.sh --release-tag=v1.11.1 --circleci-token=circletoken --github-token=githubtoken --build-image-tarballs=true --windows-signing-password=winsignpwd —macos-signing-password=macsignpwd —macos_app_password=macapppwd | jq -r 'del(.circle_yml)'

# api docs: https://circleci.com/docs/api
# Trigger a new job: https://circleci.com/docs/api/v1-reference/#new-build

set -o errexit -o pipefail -o noclobber -o nounset

GITHUB_PROJECT=drud/ddev
BUILD_IMAGE_TARBALLS=true
GITHUB_ORG=drud

# Long option parsing example: https://stackoverflow.com/a/29754866/215713
# On macOS this requires `brew install gnu-getopt`
OS=$(uname -s)
if [ ${OS} = "Darwin" ]; then PATH="/usr/local/opt/gnu-getopt/bin:$PATH"; fi
if [ ${OS} = "Darwin" ] && [ ! -f "/usr/local/opt/gnu-getopt/bin/getopt" ]; then
    echo "This script requires `brew install gnu-getopt`" && exit 1
fi

! getopt --test > /dev/null
if [[ ${PIPESTATUS[0]} -ne 4 ]]; then
    echo '`getopt --test` failed in this environment.'
    exit 1
fi

LONGOPTS=circleci-token:,github-token:,release-tag:,github-project:,windows-signing-password:,macos-signing-password:,build-image-tarballs:,chocolatey-api-key:,github-org:,macos-app-password:

! PARSED=$(getopt --longoptions=$LONGOPTS --name "$0" -- "$@" dummyarg)
if [[ ${PIPESTATUS[0]} -ne 0 ]]; then
    # e.g. return value is 1
    #  then getopt has complained about wrong arguments to stdout
    printf "\n\nFailed parsing options:\n"
    getopt --longoptions=$LONGOPTS --name "$0" -- "$@"
    exit 2
fi

eval set -- "$PARSED"

while true; do
    case "$1" in
    --circleci-token)
        CIRCLE_TOKEN=$2
        shift 2
        ;;
    --github-token)
        GITHUB_TOKEN=$2
        shift 2
        ;;
    --release-tag)
        RELEASE_TAG=$2
        shift 2
        ;;
    --github-project)
        GITHUB_PROJECT=$2
        shift 2
        ;;
    --windows-signing-password)
        DDEV_WINDOWS_SIGNING_PASSWORD=$2
        shift 2
        ;;
    --chocolatey-api-key)
        CHOCOLATEY_API_KEY=$2
        shift 2
        ;;
    --macos-signing-password)
        DDEV_MACOS_SIGNING_PASSWORD=$2
        shift 2
        ;;
    --macos-app-password)
        DDEV_MACOS_APP_PASSWORD=$2
        shift 2
        ;;
    # For debugging we can set BUILD_IMAGE_TARBALLS=false to avoid waiting for that.
    --build-image-tarballs)
        BUILD_IMAGE_TARBALLS=$2
        shift 2
        ;;
    # For debugging we can set GITHUB_ORG=rfay so chocolatey will look there for the binaries.
    -o|--github-org)
        GITHUB_ORG=$2
        shift 2
        ;;
    --)
        break;
    esac
done

trigger_build_url=https://circleci.com/api/v1.1/project/github/$GITHUB_PROJECT?circle-token=${CIRCLE_TOKEN}

set -x
BUILD_PARAMS="\"CIRCLE_JOB\": \"release_build\", \"job_name\": \"release_build\", \"GITHUB_TOKEN\":\"${GITHUB_TOKEN:-}\", \"RELEASE_TAG\": \"${RELEASE_TAG}\",\"DDEV_WINDOWS_SIGNING_PASSWORD\":\"${DDEV_WINDOWS_SIGNING_PASSWORD:-}\",\"DDEV_MACOS_SIGNING_PASSWORD\":\"${DDEV_MACOS_SIGNING_PASSWORD:-}\",\"DDEV_MACOS_APP_PASSWORD\":\"${DDEV_MACOS_APP_PASSWORD:-}\",\"CHOCOLATEY_API_KEY\":\"${CHOCOLATEY_API_KEY:-}\",\"BUILD_IMAGE_TARBALLS\":\"${BUILD_IMAGE_TARBALLS:-true}\",\"GITHUB_ORG\":\"${GITHUB_ORG}\""
if [ "${RELEASE_TAG:-}" != "" ]; then
    DATA="\"tag\": \"$RELEASE_TAG\","
fi

DATA="${DATA} \"build_parameters\": { ${BUILD_PARAMS} } "

curl -X POST -sS \
  --header "Content-Type: application/json" \
  --data "{ ${DATA} }" \
    $trigger_build_url

