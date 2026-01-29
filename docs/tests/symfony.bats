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
  _skip_if_embargoed "symfony-composer"

  run mkdir ${PROJNAME} && cd ${PROJNAME}
  assert_success

  run ddev config --project-type=symfony --docroot=public --php-version=8.3
  assert_success

  run ddev start -y
  assert_success

  run ddev composer create-project symfony/skeleton
  assert_success

  run bash -c 'printf "x\n" | ddev composer require webapp'
  assert_success

  # validate ddev launch
  DDEV_DEBUG=true run ddev launch
  assert_output "FULLURL https://${PROJNAME}.ddev.site"
  assert_success

  # validate running project
  run curl -sfIv https://${PROJNAME}.ddev.site
  assert_output --partial "server: nginx"
  assert_output --partial "HTTP/2 404"
  run curl https://${PROJNAME}.ddev.site
  assert_output --partial "<title>Welcome to Symfony!</title>"
  assert_output --partial "You are seeing this page because the homepage URL is not configured and"
  assert_output --partial "<a target=\"_blank\" href=\"https://symfony.com/community#interact\">Follow Symfony</a>"
  assert_success
}

@test "Symfony CLI quickstart with $(ddev --version)" {
  _skip_if_embargoed "symfony-cli"

  run mkdir ${PROJNAME} && cd ${PROJNAME}
  assert_success

  run ddev config --project-type=symfony --docroot=public
  assert_success

  run ddev start -y
  assert_success

  run ddev exec symfony check:requirements
  assert_success

  run ddev exec symfony new temp --webapp
  assert_success

  run ddev exec 'rsync -rltgopD temp/ ./ && rm -rf temp'
  assert_success

  # validate ddev launch
  DDEV_DEBUG=true run ddev launch
  assert_output "FULLURL https://${PROJNAME}.ddev.site"
  assert_success

  # validate running project
  run curl -sfIv https://${PROJNAME}.ddev.site
  assert_output --partial "server: nginx"
  assert_output --partial "HTTP/2 404"
  run curl https://${PROJNAME}.ddev.site
  assert_output --partial "<title>Welcome to Symfony!</title>"
  assert_output --partial "You are seeing this page because the homepage URL is not configured and"
  assert_output --partial "<a target=\"_blank\" href=\"https://symfony.com/community#interact\">Follow Symfony</a>"
  assert_success
}
