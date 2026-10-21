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
  PROJNAME=my-wp-cli-site

  run -0 mkdir -p my-wp-cli-site
  cd my-wp-cli-site

  run -0 ddev config --project-type=wordpress

  run -0 ddev start -y

  _extra_info

  run -0 _retry ddev wp core download

  run -0 ddev wp core install --url="$(ddev exec echo '$DDEV_PRIMARY_URL')" --title='My WordPress site' --admin_user=admin --admin_password=admin --admin_email=admin@example.com

  DDEV_DEBUG=true run -0 ddev launch
  assert_output "FULLURL ${PRIMARY_URL}"

  run -0 ddev wp config get WP_SITEURL
  assert_output "${PRIMARY_URL}/"

  # validate running project
  run -0 curl -sfI ${PRIMARY_URL}
  assert_line --regexp "link:.*${DDEV_HOSTNAME}.*rel=\"https://api\.w\.org/\""
  assert_output --partial "HTTP/2 200"

  # validate running project
  run -0 curl -sfI ${PRIMARY_URL}/wp-admin/
  assert_output --partial "location: ${PRIMARY_URL}/wp-login.php"
  assert_output --partial "HTTP/2 302"
}

@test "WordPress wp-cli based quickstart (different docroot) with $(ddev --version)" {
  PROJNAME=my-wp-docroot-site

  run -0 mkdir -p my-wp-docroot-site
  cd my-wp-docroot-site

  run -0 ddev config --project-type=wordpress --docroot=./web/wp

  run -0 ddev start -y

  _extra_info

  run -0 _retry ddev wp core download

  run -0 ddev wp core install --url="$(ddev exec echo '$DDEV_PRIMARY_URL')" --title='My WordPress site' --admin_user=admin --admin_password=admin --admin_email=admin@example.com

  run -0 ddev wp config get WP_SITEURL
  assert_output "${PRIMARY_URL}/"

  DDEV_DEBUG=true run -0 ddev launch
  assert_output "FULLURL ${PRIMARY_URL}"

  # validate running project
  run -0 curl -sfI ${PRIMARY_URL}
  assert_line --regexp "link:.*${DDEV_HOSTNAME}.*rel=\"https://api\.w\.org/\""
  assert_output --partial "HTTP/2 200"

  # validate running project
  run -0 curl -sfI ${PRIMARY_URL}/wp-admin/
  assert_output --partial "location: ${PRIMARY_URL}/wp-login.php"
  assert_output --partial "HTTP/2 302"

  # with no wp-cli.yml, WP-CLI gets the docroot, and "./web/wp" is normalized
  assert_file_not_exist wp-cli.yml

  run -0 ddev wp eval 'echo ABSPATH;'
  assert_output "/var/www/html/web/wp/"

  # a wp-cli.yml overrides the docroot, so WP-CLI reports the path it names
  printf 'path: nonexistent\n' > wp-cli.yml
  assert_file_exist wp-cli.yml

  run ddev wp eval 'echo ABSPATH;'
  assert_output --partial "The used path is: /var/www/html/nonexistent/"
}

@test "WordPress wp-cli based quickstart (subdirectory URL change) with $(ddev --version)" {
  PROJNAME=my-wp-subdir-site

  run -0 mkdir -p my-wp-subdir-site
  cd my-wp-subdir-site
  run -0 ddev config --project-type=wordpress
  run -0 ddev start -y

  _extra_info

  run -0 _retry ddev wp core download
  run -0 ddev wp core install --url="$(ddev exec echo '$DDEV_PRIMARY_URL')" --title='My WordPress site' --admin_user=admin --admin_password=admin --admin_email=admin@example.com

  run bash -c "
    mkdir wordpress &&
    mv wp-admin wp-content wp-includes wordpress/ &&
    mv index.php license.txt readme.html xmlrpc.php wordpress/ &&
    for file in wp-*.php; do
      case \"\$file\" in
        wp-config.php|wp-config-ddev.php) ;;
        *) mv \"\$file\" wordpress/ ;;
      esac
    done &&
    cp wordpress/index.php ./index.php &&
    perl -0pi -e 's|/wp-blog-header.php|/wordpress/wp-blog-header.php|' index.php
  "
  assert_success

  # Restart so DDEV rescans for wp-settings.php and regenerates wp-config-ddev.php with AbsPath="wordpress"
  run -0 ddev restart

  run -0 ddev wp config get WP_SITEURL
  assert_output "${PRIMARY_URL}/wordpress"

  # validate running project from the root URL
  run -0 curl -sfI ${PRIMARY_URL}
  assert_line --regexp "link:.*${DDEV_HOSTNAME}.*rel=\"https://api\.w\.org/\""
  assert_output --partial "HTTP/2 200"

  # validate admin/login URLs stay under the WordPress subdirectory
  run -0 curl -sfI ${PRIMARY_URL}/wordpress/wp-admin/
  assert_output --partial "location: ${PRIMARY_URL}/wordpress/wp-login.php"
  assert_output --partial "HTTP/2 302"
}

@test "WordPress git based quickstart with $(ddev --version)" {
  PROJNAME=my-wp-git-site

  PROJECT_GIT_URL=https://github.com/ddev/test-wordpress.git
  run -0 git clone ${PROJECT_GIT_URL} ${PROJNAME}

  cd ${PROJNAME} || exit 2

  run -0 ddev config --project-type=wordpress

  run -0 ddev start -y

  _extra_info

  run -0 ddev wp core install --url="$(ddev exec echo '$DDEV_PRIMARY_URL')" --title='My WordPress site' --admin_user=admin --admin_password=admin --admin_email=admin@example.com

  DDEV_DEBUG=true run -0 ddev launch
  assert_output "FULLURL ${PRIMARY_URL}"

  run -0 ddev wp config get WP_SITEURL
  assert_output "${PRIMARY_URL}/"

  # validate running project
  run -0 curl -sfI ${PRIMARY_URL}
  assert_line --regexp "link:.*${DDEV_HOSTNAME}.*rel=\"https://api\.w\.org/\""
  assert_output --partial "HTTP/2 200"

  # validate running project /wp-admin
  run -0 curl -sfI ${PRIMARY_URL}/wp-admin/
  assert_output --partial "location: ${PRIMARY_URL}/wp-login.php?redirect_to=${PRIMARY_URL_ENCODED}%2Fwp-admin%2F&reauth=1"
  assert_output --partial "HTTP/2 302"
}
