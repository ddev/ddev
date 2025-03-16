#!/usr/bin/env bats

setup() {
  PROJNAME=my-symfony-site
  load 'common-setup'
  _common_setup
}

# executed after each test
teardown() {
  _common_teardown
}

@test "Symfony Composer quickstart with $(ddev --version)" {
  # mkdir ${PROJNAME} && cd ${PROJNAME}
  run mkdir ${PROJNAME} && cd ${PROJNAME}
  assert_success

  # config --project-type=symfony --docroot=public
  run ddev config --project-type=symfony --docroot=public
  assert_success

  # ddev start -y
  run ddev start -y
  assert_success

  # ddev composer create symfony/skeleton
  run ddev composer create symfony/skeleton
  assert_success

  # bash -c 'printf "x\n" | ddev composer require webapp'
  run bash -c 'printf "x\n" | ddev composer require webapp'
  assert_success

  # validate ddev launch
  run bash -c "DDEV_DEBUG=true ddev launch"
  assert_output "FULLURL https://${PROJNAME}.ddev.site"
  assert_success
  # validate running project
  run curl -sf https://${PROJNAME}.ddev.site/index.php
  assert_success
  assert_output --partial "You are seeing this page because the homepage URL is not configured and"
}

@test "Symfony CLI quickstart with $(ddev --version)" {
  # mkdir ${PROJNAME} && cd ${PROJNAME}
  run mkdir ${PROJNAME} && cd ${PROJNAME}
  assert_success

  # ddev config --project-type=symfony --docroot=public
  run ddev config --project-type=symfony --docroot=public
  assert_success

  # ddev start -y
  run ddev start -y
  assert_success

  # ddev exec symfony check:requirements
  run ddev exec symfony check:requirements
  assert_success

  # ddev exec symfony new temp --version="7.1.*" --webapp
  run ddev exec symfony new temp --version="7.1.*" --webapp
  assert_success

  # ddev exec 'rsync -rltgopD temp/ ./ && rm -rf temp'
  run ddev exec 'rsync -rltgopD temp/ ./ && rm -rf temp'
  assert_success

  # validate ddev launch
  run bash -c "DDEV_DEBUG=true ddev launch"
  assert_output "FULLURL https://${PROJNAME}.ddev.site"
  assert_success
}
