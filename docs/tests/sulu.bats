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
  # mkdir ${PROJNAME} && cd ${PROJNAME}
  run mkdir ${PROJNAME} && cd ${PROJNAME}
  assert_success
  # ddev config --project-type=php --docroot=public --upload-dirs=uploads --database=mysql:8.0
  run ddev config --project-type=php --docroot=public --upload-dirs=uploads --database=mysql:8.0
  assert_success
  # ddev start
  run ddev start
  assert_success
  # ddev composer create sulu/skeleton
  run ddev composer create sulu/skeleton
  assert_success
  # export SULU_PROJECT_NAME="My Sulu Site"
  export SULU_PROJECT_NAME="My Sulu Site"
  assert_success
  # export SULU_PROJECT_KEY="${PROJNAME}"
  export SULU_PROJECT_KEY="${PROJNAME}"
  assert_success
  # export SULU_PROJECT_CONFIG_FILE="config/webspaces/my-sulu-site.xml"
  export SULU_PROJECT_CONFIG_FILE="config/webspaces/${PROJNAME}.xml"
  assert_success
  # ddev exec "mv config/webspaces/website.xml ${SULU_PROJECT_CONFIG_FILE}"
  run ddev exec "mv config/webspaces/website.xml ${SULU_PROJECT_CONFIG_FILE}"
  assert_success
  # ddev exec "sed -i -e 's|<name>.*</name>|<name>${SULU_PROJECT_NAME}</name>|g' -e 's|<key>.*</key>|<key>${SULU_PROJECT_KEY}</key>|g' ${SULU_PROJECT_CONFIG_FILE}"
  run ddev exec "sed -i -e 's|<name>.*</name>|<name>${SULU_PROJECT_NAME}</name>|g' -e 's|<key>.*</key>|<key>${SULU_PROJECT_KEY}</key>|g' ${SULU_PROJECT_CONFIG_FILE}"
  assert_success
  # Set APP_ENV and DATABASE_URL in .env.local
  # ddev dotenv set .env.local --app-env=dev --database-url="mysql://db:db@db:3306/db?serverVersion=8.0&charset=utf8mb4"
  run ddev dotenv set .env.local --app-env=dev --database-url="mysql://db:db@db:3306/db?serverVersion=8.0&charset=utf8mb4"
  # ddev exec bin/adminconsole sulu:build dev --no-interaction
  run ddev exec bin/adminconsole sulu:build dev --no-interaction
  # ddev launch
  run bash -c "DDEV_DEBUG=true ddev launch /admin"
  assert_output --partial "FULLURL https://${PROJNAME}.ddev.site/admin"
  assert_success
  # validate running project
  run curl -sfI https://${PROJNAME}.ddev.site
  assert_success
  assert_output --partial "x-generator: Sulu"
  assert_output --partial "HTTP/2 200"
}

