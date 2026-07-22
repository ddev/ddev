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

  run ddev exec console system:install --basic-setup
  assert_success

  # --- shopware-cli watcher tooling bundled with the shopware6 project type ---

  # The shopware-cli binary is baked into the web image (downloaded from its
  # GitHub release), and the wrapper execs it by absolute path so it can't recurse
  # into itself via $PATH. A working --version proves both the image build and the
  # wrapper. The version floats with the "latest" release, so we don't assert a
  # literal; we just require a version-shaped string.
  run ddev shopware-cli --version
  assert_success
  assert_output --regexp '[0-9]+\.[0-9]+\.[0-9]+'

  # The watcher commands are bundled and gated to shopware6: `ddev help` only
  # resolves them if ProjectTypes let them register for this project type.
  run ddev help admin-watch
  assert_success
  run ddev help storefront-watch
  assert_success

  # admin-watch validates its args (and thus actually runs in-container) without
  # starting the long-running Vite server.
  run ddev admin-watch
  assert_failure
  assert_output --partial "Usage: ddev admin-watch"

  DDEV_DEBUG=true run ddev launch /admin
  assert_output --partial "FULLURL https://${PROJNAME}.ddev.site/admin"
  assert_success

  # validate running project
  run curl -sfIv https://${PROJNAME}.ddev.site/admin
  assert_output --partial "sw-context-token,sw-access-key,sw-language-id,sw-version-id,sw-inheritance"
  assert_output --partial "HTTP/2 200"
  assert_success
}
