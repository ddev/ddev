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
  PROJNAME=my-processwire-zip-site

  run -0 mkdir -p my-processwire-zip-site && cd my-processwire-zip-site
  run -0 _curl_github -LJOf https://github.com/processwire/processwire/archive/master.zip
  run -0 unzip processwire-master.zip
  rm -f processwire-master.zip
  run -0 cp -R processwire-master/. .
  rm -rf processwire-master
  run -0 ddev config --project-type=php --webserver-type=apache-fpm
  DDEV_DEBUG=true run -0 ddev start -y

  _extra_info

  DDEV_DEBUG=true run -0 ddev launch
  assert_output "FULLURL ${PRIMARY_URL}"

  # Diagnostic: show traefik config files in volume
#  run docker exec ddev-router ls -la /mnt/ddev-global-cache/traefik/config/
#  echo "# Traefik config files: $output"
  # Diagnostic: show traefik router API response (just router names)
#  run docker exec ddev-router curl -s http://127.0.0.1:10999/api/http/routers
#  echo "# Traefik routers: $output"

  # Diagnostic: show traefik config files in volume
  #run docker exec ddev-router ls -la /mnt/ddev-global-cache/traefik/config/
  #echo "# Traefik config files: $output" >&3
  # Diagnostic: show traefik router API response (just router names)
  #run docker exec ddev-router curl -s http://127.0.0.1:10999/api/http/routers
  #echo "# Traefik routers: $output"

  # validate running project
  run -0 curl -sSfI ${PRIMARY_URL}
  assert_output --partial "server: Apache"
  _assert_page_contains ${PRIMARY_URL} "This tool will guide you through the installation process."
}

@test "processwire composer with $(ddev --version)" {
  PROJNAME=my-processwire-composer-site

  run -0 mkdir -p my-processwire-composer-site && cd my-processwire-composer-site
  run -0 ddev config --project-type=php --webserver-type=apache-fpm
  DDEV_DEBUG=true run -0 ddev start -y

  _extra_info

  run ddev composer create-project "processwire/processwire:^3"
  DDEV_DEBUG=true run -0 ddev launch
  assert_output "FULLURL ${PRIMARY_URL}"

  # Diagnostic: show traefik config files in volume
#  run docker exec ddev-router ls -la /mnt/ddev-global-cache/traefik/config/
#  echo "# Traefik config files: $output"
#  # Diagnostic: show traefik router API response (just router names)
#  run docker exec ddev-router curl -s http://127.0.0.1:10999/api/http/routers
#  echo "# Traefik routers: $output"

  # validate running project
  run -0 curl -sSfI ${PRIMARY_URL}
  assert_output --partial "server: Apache"
  assert_output --partial "HTTP/2 200"
  _assert_page_contains ${PRIMARY_URL} "This tool will guide you through the installation process."
}
