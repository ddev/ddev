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
  PROJNAME=my-grav-composer-site

  run -0 mkdir -p ${PROJNAME} && cd ${PROJNAME}

  run -0 ddev config --php-version=8.3 --omit-containers=db

  run -0 ddev start -y

  _extra_info

  run -0 _retry ddev composer create-project getgrav/grav

  # Grav 2.0 dropped support for the classic admin plugin; admin2 is its replacement.
  run -0 ddev exec gpm install admin2 -y

  DDEV_DEBUG=true run -0 ddev launch
  assert_output "FULLURL ${PRIMARY_URL}"

  # validate running project
  run -0 curl -sSfI ${PRIMARY_URL}
  assert_output --partial "set-cookie: grav-site-"
  assert_output --partial "location: /admin"
  assert_output --partial "HTTP/2 302"
}


@test "Grav Git Clone quickstart with $(ddev --version)" {
  PROJNAME=my-grav-git-site

  run -0 mkdir -p my-grav-git-site && cd my-grav-git-site

  run -0 git clone -b master https://github.com/getgrav/grav.git .

  run -0 ddev config --php-version=8.3 --omit-containers=db

  run -0 ddev start -y

  _extra_info

  run -0 ddev composer install

  run -0 ddev exec grav install

  run -0 ddev exec gpm install admin2 -y

  DDEV_DEBUG=true run -0 ddev launch
  assert_output "FULLURL ${PRIMARY_URL}"

  # validate running project
  run -0 curl -sSfI ${PRIMARY_URL}
  assert_output --partial "set-cookie: grav-site-"
  assert_output --partial "location: /admin"
  assert_output --partial "HTTP/2 302"
}
