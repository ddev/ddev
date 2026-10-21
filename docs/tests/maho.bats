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

  run -0 mkdir -p ${PROJNAME}
  cd ${PROJNAME}

  run -0 ddev config --project-type=maho --docroot=public

  run -0 ddev start -y

  _extra_info

  run -0 _retry ddev composer create-project mahocommerce/maho-starter

  # Silent Maho install with sample data; --force is needed because
  # ddev pre-creates app/etc/local.xml
  run -0 ddev maho install --force \
    --license_agreement_accepted yes \
    --locale en_US --timezone UTC --default_currency USD \
    --db_host db --db_name db --db_user db --db_pass db \
    --url "$(ddev exec echo '$DDEV_PRIMARY_URL')" \
    --use_secure 1 --secure_base_url "$(ddev exec echo '$DDEV_PRIMARY_URL')" --use_secure_admin 1 \
    --admin_firstname Store --admin_lastname Admin --admin_email admin@example.com \
    --admin_username admin --admin_password veryl0ngpassw0rd \
    --sample_data 1

  run -0 ddev maho index:reindex:all

  run -0 ddev maho cache:flush

  # validate ddev launch
  DDEV_DEBUG=true run -0 ddev launch
  assert_output "FULLURL ${PRIMARY_URL}"

  # validate running project
  run -0 curl -sSfIL ${PRIMARY_URL}
  assert_output --partial "server: nginx"
  assert_output --partial "HTTP/2 200"

  # Check if the admin is working
  _assert_page_contains -L ${PRIMARY_URL}/admin "Log in to Admin Panel"
}
