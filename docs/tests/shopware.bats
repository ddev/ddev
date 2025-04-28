#!/usr/bin/env bats

setup() {
  PROJNAME=my-shopware-site
  load 'common-setup'
  _common_setup
}

# executed after each test
teardown() {
  _common_teardown
}

@test "Shopware Composer based quickstart with $(ddev --version)" {
  # mkdir ${PROJNAME} && cd ${PROJNAME}
  run mkdir ${PROJNAME} && cd ${PROJNAME}
  assert_success
  # ddev config --project-type=shopware6 --docroot=public
  run ddev config --project-type=shopware6 --docroot=public
  assert_success
  # ddev start -y
  run ddev start -y
  assert_success
  # ddev . echo x | ddev composer create-project shopware/production:^v6.5
  run ddev . echo x | ddev composer create-project shopware/production:^v6.5
  # ddev exec console system:install --basic-setup
  run ddev exec console system:install --basic-setup
  assert_success
  # ddev launch
  run bash -c "DDEV_DEBUG=true ddev launch /admin"
  assert_output --partial "FULLURL https://${PROJNAME}.ddev.site/admin"
  assert_success
  # validate running project
  run curl -sfI https://${PROJNAME}.ddev.site/admin
  assert_success
  assert_output --partial "sw-context-token,sw-access-key,sw-language-id,sw-version-id,sw-inheritance"
  assert_output --partial "HTTP/2 200"
}
