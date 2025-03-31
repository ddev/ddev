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
  run mkdir ${PROJNAME} && cd ${PROJNAME}
  assert_success

  run ddev config --project-type=backdrop
  assert_success

  # Add the official Bee CLI add-on
  run ddev add-on get backdrop-ops/ddev-backdrop-bee
  assert_success

  run ddev start -y
  assert_success

  # Download Backdrop core
  run ddev bee download-core
  assert_success

  # Create admin user
  run ddev bee si --username=admin --password=Password123 --db-name=db --db-user=db --db-pass=db --db-host=db --auto
  assert_success

  run bash -c "DDEV_DEBUG=true ddev launch"
  assert_output "FULLURL https://${PROJNAME}.ddev.site"
  assert_success
  # validate running project
  run curl -sfI https://${PROJNAME}.ddev.site
  assert_success
  assert_output --partial "HTTP/2 200"
  run curl -sf https://${PROJNAME}.ddev.site
  assert_success
  assert_output --partial "Welcome to My Backdrop Site!"
}

@test "backdrop existing project with $(ddev --version)" {
  run mkdir ${PROJNAME} && cd ${PROJNAME}
  assert_success

  run git clone https://github.com/ddev/test-backdrop.git .
  assert_success

  run ddev config --project-type=backdrop
  assert_success

  # Add the official Bee CLI add-on
  run ddev add-on get backdrop-ops/ddev-backdrop-bee
  assert_success

  run ddev start -y
  assert_success

  run curl -fLO https://github.com/ddev/test-backdrop/releases/download/1.29.2/db.sql.gz
  assert_success

  run ddev import-db --file=db.sql.gz
  assert_success

  run curl -fLO https://github.com/ddev/test-backdrop/releases/download/1.29.2/files.tgz
  assert_success

  run ddev import-files --source=files.tgz
  assert_success

  run bash -c "DDEV_DEBUG=true ddev launch"
  assert_output "FULLURL https://${PROJNAME}.ddev.site"
  assert_success

  # validate running project
  run curl -sfI https://${PROJNAME}.ddev.site
  assert_success
  assert_output --partial "HTTP/2 200"
  run curl -sf https://${PROJNAME}.ddev.site
  assert_success
  assert_output --partial "Welcome to My Backdrop Site!"
}
