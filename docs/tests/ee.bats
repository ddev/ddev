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
  _skip_if_embargoed "ee-zip"

  run mkdir ${PROJNAME} && cd ${PROJNAME}
  assert_success

  # Download the latest version of Expression Engine
  run _github_release_download "ExpressionEngine/ExpressionEngine" "^ExpressionEngine.*\\.zip$" "ee.zip"
  assert_success

  run unzip ee.zip && rm -f ee.zip
  assert_success

  run ddev config --database=mysql:8.0
  assert_success

  DDEV_DEBUG=true run ddev start -y
  assert_success

  # validate ddev launch
  DDEV_DEBUG=true run ddev launch /admin.php
  assert_output "FULLURL https://${PROJNAME}.ddev.site/admin.php"
  assert_success

  run bash -c '
  docker ps -q \
    --filter "label=com.ddev.platform=ddev" \
    --filter "label=com.docker.compose.service=web" \
    --filter "label=com.docker.compose.oneoff=False" |
  xargs -r docker inspect --format "{{.Name}} {{.State.Status}} {{if .State.Health}}{{.State.Health.Status}}{{else}}no-health{{end}}"
'
  assert_output --partial "${PROJNAME}-web running healthy"
  assert_success
  echo "# Existing containers: $output" >&3

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
  run curl -sfIv https://${PROJNAME}.ddev.site/admin.php
  assert_line --partial "server: nginx"
  assert_line --partial "HTTP/2 200"
  assert_success
  run curl -sf https://${PROJNAME}.ddev.site/admin.php
  assert_output --partial "<title>Install ExpressionEngine"
  assert_success
}

@test "Expression Engine Git Clone quickstart with $(ddev --version)" {
  _skip_if_embargoed "ee-git"

  run mkdir ${PROJNAME} && cd ${PROJNAME}
  assert_success

  run git clone --depth=1 https://github.com/ExpressionEngine/ExpressionEngine .
  assert_success

  run ddev config --database=mysql:8.0
  assert_success

  DDEV_DEBUG=true run ddev start -y
  assert_success

  run ddev composer install
  assert_success

  run touch system/user/config/config.php
  assert_success

  echo "EE_INSTALL_MODE=TRUE" >.env.php
  run ddev mutagen sync
  assert_success
  assert_file_exist .env.php

  # validate ddev launch
  DDEV_DEBUG=true run ddev launch /admin.php
  assert_output "FULLURL https://${PROJNAME}.ddev.site/admin.php"
  assert_success

  # Diagnostic: show traefik config files in volume
  # run docker exec ddev-router ls -la /mnt/ddev-global-cache/traefik/config/
  # echo "# Traefik config files:" >&3
  # printf '%s\n' "$output" | sed 's/^/# /' >&3

  # Diagnostic: show traefik router API response (just router names)
  # run docker exec ddev-router curl -s http://127.0.0.1:10999/api/http/routers
  # echo "# Traefik routers:" >&3
  # printf '%s\n' "$(echo "$output" | jq -r)" | sed 's/^/# /' >&3

  # validate running project
  run curl -sfIv https://${PROJNAME}.ddev.site/admin.php
  assert_line --partial "server: nginx"
  assert_line --partial "HTTP/2 200"
  assert_success
  run curl -sf https://${PROJNAME}.ddev.site/admin.php
  assert_output --partial "<title>Install ExpressionEngine"
  assert_success
}
