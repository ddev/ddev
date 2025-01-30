#!/usr/bin/env bats

setup() {
  PROJNAME=my-frankenphp-site
  load 'common-setup'
  _common_setup
}

# executed after each test
#teardown() {
#  _common_teardown
#}

@test "FrankenPHP Drupal 11 quickstart with $(ddev --version)" {
  FRANKENPHP_SITENAME=${PROJNAME}
  run mkdir ${FRANKENPHP_SITENAME} && cd ${FRANKENPHP_SITENAME}
  assert_success

  run ddev config --project-type=drupal11 --webserver-type=generic --docroot=web --php-version=8.4
  assert_success
  ddev start -y
  assert_success

  cat <<'EOF' > .ddev/config.frankenphp.yaml
web_extra_daemons:
    - name: "frankenphp"
      command: "frankenphp php-server --listen 0.0.0.0:80 --root ${DDEV_DOCROOT} -v -a"
      directory: /var/www/html
web_extra_exposed_ports:
    - name: "frankenphp"
      container_port: 80
      http_port: 80
      https_port: 443
EOF
  assert_success

  cat <<'DOCKERFILEEND' >.ddev/web-build/Dockerfile.frankenphp
RUN curl -s https://frankenphp.dev/install.sh | sh
RUN mv frankenphp /usr/local/bin/
RUN mkdir -p /usr/local/etc && ln -s /etc/php/${DDEV_PHP_VERSION}/fpm /usr/local/etc/php
DOCKERFILEEND
  assert_success

  run ddev composer create drupal/recommended-project
  assert_success
  run ddev composer require drush/drush
  assert_success
  run ddev restart -y
  assert_success
  run ddev drush site:install demo_umami --account-name=admin --account-pass=admin -y
  assert_success

  # ddev launch
  run bash -c "DDEV_DEBUG=true ddev launch"
  assert_output "FULLURL https://${PROJNAME}.ddev.site"
  assert_success

  # validate running project
  run curl -sf -I https://${PROJNAME}.ddev.site
  assert_success
  assert_output --partial "server: Caddy"
  assert_output --partial "x-generator: Drupal 11 (https://www.drupal.org)"
  assert_output --partial "HTTP/2 200"
}
