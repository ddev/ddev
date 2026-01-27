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
  _skip_test_if_needed "shopware-composer"

  run mkdir ${PROJNAME} && cd ${PROJNAME}
  assert_success

  run ddev config --project-type=shopware6 --docroot=public
  assert_success

  run ddev start -y
  assert_success

  # Do you want to include Docker configuration from recipes?
  # [x] No permanently, never ask again for this project
  run bats_pipe echo x \| ddev composer create-project shopware/production
  assert_success
  assert_output --partial "Do you want to include Docker configuration from recipes?"
  assert_file_not_exist compose.yaml
  assert_file_not_exist compose.override.yaml

  run ddev exec console system:install --basic-setup
  assert_success

  DDEV_DEBUG=true run ddev launch /admin
  assert_output --partial "FULLURL https://${PROJNAME}.ddev.site/admin"
  assert_success

  # validate running project
  run curl -sfIv https://${PROJNAME}.ddev.site/admin
  assert_output --partial "sw-context-token,sw-access-key,sw-language-id,sw-version-id,sw-inheritance"
  assert_output --partial "HTTP/2 200"
  assert_success
}
