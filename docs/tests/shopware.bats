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
  run mkdir -p ${PROJNAME} && cd ${PROJNAME}
  assert_success

  run ddev config --project-type=shopware6 --docroot=public
  assert_success

  run ddev start -y
  assert_success

  _extra_info

  # shopware/production's composer.json now sets "php-http/discovery": false in
  # allow-plugins, so Composer no longer prompts to trust that plugin. The only
  # remaining interactive prompt is:
  #   Do you want to include Docker configuration from recipes?
  #   [x] No permanently, never ask again for this project
  # Answer `x`: DDEV provides the environment, not the recipe's Docker config.
  run bats_pipe printf "x\n" \| ddev composer create-project shopware/production
  assert_success
  assert_output --partial "Do you want to include Docker configuration from recipes?"
  assert_file_not_exist compose.yaml
  assert_file_not_exist compose.override.yaml

  run ddev exec console system:install --basic-setup
  assert_success

  DDEV_DEBUG=true run ddev launch /admin
  assert_output --partial "FULLURL ${PRIMARY_URL}/admin"
  assert_success

  # validate running project
  run curl -sfIv ${PRIMARY_URL}/admin
  assert_output --partial "sw-context-token,sw-access-key,sw-language-id,sw-version-id,sw-inheritance"
  assert_output --partial "HTTP/2 200"
  assert_success
}
