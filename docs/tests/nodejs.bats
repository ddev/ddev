#!/usr/bin/env bats

setup() {
  PROJNAME=my-nodejs-site
  load 'common-setup'
  _common_setup
}

# executed after each test
teardown() {
  _common_teardown
}

@test "Node.js quickstart with $(ddev --version)" {
  NODEJS_SITENAME=${PROJNAME}
  run mkdir ${NODEJS_SITENAME} && cd ${NODEJS_SITENAME}
  assert_success

  run ddev config --project-type=generic --webserver-type=generic
  assert_success
  DDEV_DEBUG=true run ddev start -y
  assert_success

  run ddev npm install express
  assert_success

  cat <<EOF > .ddev/config.nodejs.yaml
web_extra_exposed_ports:
    - name: node-example
      container_port: 3000
      http_port: 80
      https_port: 443

web_extra_daemons:
    - name: "node-example"
      command: "node server.js"
      directory: /var/www/html
EOF
  run ddev mutagen sync
  assert_success
  assert_file_exist .ddev/config.nodejs.yaml

  run ddev exec curl -s -O https://raw.githubusercontent.com/ddev/test-nodejs/main/server.js
  assert_success

  DDEV_DEBUG=true run ddev restart -y
  assert_success

  # Diagnostic: show traefik config files in volume
  run docker exec ddev-router ls -la /mnt/ddev-global-cache/traefik/config/
  echo "# Traefik config files: $output"
  # Diagnostic: show traefik router API response (just router names)
  run docker exec ddev-router curl -s http://127.0.0.1:10999/api/http/routers
  echo "# Traefik routers: $output"

  # ddev launch
  run bash -c "DDEV_DEBUG=true ddev launch"
  assert_success
  assert_output "FULLURL https://${PROJNAME}.ddev.site"

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
  run curl -sfv https://${PROJNAME}.ddev.site
  assert_output --partial "DDEV experimental Node.js"
  assert_success
}
