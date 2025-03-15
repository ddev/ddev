#!/usr/bin/env bats

setup() {
  PROJNAME=magento-lts
  load 'common-setup'
  _common_setup
}

# executed after each test
teardown() {
  _common_teardown
}

@test "OpenMage git based quickstart with $(ddev --version)" {
  # PROJECT_GIT_URL=https://github.com/OpenMage/magento-lts
  PROJECT_GIT_URL=https://github.com/OpenMage/magento-lts

  # git clone ${PROJECT_GIT_URL} ${PROJNAME}
  run git clone ${PROJECT_GIT_URL} ${PROJNAME}
  assert_success

  # cd magento-lts
  cd ${PROJNAME} || exit 2
  assert_success

  # ddev config --project-type=magento --php-version=8.1 --webserver-type=nginx-fpm
  run ddev config --project-type=magento --php-version=8.1 --webserver-type=nginx-fpm
  assert_success

  # ddev start -y
  run ddev start -y
  assert_success

  # ddev composer install
  run ddev composer install
  assert_success

  # ddev launch
  run bash -c "DDEV_DEBUG=true ddev launch"
  assert_output "FULLURL https://${PROJNAME}.ddev.site"
  assert_success

}
