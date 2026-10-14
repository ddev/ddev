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
  run -0 mkdir -p ${PROJNAME} && cd ${PROJNAME}
  run -0 ddev config --project-type=php --docroot=public --upload-dirs=uploads --database=mysql:8.0
  # ddev start -y
  run -0 ddev start -y

  _extra_info

  run -0 ddev composer create-project sulu/skeleton
  export SULU_PROJECT_NAME="My Sulu Site"
  export SULU_PROJECT_KEY="${PROJNAME}"
  export SULU_PROJECT_CONFIG_FILE="config/webspaces/${PROJNAME}.xml"
  run -0 ddev exec "mv config/webspaces/website.xml ${SULU_PROJECT_CONFIG_FILE}"
  run -0 ddev exec "sed -i -e 's|<name>.*</name>|<name>${SULU_PROJECT_NAME}</name>|g' -e 's|<key>.*</key>|<key>${SULU_PROJECT_KEY}</key>|g' ${SULU_PROJECT_CONFIG_FILE}"
  # Set APP_ENV and DATABASE_URL in .env.local
  run -0 ddev dotenv set .env.local --app-env=dev --database-url="mysql://db:db@db:3306/db?serverVersion=8.0&charset=utf8mb4"
  run -0 ddev exec bin/adminconsole sulu:build dev --no-interaction
  DDEV_DEBUG=true run -0 ddev launch /admin
  assert_output --partial "FULLURL ${PRIMARY_URL}/admin"
  # validate running project
  run -0 curl -sSfI ${PRIMARY_URL}
  assert_output --partial "x-generator: Sulu"
  assert_output --partial "HTTP/2 200"
}

