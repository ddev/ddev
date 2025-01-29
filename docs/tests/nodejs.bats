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
  run ddev start -y
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
  assert_success

  run ddev exec curl -s -O https://raw.githubusercontent.com/ddev/test-nodejs/refs/heads/main/server.js
  assert_success

  run ddev restart -y
  assert_success

  # ddev launch
  run bash -c "DDEV_DEBUG=true ddev launch"
  assert_output "FULLURL https://${PROJNAME}.ddev.site"
  assert_success

  # validate running project
  run curl -sf https://${PROJNAME}.ddev.site
  assert_success
  assert_output --partial "DDEV experimental Node.js"
}
