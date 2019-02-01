#!/bin/bash

# from https://circleci.com/docs/1.0/nightly-builds/
# See also https://circleci.com/docs/2.0/defining-multiple-jobs/

# trigger_job.sh $circle_token $project_optional $branch_optional

CIRCLE_TOKEN=$1
JOB=${2:-nightly_build}
PROJECT=${3:-drud/ddev}
BRANCH=${4:-master}

trigger_build_url=https://circleci.com/api/v1.1/project/github/$PROJECT/tree/$BRANCH?circle-token=$CIRCLE_TOKEN

curl --data "build_parameters[CIRCLE_JOB]=$JOB" $trigger_build_url
