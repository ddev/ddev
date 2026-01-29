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
  _skip_if_embargoed "moodle-composer"

  # mkdir ${PROJNAME} && cd ${PROJNAME}
  run mkdir ${PROJNAME} && cd ${PROJNAME}
  assert_success
  run ddev config --docroot=public --webserver-type=apache-fpm
  assert_success
  # ddev start -y
  run ddev start -y
  assert_success
  # ddev composer create-project moodle/moodle
  run ddev composer create-project moodle/moodle
  assert_success
  run ddev exec 'php admin/cli/install.php --non-interactive --agree-license --wwwroot=$DDEV_PRIMARY_URL --dbtype=mariadb --dbhost=db --dbname=db --dbuser=db --dbpass=db --fullname="DDEV Moodle Demo" --shortname=Demo --adminpass=password'
  assert_success
  # ddev launch
  DDEV_DEBUG=true run ddev launch /login
  assert_output "FULLURL https://${PROJNAME}.ddev.site/login"
  assert_success
  # validate running project
  run curl -sfIv https://${PROJNAME}.ddev.site
  assert_output --partial "HTTP/2 200"
  assert_output --partial "set-cookie: MoodleSession="
  assert_success
}
