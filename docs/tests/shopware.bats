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
  run mkdir ${PROJNAME} && cd ${PROJNAME}
  assert_success
  run ddev config --project-type=shopware6 --docroot=public
  assert_success
  run ddev start -y
  assert_success
  run bats_pipe echo x \| ddev composer create-project shopware/production
  assert_success
  run ddev exec console system:install --basic-setup
  assert_success
  run bash -c "DDEV_DEBUG=true ddev launch /admin"
  assert_output --partial "FULLURL https://${PROJNAME}.ddev.site/admin"
  assert_success
  # validate running project
  run curl -sfI https://${PROJNAME}.ddev.site/admin
  assert_success
  assert_output --partial "sw-context-token,sw-access-key,sw-language-id,sw-version-id,sw-inheritance"
  assert_output --partial "HTTP/2 200"
}
