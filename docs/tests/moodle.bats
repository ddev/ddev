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
  # mkdir ${PROJNAME} && cd ${PROJNAME}
  run mkdir ${PROJNAME} && cd ${PROJNAME}
  assert_success
  # ddev config --composer-root=public --docroot=public --webserver-type=apache-fpm
  run ddev config --composer-root=public --docroot=public --webserver-type=apache-fpm
  assert_success
  # ddev start -y
  run ddev start -y
  assert_success
  # ddev composer create moodle/moodle
  run ddev composer create moodle/moodle
  assert_success
  # ddev exec 'php public/admin/cli/install.php --non-interactive --agree-license --wwwroot=$DDEV_PRIMARY_URL --dbtype=mariadb --dbhost=db --dbname=db --dbuser=db --dbpass=db --fullname="DDEV Moodle Demo" --shortname=Demo --adminpass=password'
  run ddev exec 'php public/admin/cli/install.php --non-interactive --agree-license --wwwroot=$DDEV_PRIMARY_URL --dbtype=mariadb --dbhost=db --dbname=db --dbuser=db --dbpass=db --fullname="DDEV Moodle Demo" --shortname=Demo --adminpass=password'
  assert_success
  # ddev launch
  run bash -c "DDEV_DEBUG=true ddev launch /login"
  assert_output "FULLURL https://${PROJNAME}.ddev.site/login"
  assert_success
  # validate running project
  run curl -sfI https://${PROJNAME}.ddev.site
  assert_success
  assert_output --partial "HTTP/2 200"
  assert_output --partial "set-cookie: MoodleSession="
}
