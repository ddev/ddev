# CircleCI Script Usage

## Tags build automatically

If you just push a tag, circleci will build that tag and make its artifacts available both on CircleCI and by creating a GitHub release draft containing the artifacts. This is a result of the tag_build *workflow*, which calls the job.

## trigger_build.sh is obsolete since we use workflows

Sadly, the below information is currently inoperative due to CircleCI not supporting access to their API if workflows are used...

trigger_build.sh can trigger a nightly or a normal build. It triggers *jobs* not workflows.

It always requires a circle token for the second argument, and by default will run the main build workflow:

* trigger normal build:
`.circleci/trigger_build.sh 0123456773884887377aabvb `

* trigger nightly build in drud/ddev (default):
`.circleci/trigger_build.sh 0123456773884887377aabvb nightly_build`

* trigger nightly using rfay/ddev master branch (testing must be enabled on rfay/ddev):
`.circleci/trigger_build.sh 0123456773884887377aabvb nightly_build rfay/ddev`

* trigger nightly on rfay/ddev on branch 20170803_workflows_contexts:
`.circleci/trigger_build.sh 0123456773884887377aabvb nightly_build rfay/ddev 20170803_workflows_contexts`
