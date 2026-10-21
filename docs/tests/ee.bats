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
  PROJNAME=my-ee-zip-site

  run -0 mkdir -p ${PROJNAME}
  cd ${PROJNAME}

  # Download the latest version of Expression Engine
  run -0 _github_release_download "ExpressionEngine/ExpressionEngine" "^ExpressionEngine.*\\.zip$" "ee.zip"

  run -0 unzip ee.zip
  rm -f ee.zip

  run -0 ddev config --database=mysql:8.0

  DDEV_DEBUG=true run -0 ddev start -y

  _extra_info

  # validate ddev launch
  DDEV_DEBUG=true run -0 ddev launch /admin.php
  assert_output "FULLURL ${PRIMARY_URL}/admin.php"

  # Diagnostic: show traefik config files in volume
  # run docker exec ddev-router ls -la /mnt/ddev-global-cache/traefik/config/
  # echo "# Traefik config files:" >&3
  # printf '%s\n' "$output" | sed 's/^/# /' >&3

  # run docker exec ddev-router curl -sI http://ddev-my-ee-site-web:80/admin.php
  # echo "# curl from inside router:" >&3
  # printf '%s\n' "$output" | sed 's/^/# /' >&3
  # assert_line --partial "200 OK"

  # Diagnostic: show traefik router API response (just router names)
  # run docker exec ddev-router curl -s http://127.0.0.1:10999/api/http/routers
  # echo "# Traefik routers: \n $(echo $output | jq -r)" >&3

  # validate running project
  run -0 curl -sSfI ${PRIMARY_URL}/admin.php
  assert_line --partial "server: nginx"
  assert_line --partial "HTTP/2 200"
  run -0 curl -sf ${PRIMARY_URL}/admin.php
  assert_output --partial "<title>Install ExpressionEngine"
}

@test "Expression Engine Git Clone quickstart with $(ddev --version)" {
  PROJNAME=my-ee-git-site

  run -0 mkdir -p ${PROJNAME}
  cd ${PROJNAME}

  run -0 git clone --depth=1 https://github.com/ExpressionEngine/ExpressionEngine .

  run -0 ddev config --database=mysql:8.0

  DDEV_DEBUG=true run -0 ddev start -y

  _extra_info

  run -0 ddev composer install

  run -0 touch system/user/config/config.php

  echo "EE_INSTALL_MODE=TRUE" >.env.php
  run -0 ddev mutagen sync
  assert_file_exist .env.php

  # validate ddev launch
  DDEV_DEBUG=true run -0 ddev launch /admin.php
  assert_output "FULLURL ${PRIMARY_URL}/admin.php"

  # Diagnostic: show traefik config files in volume
  # run docker exec ddev-router ls -la /mnt/ddev-global-cache/traefik/config/
  # echo "# Traefik config files:" >&3
  # printf '%s\n' "$output" | sed 's/^/# /' >&3

  # Diagnostic: show traefik router API response (just router names)
  # run docker exec ddev-router curl -s http://127.0.0.1:10999/api/http/routers
  # echo "# Traefik routers:" >&3
  # printf '%s\n' "$(echo "$output" | jq -r)" | sed 's/^/# /' >&3

  # validate running project
  run -0 curl -sSfI ${PRIMARY_URL}/admin.php
  assert_line --partial "server: nginx"
  assert_line --partial "HTTP/2 200"
  run -0 curl -sf ${PRIMARY_URL}/admin.php
  assert_output --partial "<title>Install ExpressionEngine"
}
