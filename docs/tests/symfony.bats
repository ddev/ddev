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

  run -0 mkdir -p ${PROJNAME} && cd ${PROJNAME}

  run -0 ddev config --project-type=symfony --docroot=public

  run -0 ddev start -y

  _extra_info

  # Composer has no movable "lts" alias (unlike symfony-cli's --version=lts
  # below), so this pins to the current LTS branch explicitly. See
  # https://symfony.com/releases for the current LTS when this needs bumping.
  run -0 ddev composer create-project symfony/skeleton:"7.4.*"

  run -0 bash -c 'printf "x\n" | ddev composer require webapp'

  # validate ddev launch
  DDEV_DEBUG=true run -0 ddev launch
  assert_output "FULLURL ${PRIMARY_URL}"

  # validate running project
  run curl -sSfI ${PRIMARY_URL}
  assert_output --partial "server: nginx"
  assert_output --partial "HTTP/2 404"
  run -0 curl ${PRIMARY_URL}
  assert_output --partial "<title>Welcome to Symfony!</title>"
  assert_output --partial "You are seeing this page because the homepage URL is not configured and"
  assert_output --partial "<a target=\"_blank\" href=\"https://symfony.com/community#interact\">Follow Symfony</a>"
}

@test "Symfony CLI quickstart with $(ddev --version)" {
  PROJNAME=my-symfony-cli-site

  run -0 mkdir -p ${PROJNAME} && cd ${PROJNAME}

  run -0 ddev config --project-type=symfony --docroot=public

  run -0 ddev start -y

  _extra_info

  run -0 ddev exec symfony check:requirements

  # --version=lts always resolves to the current long-term support release,
  # see https://symfony.com/doc/current/setup.html#symfony-lts-versions.
  run -0 ddev exec symfony new temp --webapp --version=lts

  run -0 ddev exec 'rsync -rltgopD temp/ ./ && rm -rf temp'

  # validate ddev launch
  DDEV_DEBUG=true run -0 ddev launch
  assert_output "FULLURL ${PRIMARY_URL}"

  # validate running project
  run curl -sSfI ${PRIMARY_URL}
  assert_output --partial "server: nginx"
  assert_output --partial "HTTP/2 404"
  run -0 curl ${PRIMARY_URL}
  assert_output --partial "<title>Welcome to Symfony!</title>"
  assert_output --partial "You are seeing this page because the homepage URL is not configured and"
  assert_output --partial "<a target=\"_blank\" href=\"https://symfony.com/community#interact\">Follow Symfony</a>"
}
