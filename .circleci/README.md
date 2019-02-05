# CircleCI Script Usage

## Tags build automatically

If you just push a tag, circleci will build that tag and make its artifacts available both on CircleCI and by creating a GitHub release draft containing the artifacts. This is a result of the tag_build *workflow*, which calls the job.

## Push a release with trigger_release.sh

`.circleci/trigger_release.sh circlepikey3ea58316baf928b <VERSION> githubpersonaltokenc1ad9f7c353962dea  | jq -r 'del(.circle_yml)'`

## trigger_job.sh options

trigger_job.sh can be used to trigger *jobs* but not workflows. As our Circleci strategy is mostly built around workflows, this is less useful than it used to be.

.circleci/trigger_job.sh <circle_token> tag_build drud/ddev <optional_branch> <github_personal_access_token>  <release_tag>  | jq -r

trigger_job.sh can trigger a nightly or a normal build. It triggers *jobs* not workflows.

It always requires a circle token for the second argument, and by default will run the main build workflow:

* trigger normal build:
`.circleci/trigger_job.sh 0123456773884887377aabvb `

* trigger nightly build in drud/ddev (default):
`.circleci/trigger_job.sh 0123456773884887377aabvb nightly_build`

* trigger nightly using rfay/ddev master branch (testing must be enabled on rfay/ddev):
`.circleci/trigger_job.sh 0123456773884887377aabvb nightly_build rfay/ddev`

* trigger nightly on rfay/ddev on branch 20170803_workflows_contexts:
`.circleci/trigger_job.sh 0123456773884887377aabvb nightly_build rfay/ddev 20170803_workflows_contexts`
