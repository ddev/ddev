#!/bin/bash

# trigger_release.sh $circle_token $release_tag $github_token $project_optional

# .circleci/trigger_release.sh circlepikey0908b3a58316baf928b v1.5.9 githubpersonaltokenc5ad9f7c353962dea optional/ddev  | jq -r 'del(.circle_yml)'

# api docs: https://circleci.com/docs/api
# Trigger a new job: https://circleci.com/docs/api/v1-reference/#new-build

CIRCLE_TOKEN=$1
RELEASE_TAG=$2
GITHUB_TOKEN=$3
PROJECT=${4:-drud/ddev}

trigger_build_url=https://circleci.com/api/v1.1/project/github/$PROJECT?circle-token=${CIRCLE_TOKEN}

set -x
BUILD_PARAMS="\"CIRCLE_JOB\": \"release_build\", \"job_name\": \"release_build\", \"GITHUB_TOKEN\":\"${GITHUB_TOKEN}\", \"RELEASE_TAG\": \"${RELEASE_TAG}\""
if [ "${RELEASE_TAG}" != "" ]; then
    DATA="\"tag\": \"$RELEASE_TAG\","
fi

DATA="${DATA} \"build_parameters\": { ${BUILD_PARAMS} } "

curl -X POST -sS \
  --header "Content-Type: application/json" \
  --data "{ ${DATA} }" \
    $trigger_build_url

