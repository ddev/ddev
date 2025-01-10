#!/usr/bin/env bats

setup() {
  PROJNAME=my-backdrop-site
  load 'common-setup'
  _common_setup
}

# executed after each test
teardown() {
  _common_teardown
}

@test "backdrop new-project quickstart with $(ddev --version)" {
  # mkdir my-backdrop-site && cd my-backdrop-site
  run mkdir -p my-backdrop-site && cd my-backdrop-site
  assert_success
  # curl -LJO https://github.com/backdrop/backdrop/releases/latest/download/backdrop.zip
  run curl -LJO https://github.com/backdrop/backdrop/releases/latest/download/backdrop.zip
  assert_success
  # unzip ./backdrop.zip && rm -f backdrop.zip && mv -f ./backdrop/{.,}* . ; rm -rf backdrop
  run unzip -o ./backdrop.zip && rm -f backdrop.zip && mv -f ./backdrop/{.,}* . ; rm -rf backdrop
  assert_success
  # ddev config --project-type=backdrop
  run ddev config --project-type=backdrop
  assert_success
  # ddev start
  run ddev start -y
  assert_success
  # ddev launch
  run bash -c "DDEV_DEBUG=true ddev launch"
  assert_output "FULLURL https://${PROJNAME}.ddev.site"
  assert_success
  # validate running project
  run curl -sfI https://${PROJNAME}.ddev.site
  assert_success
  assert_output --partial "location: https://${PROJNAME}.ddev.site/core/install.php"
  assert_output --partial "HTTP/2 302"
}

@test "backdrop existing project with $(ddev --version)" {
  # PROJECT_GIT_URL=https://github.com/ddev/test-backdrop.git
  PROJECT_GIT_URL=https://github.com/ddev/test-backdrop.git
  # git clone ${PROJECT_GIT_URL} my-backdrop-site
  run git clone ${PROJECT_GIT_URL} ${PROJNAME}
  assert_success
  # cd my-backdrop-site
  cd ${PROJNAME} || exit 2
  assert_success
  # ddev config --project-type=backdrop
  run ddev config --project-type=backdrop
  assert_success
  # ddev start
  run ddev start -y
  assert_success
  run curl -fLO https://github.com/ddev/test-backdrop/releases/download/1.29.2/db.sql.gz
  assert_success
  # ddev import-db --file=/path/to/db.sql.gz
  run ddev import-db --file=db.sql.gz
  assert_success
  run curl -fLO https://github.com/ddev/test-backdrop/releases/download/1.29.2/files.tgz
  assert_success
  # ddev import-files --source=/path/to/files.tar.gz
  run ddev import-files --source=files.tgz
  # ddev launch
  run bash -c "DDEV_DEBUG=true ddev launch"
  assert_output "FULLURL https://${PROJNAME}.ddev.site"
  assert_success
  run curl -sfI https://${PROJNAME}.ddev.site
  assert_success
  assert_output --partial "HTTP/2 200"
  run curl -sf https://${PROJNAME}.ddev.site
  assert_success
  assert_output --partial "Welcome to My Backdrop Site!"
}
