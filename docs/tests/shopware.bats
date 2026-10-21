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
  run -0 mkdir -p ${PROJNAME}
  cd ${PROJNAME}

  run -0 ddev config --project-type=shopware6 --docroot=public

  run -0 ddev start -y

  _extra_info

  # shopware/production's composer.json now sets "php-http/discovery": false in
  # allow-plugins, so Composer no longer prompts to trust that plugin. The only
  # remaining interactive prompt is:
  #   Do you want to include Docker configuration from recipes?
  #   [x] No permanently, never ask again for this project
  # Answer `x`: DDEV provides the environment, not the recipe's Docker config.
  run -0 bats_pipe printf "x\n" \| ddev composer create-project shopware/production
  assert_output --partial "Do you want to include Docker configuration from recipes?"
  assert_file_not_exist compose.yaml
  assert_file_not_exist compose.override.yaml

  run -0 ddev console system:install --basic-setup

  # --- shopware-cli watcher tooling bundled with the shopware6 project type ---

  # The shopware-cli binary is baked into the web image (downloaded from its
  # GitHub release), and the wrapper execs it by absolute path so it can't recurse
  # into itself via $PATH. A working --version proves both the image build and the
  # wrapper. The version floats with the "latest" release, so we don't assert a
  # literal; we just require a version-shaped string.
  run -0 ddev shopware-cli --version
  assert_output --regexp '[0-9]+\.[0-9]+\.[0-9]+'

  # The watcher commands are bundled and gated to shopware6: `ddev help` only
  # resolves them if ProjectTypes let them register for this project type.
  run -0 ddev help admin-watch
  run -0 ddev help storefront-watch

  # admin-watch validates its args (and thus actually runs in-container) without
  # starting the long-running Vite server.
  run ddev admin-watch
  assert_failure
  assert_output --partial "Usage: ddev admin-watch"

  DDEV_DEBUG=true run -0 ddev launch /admin
  assert_output --partial "FULLURL ${PRIMARY_URL}/admin"

  # validate running project
  run -0 curl -sSfI ${PRIMARY_URL}/admin
  assert_output --partial "sw-context-token,sw-access-key,sw-language-id,sw-version-id,sw-inheritance"
  assert_output --partial "HTTP/2 200"
}
