#!/usr/bin/env bats

setup() {
  PROJNAME=my-pimcore-site
  load 'common-setup'
  _common_setup
}

# executed after each test
teardown() {
  _common_teardown
}

@test "Pimcore Composer quickstart with $(ddev --version)" {
  skip "Pimcore requires a license key"

  run -0 mkdir -p ${PROJNAME} && cd ${PROJNAME}
  run -0 ddev config --project-type=php --docroot=public --webimage-extra-packages='php${DDEV_PHP_VERSION}-amqp'
  run -0 ddev start -y

  _extra_info

  run -0 ddev composer create-project pimcore/skeleton
  run -0 ddev exec pimcore-install --mysql-username=db --mysql-password=db --mysql-host-socket=db --mysql-database=db --admin-password=admin --admin-username=admin
  # echo "web_extra_daemons:
  #   - name: consumer
  #     command: 'while true; do /var/www/html/bin/console messenger:consume pimcore_core pimcore_maintenance pimcore_scheduled_tasks pimcore_image_optimize pimcore_asset_update --memory-limit=250M --time-limit=3600; done'
  #     directory: /var/www/html" >.ddev/config.pimcore.yaml
  run echo "web_extra_daemons:
     - name: consumer
       command: 'while true; do /var/www/html/bin/console messenger:consume pimcore_core pimcore_maintenance pimcore_scheduled_tasks pimcore_image_optimize pimcore_asset_update --memory-limit=250M --time-limit=3600; done'
       directory: /var/www/html" >.ddev/config.pimcore.yaml
  assert_success
  run -0 ddev restart -y
  DDEV_DEBUG=true run -0 ddev launch /admin
  assert_output "FULLURL ${PRIMARY_URL}/admin"
  # validate running project
  run -0 curl -sSfI ${PRIMARY_URL}
  assert_output --partial "HTTP/2 200"
  assert_output --partial "x-powered-by: pimcore"
  assert_output --partial "x-pimcore-output-cache-disable-reason:"
}
