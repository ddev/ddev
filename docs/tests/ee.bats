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
  run _github_release_download "ExpressionEngine/ExpressionEngine" "^ExpressionEngine.*\\.zip$" "ee.zip"
  assert_success

  # unzip ee.zip && rm -f ee.zip
  run unzip ee.zip && rm -f ee.zip
  assert_success

  # ddev config --database=mysql:8.0
  run ddev config --database=mysql:8.0
  assert_success

  # ddev start -y
  DDEV_DEBUG=true run ddev start -y
  assert_success

  # validate ddev launch
  run bash -c "DDEV_DEBUG=true ddev launch /admin.php"
  assert_output "FULLURL https://${PROJNAME}.ddev.site/admin.php"
  assert_success

  run docker ps \
    --filter "label=com.ddev.platform=ddev" \
    --filter "label=com.docker.compose.service=web" \
    --filter "label=com.docker.compose.oneoff=False" \
    --format 'table {{.ID}}\t{{.Names}}\t{{.Status}}'
  echo "# Existing containers: $output" >&3

  # Diagnostic: show traefik config files in volume
  run docker exec ddev-router ls -la /mnt/ddev-global-cache/traefik/config/
  echo "# Traefik config files: $output" >&3
  # Diagnostic: show traefik router API response (just router names)
  run docker exec ddev-router curl -s http://127.0.0.1:10999/api/http/routers
  echo "# Traefik routers: $output"
  # validate running project
  run curl -sfIv https://${PROJNAME}.ddev.site/admin.php
  assert_output --partial "server: nginx"
  assert_output --partial "HTTP/2 200"
  assert_success
  run curl -sfv https://${PROJNAME}.ddev.site/admin.php
  assert_output --partial "<title>Install ExpressionEngine"
  assert_success
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
  DDEV_DEBUG=true run ddev start -y
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

  # Diagnostic: show traefik config files in volume
  run docker exec ddev-router ls -la /mnt/ddev-global-cache/traefik/config/
  echo "# Traefik config files: $output"
  # Diagnostic: show traefik router API response (just router names)
  run docker exec ddev-router curl -s http://127.0.0.1:10999/api/http/routers
  echo "# Traefik routers: $output"

  # validate running project
  run curl -sfIv https://${PROJNAME}.ddev.site/admin.php
  assert_output --partial "server: nginx"
  assert_output --partial "HTTP/2 200"
  assert_success
  run curl -sfv https://${PROJNAME}.ddev.site/admin.php
  assert_output --partial "<title>Install ExpressionEngine"
  assert_success
}
