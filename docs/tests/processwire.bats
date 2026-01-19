#!/usr/bin/env bats

setup() {
  PROJNAME=my-processwire-site
  load 'common-setup'
  _common_setup
}

# executed after each test
teardown() {
  _common_teardown
}

@test "processwire zipball with $(ddev --version)" {
  run mkdir -p my-processwire-site && cd my-processwire-site
  assert_success
  run _curl_github -LJOf https://github.com/processwire/processwire/archive/master.zip
  assert_success
  run unzip processwire-master.zip && rm -f processwire-master.zip && mv processwire-master/* . && mv processwire-master/.* . 2>/dev/null && rm -rf processwire-master
  assert_success
  run ddev config --project-type=php --webserver-type=apache-fpm
  assert_success
  DDEV_DEBUG=true run ddev start -y
  assert_success
  run bash -c "DDEV_DEBUG=true ddev launch"
  assert_output "FULLURL https://${PROJNAME}.ddev.site"
  assert_success

  # Diagnostic: show traefik config files in volume
  run docker exec ddev-router ls -la /mnt/ddev-global-cache/traefik/config/
  echo "# Traefik config files: $output"
  # Diagnostic: show traefik router API response (just router names)
  run docker exec ddev-router curl -s http://127.0.0.1:10999/api/http/routers
  echo "# Traefik routers: $output"

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
  run docker exec ddev-router ls -la /mnt/ddev-global-cache/traefik/config/
  echo "# Traefik config files: $output" >&3
  # Diagnostic: show traefik router API response (just router names)
  run docker exec ddev-router curl -s http://127.0.0.1:10999/api/http/routers
  echo "# Traefik routers: $output"
  # validate running project
  run curl -sfIv https://${PROJNAME}.ddev.site
  assert_output --partial "server: Apache"
  assert_success
  run curl -sfv https://${PROJNAME}.ddev.site
  assert_output --partial "This tool will guide you through the installation process."
  assert_success
}

@test "processwire composer with $(ddev --version)" {
  # mkdir my-processwire-site && cd my-processwire-site
  run mkdir -p my-processwire-site && cd my-processwire-site
  assert_success
  # ddev config --project-type=php --webserver-type=apache-fpm
  run ddev config --project-type=php --webserver-type=apache-fpm
  assert_success
  # ddev start -y
  DDEV_DEBUG=true run ddev start -y
  assert_success
  # ddev composer create-project "processwire/processwire:^3"
  run ddev composer create-project "processwire/processwire:^3"
  # ddev launch
  run bash -c "DDEV_DEBUG=true ddev launch"
  assert_output "FULLURL https://${PROJNAME}.ddev.site"
  assert_success

  # Diagnostic: show traefik config files in volume
  run docker exec ddev-router ls -la /mnt/ddev-global-cache/traefik/config/
  echo "# Traefik config files: $output"
  # Diagnostic: show traefik router API response (just router names)
  run docker exec ddev-router curl -s http://127.0.0.1:10999/api/http/routers
  echo "# Traefik routers: $output"

  # validate running project
  run curl -sfIv https://${PROJNAME}.ddev.site
  assert_output --partial "server: Apache"
  assert_output --partial "HTTP/2 200"
  assert_success
  run curl -sfv https://${PROJNAME}.ddev.site
  assert_output --partial "This tool will guide you through the installation process."
  assert_success
}
