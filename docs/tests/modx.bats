#!/usr/bin/env bats

setup() {
  PROJNAME=my-modx-site
  load 'common-setup'
  _common_setup
}

# executed after each test
teardown() {
  _common_teardown
}

@test "MODX Composer quickstart with $(ddev --version)" {
  run mkdir -p ${PROJNAME} && cd ${PROJNAME}
  assert_success

  run ddev config --project-type=modx
  assert_success

  run ddev start -y
  assert_success

  # Install MODX Revolution via Composer (resolves to the latest stable release)
  run ddev composer create-project modx/revolution
  assert_success

  # Perform a fresh install using the DDEV database credentials
  run ddev exec php setup/cli-install.php \
    --database_server=db --database=db --database_user=db --database_password=db \
    --table_prefix=modx_ --http_host=${PROJNAME}.ddev.site \
    --cmsadmin=admin --cmspassword=Admin123! --cmsadminemail=admin@example.com --language=en
  assert_success

  # validate ddev launch
  DDEV_DEBUG=true run ddev launch /manager/
  assert_output "FULLURL https://${PROJNAME}.ddev.site/manager/"
  assert_success

  # validate running project
  run curl -sfIv https://${PROJNAME}.ddev.site
  assert_output --partial "HTTP/2 200"
  assert_success

  # validate the MODX manager (backend) is reachable
  run curl -sfIv https://${PROJNAME}.ddev.site/manager/
  assert_output --partial "HTTP/2 200"
  assert_success
}

@test "MODX ZIP Download quickstart with $(ddev --version)" {
  skip "Does not have a test yet"
}
