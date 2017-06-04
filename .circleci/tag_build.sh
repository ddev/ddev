#!/bin/bash

# from https://circleci.com/docs/1.0/nightly-builds/
# See also https://circleci.com/docs/2.0/defining-multiple-jobs/

# ag_build.sh $circle_token $tag $project_optional

CIRCLE_TOKEN=$1
TAG=$2
PROJECT=${3:-drud/ddev}

# https://circleci.com/api/v1.1/project/:vcs-type/:username/:project?circle-token=:token

trigger_build_url=https://circleci.com/api/v1.1/project/github/$PROJECT?circle-token=$CIRCLE_TOKEN

echo $trigger_build_url

curl --data "tag=$TAG" --data "build_parameters[CIRCLE_JOB]=tag_build" $trigger_build_url