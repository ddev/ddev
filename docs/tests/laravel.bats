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
  PROJNAME=my-laravel-composer-site

  run -0 mkdir -p my-laravel-composer-site && cd my-laravel-composer-site

  run -0 ddev config --project-type=laravel --docroot=public

  run -0 ddev start -y

  run -0 _retry ddev composer create-project laravel/laravel

  _extra_info

  DDEV_DEBUG=true run -0 ddev launch
  assert_output --partial "FULLURL ${PRIMARY_URL}"

  # validate running project
  run -0 curl -sSfI ${PRIMARY_URL}
  assert_output --partial "HTTP/2 200"

  _assert_page_contains ${PRIMARY_URL} "Laravel"

  # check used database
  run -0 ddev artisan about
  assert_output --partial "mariadb"
}

@test "Laravel composer (SQLite) based quickstart with $(ddev --version)" {
  PROJNAME=my-laravel-sqlite-site

  run -0 mkdir -p my-laravel-sqlite-site && cd my-laravel-sqlite-site

  run -0 ddev config --project-type=laravel --docroot=public --omit-containers=db

  run -0 ddev start -y

  run -0 _retry ddev composer create-project laravel/laravel

  _extra_info

  DDEV_DEBUG=true run -0 ddev launch
  assert_output --partial "FULLURL ${PRIMARY_URL}"

  # validate running project
  run -0 curl -sSfI ${PRIMARY_URL}
  assert_output --partial "HTTP/2 200"

  _assert_page_contains ${PRIMARY_URL} "Laravel"

  # check used database
  run -0 ddev artisan about
  assert_output --partial "sqlite"
}

@test "Laravel installer quickstart with $(ddev --version)" {
  PROJNAME=my-laravel-installer-site

  run -0 mkdir -p my-laravel-installer-site && cd my-laravel-installer-site

  run -0 ddev config --project-type=laravel --docroot=public

  cat <<'DOCKERFILEEND' >.ddev/web-build/Dockerfile.laravel
ARG COMPOSER_HOME=/usr/local/composer
RUN composer global require laravel/installer
RUN ln -s $COMPOSER_HOME/vendor/bin/laravel /usr/local/bin/laravel
DOCKERFILEEND
  assert_file_exist .ddev/web-build/Dockerfile.laravel

  run -0 ddev start -y

  ddev exec "yes | laravel new temp --database=sqlite --no-interaction"

  run -0 ddev exec 'rsync -rltgopD temp/ ./ && rm -rf temp'

  # check used database
  run -0 ddev artisan about
  assert_output --partial "sqlite"

  # and switch to MariaDB by removing .env and running post-install actions
  run -0 rm -f .ddev/web-build/Dockerfile.laravel .env

  run -0 ddev restart -y

  run -0 ddev composer run-script post-root-package-install
  run -0 ddev composer run-script post-create-project-cmd

  _extra_info

  DDEV_DEBUG=true run -0 ddev launch
  assert_output --partial "FULLURL ${PRIMARY_URL}"

  # validate running project
  run -0 curl -sSfI ${PRIMARY_URL}
  assert_output --partial "HTTP/2 200"

  _assert_page_contains ${PRIMARY_URL} "Laravel"

  # check used database
  run -0 ddev artisan about
  assert_output --partial "mariadb"
}
