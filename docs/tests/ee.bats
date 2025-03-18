#!/usr/bin/env bats

setup() {
  PROJNAME=my-ee-site
  load 'common-setup'
  _common_setup
}

# executed after each test
teardown() {
  _common_teardown
}

@test "Expression Engine Zip File Download quickstart with $(ddev --version)" {
  # mkdir ${PROJNAME} && cd ${PROJNAME}
  run mkdir ${PROJNAME} && cd ${PROJNAME}
  assert_success

  # Download the latest version of Expression Engine
  run curl -o ee.zip -L $(curl -sL https://api.github.com/repos/ExpressionEngine/ExpressionEngine/releases/latest | docker run -i --rm ddev/ddev-utilities jq -r '.assets | map(select(.name | test("^ExpressionEngine.*\\.zip$")))[0].browser_download_url')
  assert_success

  # unzip ee.zip && rm -f ee.zip
  run unzip ee.zip && rm -f ee.zip
  assert_success

  # ddev config --database=mysql:8.0
  run ddev config --database=mysql:8.0
  assert_success

  # ddev start -y
  run ddev start -y
  assert_success

  # validate ddev launch
  run bash -c "DDEV_DEBUG=true ddev launch /admin.php"
  assert_output "FULLURL https://${PROJNAME}.ddev.site/admin.php"
  assert_success
    # validate running project
  run curl -sfI https://${PROJNAME}.ddev.site/admin.php
  assert_success
  assert_output --partial "server: nginx"
  assert_output --partial "HTTP/2 200"
  run curl -sf https://${PROJNAME}.ddev.site/admin.php
  assert_success
  assert_output --partial "<title>Install ExpressionEngine"
}

@test "Expression Engine Git Clone quickstart with $(ddev --version)" {
  # mkdir ${PROJNAME} && cd ${PROJNAME}
  run mkdir ${PROJNAME} && cd ${PROJNAME}
  assert_success

  # git clone --depth=1 https://github.com/ExpressionEngine/ExpressionEngine my-ee-site
  run git clone --depth=1 https://github.com/ExpressionEngine/ExpressionEngine .
  assert_success

  # ddev config --auto
  run ddev config --database=mysql:8.0
  assert_success

  # ddev start -y
  run ddev start -y
  assert_success

  # ddev composer install
  run ddev composer install
  assert_success

  # touch system/user/config/config.php
  run touch system/user/config/config.php
  assert_success

  # echo "EE_INSTALL_MODE=TRUE" >.env.php
  echo "EE_INSTALL_MODE=TRUE" >.env.php
  ddev mutagen sync
  assert_file_exist .env.php

  # validate ddev launch
  run bash -c "DDEV_DEBUG=true ddev launch /admin.php"
  assert_output "FULLURL https://${PROJNAME}.ddev.site/admin.php"
  assert_success
  # validate running project
  run curl -sfI https://${PROJNAME}.ddev.site/admin.php
  assert_success
  assert_output --partial "server: nginx"
  assert_output --partial "HTTP/2 200"
  run curl -sf https://${PROJNAME}.ddev.site/admin.php
  assert_success
  assert_output --partial "<title>Install ExpressionEngine"
}
