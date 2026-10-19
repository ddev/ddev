#!/usr/bin/env bats

setup() {
  PROJNAME=my-wp-bedrock-site
  load 'common-setup'
  _common_setup
}

# executed after each test
teardown() {
  _common_teardown
}

@test "WordPress Bedrock quickstart with $(ddev --version)" {
  PROJNAME=my-wp-bedrock-site

  run -0 mkdir -p my-wp-bedrock-site && cd my-wp-bedrock-site

  run -0 ddev config --project-type=wp-bedrock

  run -0 ddev start -y

  _extra_info

  run -0 _retry ddev composer create-project roots/bedrock

  run -0 ddev wp core install --url="${PRIMARY_URL}" --title='My Bedrock Site' --admin_user=admin --admin_password=admin --admin_email=admin@example.com

  DDEV_DEBUG=true run -0 ddev launch /wp/wp-admin/
  assert_output "FULLURL ${PRIMARY_URL}/wp/wp-admin/"

  # validate running project
  run -0 curl -sfI ${PRIMARY_URL}
  assert_line --regexp "link:.*${DDEV_HOSTNAME}.*rel=\"https://api\.w\.org/\""
  assert_output --partial "HTTP/2 200"

  # validate /wp/wp-admin/ redirects to login when unauthenticated
  run -0 curl -sfI ${PRIMARY_URL}/wp/wp-admin/
  assert_output --partial "location: ${PRIMARY_URL}/wp/wp-login.php"
  assert_output --partial "HTTP/2 302"

  # validate actual login: POST credentials, follow redirect to wp-admin, check for Dashboard
  COOKIE_JAR=$(mktemp)
  curl -sf -c "${COOKIE_JAR}" ${PRIMARY_URL}/wp/wp-login.php > /dev/null
  run -0 curl -sf -c "${COOKIE_JAR}" -b "${COOKIE_JAR}" \
    --data "log=admin&pwd=admin&wp-submit=Log+In&redirect_to=%2Fwp%2Fwp-admin%2F&testcookie=1" \
    --location ${PRIMARY_URL}/wp/wp-login.php
  assert_output --partial "Dashboard"
  rm -f "${COOKIE_JAR}"

  # Bedrock serves from web but keeps WordPress in web/wp, so its own wp-cli.yml
  # has to win over the docroot DDEV hands to WP-CLI
  assert_file_exist wp-cli.yml
  assert_file_contains wp-cli.yml "^path: web/wp$"

  run -0 ddev wp eval 'echo ABSPATH;'
  assert_output "/var/www/html/web/wp/"

  # an alias makes wp-cli.yml unparseable as strict YAML, which must not knock
  # WP-CLI back to the docroot
  cat >> wp-cli.yml <<'END'

@preprod:
  ssh: user@example.com:/var/www/preprod
  path: /var/www/preprod/web/wp
END

  run -0 ddev wp eval 'echo ABSPATH;'
  assert_output "/var/www/html/web/wp/"
}
