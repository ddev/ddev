#!/usr/bin/env bats

setup() {
  PROJNAME=my-grav-site
  load 'common-setup'
  _common_setup
}

# executed after each test
teardown() {
  _common_teardown
}

@test "Grav Composer quickstart with $(ddev --version)" {
  _skip_if_embargoed "grav-composer"

  run mkdir ${PROJNAME} && cd ${PROJNAME}
  assert_success

  run ddev config --php-version=8.3 --omit-containers=db
  assert_success

  run ddev start -y
  assert_success

  run ddev composer create-project getgrav/grav
  assert_success

  run ddev exec gpm install admin -y
  assert_success

  DDEV_DEBUG=true run ddev launch
  assert_output "FULLURL https://${PROJNAME}.ddev.site"
  assert_success

  # validate running project
  run curl -sfIv https://${PROJNAME}.ddev.site
  assert_output --partial "set-cookie: grav-site-"
  assert_output --partial "location: /admin"
  assert_output --partial "HTTP/2 302"
  assert_success
}


@test "Grav Git Clone quickstart with $(ddev --version)" {
  _skip_if_embargoed "grav-git"

  run mkdir my-grav-site && cd my-grav-site
  assert_success

  run git clone -b master https://github.com/getgrav/grav.git .
  assert_success

  run ddev config --php-version=8.3 --omit-containers=db
  assert_success

  run ddev start -y
  assert_success

  run ddev composer install
  assert_success

  run ddev exec grav install
  assert_success

  run ddev exec gpm install admin -y
  assert_success

  DDEV_DEBUG=true run ddev launch
  assert_output "FULLURL https://${PROJNAME}.ddev.site"
  assert_success

  # validate running project
  run curl -sfIv https://${PROJNAME}.ddev.site
  assert_output --partial "set-cookie: grav-site-"
  assert_output --partial "location: /admin"
  assert_output --partial "HTTP/2 302"
  assert_success
}
