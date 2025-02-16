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
  # mkdir ${PROJNAME} && cd ${PROJNAME}
  run mkdir ${PROJNAME} && cd ${PROJNAME}
  assert_success
  # ddev config --omit-containers=db
  run ddev config --omit-containers=db
  assert_success
  # ddev start -y
  run ddev start -y
  assert_success
  # ddev composer create getgrav/grav
  run ddev composer create getgrav/grav
  assert_success
  # ddev exec gpm install admin -y
  run ddev exec gpm install admin -y
  assert_success
  # ddev launch
  run bash -c "DDEV_DEBUG=true ddev launch"
  assert_output "FULLURL https://${PROJNAME}.ddev.site"
  assert_success
  # validate running project
  run curl -sfI https://${PROJNAME}.ddev.site
  assert_success
  assert_output --partial "set-cookie: grav-site-"
  assert_output --partial "location: /admin"
  assert_output --partial "HTTP/2 302"
}


@test "Grav Git Clone quickstart with $(ddev --version)" {
  # mkdir my-grav-site && cd my-grav-site
  run mkdir my-grav-site && cd my-grav-site
  assert_success
  # git clone -b master https://github.com/getgrav/grav.git .
  run git clone -b master https://github.com/getgrav/grav.git .
  assert_success
  # ddev config --omit-containers=db
  run ddev config --omit-containers=db
  assert_success
  # ddev start -y
  run ddev start -y
  assert_success
  # ddev composer install
  run ddev composer install
  assert_success
  # ddev exec grav install
  run ddev exec grav install
  assert_success
  # ddev exec gpm install admin -y
  run ddev exec gpm install admin -y
  assert_success
  # ddev launch
  run bash -c "DDEV_DEBUG=true ddev launch"
  assert_output "FULLURL https://${PROJNAME}.ddev.site"
  assert_success
  # validate running project
  run curl -sfI https://${PROJNAME}.ddev.site
  assert_success
  assert_output --partial "set-cookie: grav-site-"
  assert_output --partial "location: /admin"
  assert_output --partial "HTTP/2 302"
}
