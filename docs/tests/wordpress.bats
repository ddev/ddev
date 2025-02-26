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
  # ddev start -y
  run ddev start -y
  assert_success
  # ddev wp core download
  run ddev wp core download
  assert_success
  # ddev wp core install --url='https://${PROJNAME}.ddev.site' --title='My WordPress site' --admin_user=admin --admin_password=admin --admin_email=admin@example.com
  run ddev wp core install --url='https://${PROJNAME}.ddev.site' --title='My WordPress site' --admin_user=admin --admin_password=admin --admin_email=admin@example.com
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
  # ddev start -y
  run ddev start -y
  assert_success
  # ddev composer create-project roots/bedrock .
  run ddev composer create-project roots/bedrock .
  assert_success
  # cp .env.example .env
  run cp .env.example .env
  assert_success
  # Set database name to db in .env
  # ddev dotenv set .env --db-name=db --db-user=db --db-password=db --db-host=db --wp-home=https://${PROJNAME}.ddev.site --wp-siteurl=https://${PROJNAME}.ddev.site/wp
  run ddev dotenv set .env --db-name=db --db-user=db --db-password=db --db-host=db --wp-home=https://${PROJNAME}.ddev.site --wp-siteurl=https://${PROJNAME}.ddev.site/wp
  assert_success
  # ddev wp core install --url='https://${PROJNAME}.ddev.site' --title='My WordPress site' --admin_user=admin --admin_password=admin --admin_email=admin@example.com
  run ddev wp core install --url='https://${PROJNAME}.ddev.site' --title='My WordPress site' --admin_user=admin --admin_password=admin --admin_email=admin@example.com
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

@test "WordPress git based quickstart with $(ddev --version)" {
  # PROJECT_GIT_URL=https://github.com/ddev/test-wordpress.git
  PROJECT_GIT_URL=https://github.com/ddev/test-wordpress.git
  # git clone ${PROJECT_GIT_URL} ${PROJNAME}
  run git clone ${PROJECT_GIT_URL} ${PROJNAME}
  assert_success
  # cd my-typo3-site
  cd ${PROJNAME} || exit 2
  assert_success
  # ddev config --project-type=wordpress
  run ddev config --project-type=wordpress
  assert_success
  # ddev start -y
  run ddev start -y
  assert_success
  # ddev wp core install --url='https://${PROJNAME}.ddev.site' --title='My WordPress site' --admin_user=admin --admin_password=admin --admin_email=admin@example.com
  run ddev wp core install --url='https://${PROJNAME}.ddev.site' --title='My WordPress site' --admin_user=admin --admin_password=admin --admin_email=admin@example.com
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
