#!/usr/bin/env bats

setup() {
  PROJNAME=my-moodle-site
  load 'common-setup'
  _common_setup
}

# executed after each test
teardown() {
  _common_teardown
}

@test "Moodle quickstart with $(ddev --version)" {
  run -0 mkdir -p ${PROJNAME} && cd ${PROJNAME}

  run -0 ddev config --docroot=public --webserver-type=apache-fpm

  run -0 ddev start -y

  _extra_info

  run -0 _retry ddev composer create-project moodle/moodle

  run -0 ddev exec 'php admin/cli/install.php --non-interactive --agree-license --wwwroot=$DDEV_PRIMARY_URL --dbtype=mariadb --dbhost=db --dbname=db --dbuser=db --dbpass=db --fullname="DDEV Moodle Demo" --shortname=Demo --adminemail=admin@example.com --adminpass=password'

  DDEV_DEBUG=true run -0 ddev launch /login
  assert_line "FULLURL ${PRIMARY_URL}/login"

  # validate running project
  run -0 curl -sSfI ${PRIMARY_URL}
  assert_output --partial "HTTP/2 303"
  assert_line --regexp "location: .*/login/index.php"
  assert_output --partial "set-cookie: MoodleSession="
  assert_output --partial "server: Apache"

  run -0 curl -sSfI ${PRIMARY_URL}/login/index.php
  assert_output --partial "HTTP/2 200"
  assert_output --partial "set-cookie: MoodleSession="
  assert_output --partial "server: Apache"
}
