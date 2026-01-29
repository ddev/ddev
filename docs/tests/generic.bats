#!/usr/bin/env bats

setup() {
  PROJNAME=my-generic-site
  load 'common-setup'
  _common_setup
}

# executed after each test
teardown() {
  _common_teardown
}

@test "Generic PHP built-in server quickstart with $(ddev --version)" {
  _skip_if_embargoed "generic-php"

  GENERIC_SITENAME=${PROJNAME}
  run mkdir ${GENERIC_SITENAME} && cd ${GENERIC_SITENAME}
  assert_success

  run ddev config --project-type=php
  assert_success

  echo "<?php phpinfo(); ?>" > index.php
  run ddev mutagen sync
  assert_success
  assert_file_exist index.php

  cat <<'EOF' > .ddev/config.php-server.yaml
webserver_type: generic
web_extra_daemons:
    - name: "php-server"
      command: "php -S 0.0.0.0:8000 -t \"${DDEV_DOCROOT:-.}\""
      directory: /var/www/html
web_extra_exposed_ports:
    - name: "php-server"
      container_port: 8000
      http_port: 80
      https_port: 443
EOF
  run ddev mutagen sync
  assert_success
  assert_file_exist .ddev/config.php-server.yaml

  run ddev start -y
  assert_success

  # ddev launch
  DDEV_DEBUG=true run ddev launch
  assert_output "FULLURL https://${PROJNAME}.ddev.site"
  assert_success

  # validate running project
  run curl -sfI https://${PROJNAME}.ddev.site
  assert_output --partial "x-powered-by: PHP"
  assert_success

  run curl -sf https://${PROJNAME}.ddev.site
  assert_output --partial "Built-in HTTP server"
  assert_success
}
