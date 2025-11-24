#!/usr/bin/env bats

setup() {
  PROJNAME=my-frankenphp-site
  load 'common-setup'
  _common_setup
}

# executed after each test
teardown() {
  _common_teardown
}

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
      command: "frankenphp php-server --listen=0.0.0.0:80 --root=\"/var/www/html/${DDEV_DOCROOT:-}\" -v -a"
      directory: /var/www/html
web_extra_exposed_ports:
    - name: "frankenphp"
      container_port: 80
      http_port: 80
      https_port: 443
EOF
  assert_success

  cat <<'DOCKERFILEEND' >.ddev/web-build/Dockerfile.frankenphp
RUN curl -fsSL https://key.henderkes.com/static-php.gpg -o /usr/share/keyrings/static-php.gpg && \
    echo "deb [signed-by=/usr/share/keyrings/static-php.gpg] https://deb.henderkes.com/ stable main" > /etc/apt/sources.list.d/static-php.list
# Install FrankenPHP and extensions, see https://frankenphp.dev/docs/#deb-packages for details.
# You can find the list of available extensions at https://deb.henderkes.com/pool/main/p/
RUN (apt-get update || true) && DEBIAN_FRONTEND=noninteractive apt-get install -y -o Dpkg::Options::="--force-confnew" --no-install-recommends --no-install-suggests \
    frankenphp \
    php-zts-cli \
    php-zts-gd \
    php-zts-pdo-mysql
# Make sure that 'php' command uses the ZTS version of PHP
# and that the php.ini in use by FrankenPHP is the one from DDEV.
RUN ln -sf /usr/bin/php-zts /usr/local/bin/php && \
    ln -sf /etc/php/${DDEV_PHP_VERSION}/fpm/php.ini /etc/php-zts/php.ini
DOCKERFILEEND
  assert_success

  run ddev composer create-project drupal/recommended-project
  assert_success
  run ddev composer require drush/drush
  assert_success
  run ddev restart -y
  assert_success
  run ddev drush site:install demo_umami --account-name=admin --account-pass=admin -y
  assert_success

  # ddev launch
  DDEV_DEBUG=true run ddev launch
  assert_output "FULLURL https://${PROJNAME}.ddev.site"
  assert_success

  # validate running project
  run curl -sf -I https://${PROJNAME}.ddev.site
  assert_success
  assert_output --regexp "server: (Caddy|FrankenPHP)"
  assert_output --partial "x-generator: Drupal 11 (https://www.drupal.org)"
  assert_output --partial "HTTP/2 200"
}
