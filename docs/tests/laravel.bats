#!/usr/bin/env bats

setup() {
  PROJNAME=my-laravel-site
  load 'common-setup'
  _common_setup
}

# executed after each test
teardown() {
  _common_teardown
}

@test "Laravel composer based quickstart with $(ddev --version)" {
  _skip_test_if_needed "laravel-composer"

  run mkdir my-laravel-site && cd my-laravel-site
  assert_success

  run ddev config --project-type=laravel --docroot=public
  assert_success

  run ddev start -y
  assert_success

  run ddev composer create-project "laravel/laravel:^12"
  assert_success

  DDEV_DEBUG=true run ddev launch
  assert_output --partial "FULLURL https://${PROJNAME}.ddev.site"
  assert_success

  # validate running project
  run curl -sfIv https://${PROJNAME}.ddev.site
  assert_output --partial "HTTP/2 200"
  assert_success

  run curl -sfv https://${PROJNAME}.ddev.site
  assert_output --partial "Laravel"
  assert_success

  # check used database
  run ddev artisan about
  assert_success
  assert_output --partial "mariadb"
}

@test "Laravel composer (SQLite) based quickstart with $(ddev --version)" {
  _skip_test_if_needed "laravel-composer-sqlite"

  run mkdir my-laravel-site && cd my-laravel-site
  assert_success

  run ddev config --project-type=laravel --docroot=public --omit-containers=db
  assert_success

  run ddev start -y
  assert_success

  run ddev composer create-project "laravel/laravel:^12"
  assert_success

  DDEV_DEBUG=true run ddev launch
  assert_output --partial "FULLURL https://${PROJNAME}.ddev.site"
  assert_success

  # validate running project
  run curl -sfIv https://${PROJNAME}.ddev.site
  assert_output --partial "HTTP/2 200"
  assert_success

  run curl -sfv https://${PROJNAME}.ddev.site
  assert_output --partial "Laravel"
  assert_success

  # check used database
  run ddev artisan about
  assert_success
  assert_output --partial "sqlite"
}

@test "Laravel installer quickstart with $(ddev --version)" {
  _skip_test_if_needed "laravel-installer"

  run mkdir my-laravel-site && cd my-laravel-site
  assert_success

  run ddev config --project-type=laravel --docroot=public
  assert_success

  cat <<'DOCKERFILEEND' >.ddev/web-build/Dockerfile.laravel
ARG COMPOSER_HOME=/usr/local/composer
RUN composer global require laravel/installer
RUN ln -s $COMPOSER_HOME/vendor/bin/laravel /usr/local/bin/laravel
DOCKERFILEEND
  assert_file_exist .ddev/web-build/Dockerfile.laravel

  run ddev start -y
  assert_success

  ddev exec "yes | laravel new temp --database=sqlite --no-interaction"
  assert_success

  run ddev exec 'rsync -rltgopD temp/ ./ && rm -rf temp'
  assert_success

  # check used database
  run ddev artisan about
  assert_success
  assert_output --partial "sqlite"

  # and switch to MariaDB by removing .env and running post-install actions
  run rm -f .ddev/web-build/Dockerfile.laravel .env
  assert_success

  run ddev restart -y
  assert_success

  run ddev composer run-script post-root-package-install
  assert_success
  run ddev composer run-script post-create-project-cmd
  assert_success

  DDEV_DEBUG=true run ddev launch
  assert_output --partial "FULLURL https://${PROJNAME}.ddev.site"
  assert_success

  # validate running project
  run curl -sfIv https://${PROJNAME}.ddev.site
  assert_output --partial "HTTP/2 200"
  assert_success

  run curl -sfv https://${PROJNAME}.ddev.site
  assert_output --partial "Laravel"
  assert_success

  # check used database
  run ddev artisan about
  assert_success
  assert_output --partial "mariadb"
}
