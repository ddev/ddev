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
  run -0 mkdir -p ${NODEJS_SITENAME} && cd ${NODEJS_SITENAME}

  run -0 ddev config --project-type=generic --webserver-type=generic
  DDEV_DEBUG=true run -0 ddev start -y

  run -0 ddev npm install express

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
  run -0 ddev mutagen sync
  assert_file_exist .ddev/config.nodejs.yaml

  run -0 ddev exec curl -s -O https://raw.githubusercontent.com/ddev/test-nodejs/main/server.js

  DDEV_DEBUG=true run -0 ddev restart -y

  # web_extra_exposed_ports only takes effect after this restart, so the
  # generic webserver has no primary URL to report until now (see
  # GetPrimaryRouterHTTP(S)Port in pkg/ddevapp/ddevapp.go).
  _extra_info

  # Diagnostic: show traefik config files in volume
#  run docker exec ddev-router ls -la /mnt/ddev-global-cache/traefik/config/
#  echo "# Traefik config files: $output"
  # Diagnostic: show traefik router API response (just router names)
#  run docker exec ddev-router curl -s http://127.0.0.1:10999/api/http/routers
#  echo "# Traefik routers: $output"

  DDEV_DEBUG=true run -0 ddev launch
  assert_output "FULLURL ${PRIMARY_URL}"

  # Diagnostic: show traefik config files in volume
#  run docker exec ddev-router ls -la /mnt/ddev-global-cache/traefik/config/
#  echo "# Traefik config files: $output" >&3
  # Diagnostic: show traefik router API response (just router names)
#  run docker exec ddev-router curl -s http://127.0.0.1:10999/api/http/routers
#  echo "# Traefik routers: $output"

  # validate running project
  _assert_page_contains ${PRIMARY_URL} "DDEV experimental Node.js"
}
