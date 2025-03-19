#!/usr/bin/env bats

setup() {
  PROJNAME=my-openmage-site
  load 'common-setup'
  _common_setup
}

# executed after each test
teardown() {
  _common_teardown
}

@test "OpenMage git based quickstart with $(ddev --version)" {
  run mkdir ${PROJNAME} && cd ${PROJNAME}
  assert_success

  run git clone --depth=1 https://github.com/OpenMage/magento-lts .
  assert_success

  run ddev config --project-type=magento --php-version=8.1 --web-environment-add=MAGE_IS_DEVELOPER_MODE=1
  assert_success

  run ddev start -y
  assert_success

  run ddev composer install
  assert_success

  # Silent OpenMage install with sample data
  run ddev openmage-install -q
  assert_success

  run bash -c "DDEV_DEBUG=true ddev launch"
  assert_output "FULLURL https://${PROJNAME}.ddev.site"
  assert_success

  # validate running project
  run curl -sfI https://${PROJNAME}.ddev.site
  assert_success
  assert_output --partial "server: nginx"
  assert_output --partial "HTTP/2 200"
  assert_output --partial "set-cookie: om_frontend"

  # Check if the frontend is working
  run curl -sf https://${PROJNAME}.ddev.site
  assert_success
  assert_output --partial "<title>Madison Island</title>"
  assert_output --partial "<meta name=\"keywords\" content=\"Magento, Varien, E-commerce\" />"

  # Check if the admin is working
  run curl -sf https://${PROJNAME}.ddev.site/index.php/admin/
  assert_success
  assert_output --partial "Log into OpenMage Admin Page"
}
