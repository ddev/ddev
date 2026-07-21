#!/usr/bin/env bats

setup() {
  PROJNAME=my-maho-site
  load 'common-setup'
  _common_setup
}

# executed after each test
teardown() {
  _common_teardown
}

@test "Maho Composer quickstart with $(ddev --version)" {
  PROJNAME=my-maho-site

  run mkdir -p ${PROJNAME} && cd ${PROJNAME}
  assert_success

  run ddev config --project-type=maho --docroot=public
  assert_success

  run ddev start -y
  assert_success

  run ddev composer create-project mahocommerce/maho-starter
  assert_success

  # Silent Maho install with sample data; --force is needed because
  # ddev pre-creates app/etc/local.xml
  run ddev maho install --force \
    --license_agreement_accepted yes \
    --locale en_US --timezone UTC --default_currency USD \
    --db_host db --db_name db --db_user db --db_pass db \
    --url "https://${PROJNAME}.ddev.site/" \
    --use_secure 1 --secure_base_url "https://${PROJNAME}.ddev.site/" --use_secure_admin 1 \
    --admin_firstname Store --admin_lastname Admin --admin_email admin@example.com \
    --admin_username admin --admin_password veryl0ngpassw0rd \
    --sample_data 1
  assert_success

  run ddev maho index:reindex:all
  assert_success

  run ddev maho cache:flush
  assert_success

  # validate ddev launch
  DDEV_DEBUG=true run ddev launch
  assert_output "FULLURL https://${PROJNAME}.ddev.site"
  assert_success

  # validate running project
  run curl -sfIv https://${PROJNAME}.ddev.site
  assert_output --partial "server: nginx"
  assert_output --partial "HTTP/2 200"
  assert_success

  # Check if the admin is working
  run curl -sfvL https://${PROJNAME}.ddev.site/admin
  assert_output --partial "Log in to Admin Panel"
  assert_success
}
