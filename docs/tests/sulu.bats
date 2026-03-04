#!/usr/bin/env bats

setup() {
  PROJNAME=my-sulu-site
  load 'common-setup'
  _common_setup
}

# executed after each test
teardown() {
  _common_teardown
}

@test "Sulu quickstart with $(ddev --version)" {
  _skip_if_embargoed "sulu-composer"

  run mkdir -p ${PROJNAME} && cd ${PROJNAME}
  assert_success
  run ddev config --project-type=php --docroot=public --upload-dirs=uploads --database=mysql:8.0
  assert_success
  # ddev start -y
  run ddev start -y
  assert_success
  run ddev composer create-project sulu/skeleton
  assert_success
  export SULU_PROJECT_NAME="My Sulu Site"
  assert_success
  export SULU_PROJECT_KEY="${PROJNAME}"
  assert_success
  export SULU_PROJECT_CONFIG_FILE="config/webspaces/${PROJNAME}.xml"
  assert_success
  run ddev exec "mv config/webspaces/website.xml ${SULU_PROJECT_CONFIG_FILE}"
  assert_success
  run ddev exec "sed -i -e 's|<name>.*</name>|<name>${SULU_PROJECT_NAME}</name>|g' -e 's|<key>.*</key>|<key>${SULU_PROJECT_KEY}</key>|g' ${SULU_PROJECT_CONFIG_FILE}"
  assert_success
  # Set APP_ENV and DATABASE_URL in .env.local
  run ddev dotenv set .env.local --app-env=dev --database-url="mysql://db:db@db:3306/db?serverVersion=8.0&charset=utf8mb4"
  assert_success
  run ddev exec bin/adminconsole sulu:build dev --no-interaction
  assert_success
  DDEV_DEBUG=true run ddev launch /admin
  assert_output --partial "FULLURL https://${PROJNAME}.ddev.site/admin"
  assert_success
  # validate running project
  run curl -sfIv https://${PROJNAME}.ddev.site
  assert_output --partial "x-generator: Sulu"
  assert_output --partial "HTTP/2 200"
  assert_success
}

