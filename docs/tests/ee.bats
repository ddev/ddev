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

  # Download the latest version of the ExpressionEngine zip file from GitHub
  LATEST_RELEASE=$(curl -fsSL -H 'Accept: application/json' https://github.com/ExpressionEngine/ExpressionEngine/releases/latest || (printf "${RED}Failed to get find latest release${RESET}\n" >/dev/stderr && exit 107))
  LATEST_VERSION=$(echo $LATEST_RELEASE | sed -e 's/.*"tag_name":"\([^"]*\)".*/\1/')
  VERSION=$LATEST_VERSION
  if [ $# -ge 1 ]; then
    VERSION=$1
  fi
  run curl -LJO https://github.com/ExpressionEngine/ExpressionEngine/releases/download/$VERSION/ExpressionEngine$VERSION.zip
  assert_success

  # Unzip and move the extracted assets to the parent root
  run unzip -o ./ExpressionEngine$VERSION.zip && rm -f ExpressionEngine$VERSION.zip && shopt -s dotglob 2>/dev/null || setopt dotglob && mv -f ./ExpressionEngine$VERSION/* . && shopt -u dotglob 2>/dev/null || unsetopt dotglob && rm -rf ExpressionEngine$VERSION
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
  run echo "EE_INSTALL_MODE=TRUE" >.env.php
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
