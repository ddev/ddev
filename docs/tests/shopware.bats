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

  # 1. Do you trust "php-http/discovery" to execute code and wish to enable it now?
  #    (writes "allow-plugins" to composer.json) [y,n,d,?]
  # 2. Do you want to include Docker configuration from recipes?
  #    [x] No permanently, never ask again for this project
  run bats_pipe printf "y\nx\n" \| ddev composer create-project shopware/production
  assert_success
  assert_output --partial "execute code and wish to enable it now?"
  assert_output --partial "Do you want to include Docker configuration from recipes?"
  assert_file_not_exist compose.yaml
  assert_file_not_exist compose.override.yaml

  # TODO: Remove this after upstream makes a new release, see https://github.com/ddev/ddev/pull/8557
  run ddev composer require "twig/twig:<3.28"
  assert_success

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
