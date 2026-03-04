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
  _skip_if_embargoed "wordpress-cli"
  PROJNAME=my-wp-cli-site

  run mkdir -p my-wp-cli-site && cd my-wp-cli-site
  assert_success
  run ddev config --project-type=wordpress
  assert_success
  run ddev start -y
  assert_success
  run ddev wp core download
  assert_success
  run ddev wp core install --url='https://${PROJNAME}.ddev.site' --title='My WordPress site' --admin_user=admin --admin_password=admin --admin_email=admin@example.com
  assert_success
  DDEV_DEBUG=true run ddev launch
  assert_output "FULLURL https://${PROJNAME}.ddev.site"
  assert_success
  # validate running project
  run curl -sfI https://${PROJNAME}.ddev.site
  assert_line --regexp "link:.*${PROJNAME}\.ddev\.site.*rel=\"https://api\.w\.org/\""
  assert_output --partial "HTTP/2 200"
  assert_success
  # validate running project
  run curl -sfI https://${PROJNAME}.ddev.site/wp-admin/
  assert_output --partial "location: https://${PROJNAME}.ddev.site/wp-login.php"
  assert_output --partial "HTTP/2 302"
  assert_success
}

@test "WordPress wp-cli based quickstart (different docroot) with $(ddev --version)" {
  _skip_if_embargoed "wordpress-cli-docroot"
  PROJNAME=my-wp-docroot-site

  # mkdir -p my-wp-docroot-site && cd my-wp-docroot-site
  run mkdir -p my-wp-docroot-site && cd my-wp-docroot-site
  assert_success
  run ddev config --project-type=wordpress --docroot=web/wp
  assert_success
  run ddev start -y
  assert_success
  run ddev wp core download
  assert_success
  run ddev wp core install --url='https://${PROJNAME}.ddev.site' --title='My WordPress site' --admin_user=admin --admin_password=admin --admin_email=admin@example.com
  assert_success
  DDEV_DEBUG=true run ddev launch
  assert_output "FULLURL https://${PROJNAME}.ddev.site"
  assert_success
  # validate running project
  run curl -sfI https://${PROJNAME}.ddev.site
  assert_line --regexp "link:.*${PROJNAME}\.ddev\.site.*rel=\"https://api\.w\.org/\""
  assert_output --partial "HTTP/2 200"
  assert_success
  # validate running project
  run curl -sfI https://${PROJNAME}.ddev.site/wp-admin/
  assert_output --partial "location: https://${PROJNAME}.ddev.site/wp-login.php"
  assert_output --partial "HTTP/2 302"
  assert_success
}

@test "WordPress Bedrock based quickstart with $(ddev --version)" {
  _skip_if_embargoed "wordpress-bedrock"
  PROJNAME=my-wp-bedrock-site

  run mkdir -p my-wp-bedrock-site && cd my-wp-bedrock-site
  assert_success
  run ddev config --project-type=wordpress --docroot=web
  assert_success
  run ddev start -y
  assert_success
  run ddev composer create-project roots/bedrock
  assert_success
  run cp .env.example .env
  assert_success
  # Set database name to db in .env
  run ddev dotenv set .env --db-name=db --db-user=db --db-password=db --db-host=db --wp-home=https://${PROJNAME}.ddev.site --wp-siteurl=https://${PROJNAME}.ddev.site/wp
  assert_success
  run ddev wp core install --url='https://${PROJNAME}.ddev.site' --title='My WordPress site' --admin_user=admin --admin_password=admin --admin_email=admin@example.com
  assert_success
  DDEV_DEBUG=true run ddev launch
  assert_output "FULLURL https://${PROJNAME}.ddev.site"
  assert_success
  # validate running project
  run curl -sfI https://${PROJNAME}.ddev.site
  assert_line --regexp "link:.*${PROJNAME}\.ddev\.site.*rel=\"https://api\.w\.org/\""
  assert_output --partial "HTTP/2 200"
  assert_success
  # validate running project /wp-admin/
  # Some environments return 302 redirect to /wp/wp-admin/, others return 200
  run curl -sfI https://${PROJNAME}.ddev.site/wp-admin/
  assert_line --regexp "link:.*${PROJNAME}\.ddev\.site.*rel=\"https://api\.w\.org/\""
  assert_line --regexp 'HTTP/2 (200|302)'
  assert_success
}

@test "WordPress git based quickstart with $(ddev --version)" {
  _skip_if_embargoed "wordpress-git"
  PROJNAME=my-wp-git-site

  # PROJECT_GIT_URL=https://github.com/ddev/test-wordpress.git
  PROJECT_GIT_URL=https://github.com/ddev/test-wordpress.git
  run git clone ${PROJECT_GIT_URL} ${PROJNAME}
  assert_success
  cd ${PROJNAME} || exit 2
  assert_success
  run ddev config --project-type=wordpress
  assert_success
  run ddev start -y
  assert_success
  run ddev wp core install --url='https://${PROJNAME}.ddev.site' --title='My WordPress site' --admin_user=admin --admin_password=admin --admin_email=admin@example.com
  assert_success
  DDEV_DEBUG=true run ddev launch
  assert_output "FULLURL https://${PROJNAME}.ddev.site"
  assert_success
  # validate running project
  run curl -sfI https://${PROJNAME}.ddev.site
  assert_line --regexp "link:.*${PROJNAME}\.ddev\.site.*rel=\"https://api\.w\.org/\""
  assert_output --partial "HTTP/2 200"
  assert_success
  # validate running project /wp-admin
  run curl -sfI https://${PROJNAME}.ddev.site/wp-admin/
  assert_output --partial "location: https://${PROJNAME}.ddev.site/wp-login.php?redirect_to=https%3A%2F%2F${PROJNAME}.ddev.site%2Fwp-admin%2F&reauth=1"
  assert_output --partial "HTTP/2 302"
  assert_success
}
