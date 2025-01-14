#!/usr/bin/env bats

setup() {
  PROJNAME=my-wp-site
  load 'common-setup'
  _common_setup
}

# executed after each test
teardown() {
  _common_teardown
}

@test "WordPress wp-cli based quickstart with $(ddev --version)" {
  # mkdir my-wp-site && cd my-wp-site
  run mkdir my-wp-site && cd my-wp-site
  assert_success
  # ddev config --project-type=wordpress
  run ddev config --project-type=wordpress
  assert_success
  # ddev start
  run ddev start
  assert_success
  # ddev wp core download
  run ddev wp core download
  assert_success
  # ddev wp core install --url='$DDEV_PRIMARY_URL' --title='My WordPress site' --admin_user=admin --admin_password=admin --admin_email=admin@example.com
  run ddev wp core install --url='$DDEV_PRIMARY_URL' --title='My WordPress site' --admin_user=admin --admin_password=admin --admin_email=admin@example.com
  assert_success
  # ddev launch
  run bash -c "DDEV_DEBUG=true ddev launch"
  assert_output "FULLURL https://${PROJNAME}.ddev.site"
  assert_success
  # validate running project
  run curl -sfI https://${PROJNAME}.ddev.site
  assert_success
  assert_output --partial "link: <https://my-wp-site.ddev.site/wp-json/>; rel=\"https://api.w.org/\""
  assert_output --partial "HTTP/2 200"
  # validate running project
  run curl -sfI https://${PROJNAME}.ddev.site/wp-admin/
  assert_success
  assert_output --partial "location: https://my-wp-site.ddev.site/wp-login.php"
  assert_output --partial "HTTP/2 302"
}

@test "WordPress Bedrock based quickstart with $(ddev --version)" {
  # mkdir my-wp-site && cd my-wp-site
  run mkdir my-wp-site && cd my-wp-site
  assert_success
  # ddev config --project-type=wordpress --docroot=web
  run ddev config --project-type=wordpress --docroot=web
  assert_success
  # ddev start
  run ddev start
  assert_success
  # ddev composer create roots/bedrock
  run ddev composer create roots/bedrock
  assert_success
  # cp .env.example .env
  run cp .env.example .env
  assert_success
  # Set database name to db in .env
  run sed -i "s/DB_NAME='database_name'/DB_NAME='db'/g" .env
  assert_success
  # Set database user to db in .env
  run sed -i "s/DB_USER='database_user'/DB_USER='db'/g" .env
  assert_success
  # Set database password to db in .env
  run sed -i "s/DB_PASSWORD='database_password'/DB_PASSWORD='db'/g" .env
  assert_success
  # Set database host to db in .env
  run sed -i "s/# DB_HOST='localhost'/DB_HOST='db'/g" .env
  assert_success
  # Set WP_HOME to ${DDEV_PRIMARY_URL} in .env
  run sed -i "s/WP_HOME='http:\/\/example.com'/WP_HOME=\"\${DDEV_PRIMARY_URL}\"/g" .env
  assert_success
  # Set WP_SITEURL to ${WP_HOME}/wp in .env
  run sed -i "s/WP_SITEURL=\"\${WP_HOME}\/wp\"/WP_SITEURL=\"\${WP_HOME}\/wp\"/g" .env
  assert_success
  # ddev wp core install --url='$DDEV_PRIMARY_URL' --title='My WordPress site' --admin_user=admin --admin_password=admin --admin_email=admin@example.com
  run ddev wp core install --url='$DDEV_PRIMARY_URL' --title='My WordPress site' --admin_user=admin --admin_password=admin --admin_email=admin@example.com
  assert_success
  # ddev launch
  run bash -c "DDEV_DEBUG=true ddev launch"
  assert_output "FULLURL https://${PROJNAME}.ddev.site"
  assert_success
  # validate running project
  run curl -sfI https://${PROJNAME}.ddev.site
  assert_success
  assert_output --partial "link: <https://${PROJNAME}.ddev.site/wp-json/>; rel=\"https://api.w.org/\""
  assert_output --partial "HTTP/2 200"
  # validate running project
  run curl -sfI https://${PROJNAME}.ddev.site/wp-admin/
  assert_success
  assert_output --partial "location: https://${PROJNAME}.ddev.site/wp/wp-admin/"
  assert_output --partial "link: <https://${PROJNAME}.ddev.site/wp-json/>; rel=\"https://api.w.org/\""
  assert_output --partial "HTTP/2 302"
}
