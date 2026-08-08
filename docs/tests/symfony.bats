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
  PROJNAME=my-symfony-composer-site

  run mkdir -p ${PROJNAME} && cd ${PROJNAME}
  assert_success

  run ddev config --project-type=symfony --docroot=public
  assert_success

  run ddev start -y
  assert_success

  _extra_info

  # Composer has no movable "lts" alias (unlike symfony-cli's --version=lts
  # below), so this pins to the current LTS branch explicitly. See
  # https://symfony.com/releases for the current LTS when this needs bumping.
  run ddev composer create-project symfony/skeleton:"7.4.*"
  assert_success

  run bash -c 'printf "x\n" | ddev composer require webapp'
  assert_success

  # validate ddev launch
  DDEV_DEBUG=true run ddev launch
  assert_output "FULLURL ${PRIMARY_URL}"
  assert_success

  # validate running project
  run curl -sfIv ${PRIMARY_URL}
  assert_output --partial "server: nginx"
  assert_output --partial "HTTP/2 404"
  run curl ${PRIMARY_URL}
  assert_output --partial "<title>Welcome to Symfony!</title>"
  assert_output --partial "You are seeing this page because the homepage URL is not configured and"
  assert_output --partial "<a target=\"_blank\" href=\"https://symfony.com/community#interact\">Follow Symfony</a>"
  assert_success
}

@test "Symfony CLI quickstart with $(ddev --version)" {
  PROJNAME=my-symfony-cli-site

  run mkdir -p ${PROJNAME} && cd ${PROJNAME}
  assert_success

  run ddev config --project-type=symfony --docroot=public
  assert_success

  run ddev start -y
  assert_success

  _extra_info

  run ddev exec symfony check:requirements
  assert_success

  # --version=lts always resolves to the current long-term support release,
  # see https://symfony.com/doc/current/setup.html#symfony-lts-versions.
  run ddev exec symfony new temp --webapp --version=lts
  assert_success

  run ddev exec 'rsync -rltgopD temp/ ./ && rm -rf temp'
  assert_success

  # validate ddev launch
  DDEV_DEBUG=true run ddev launch
  assert_output "FULLURL ${PRIMARY_URL}"
  assert_success

  # validate running project
  run curl -sfIv ${PRIMARY_URL}
  assert_output --partial "server: nginx"
  assert_output --partial "HTTP/2 404"
  run curl ${PRIMARY_URL}
  assert_output --partial "<title>Welcome to Symfony!</title>"
  assert_output --partial "You are seeing this page because the homepage URL is not configured and"
  assert_output --partial "<a target=\"_blank\" href=\"https://symfony.com/community#interact\">Follow Symfony</a>"
  assert_success
}
