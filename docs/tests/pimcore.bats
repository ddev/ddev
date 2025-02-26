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
  # mkdir -p ${PROJNAME} && cd ${PROJNAME}
  run mkdir -p ${PROJNAME} && cd ${PROJNAME}
  assert_success
  # ddev config --project-type=php --docroot=public --webimage-extra-packages='php${DDEV_PHP_VERSION}-amqp'
  run ddev config --project-type=php --docroot=public --webimage-extra-packages='php${DDEV_PHP_VERSION}-amqp'
  assert_success
  # ddev start -y
  run ddev start -y
  assert_success
  # ddev composer create-project pimcore/skeleton .
  run ddev composer create-project pimcore/skeleton .
  assert_success
  # ddev exec pimcore-install --mysql-username=db --mysql-password=db --mysql-host-socket=db --mysql-database=db --admin-password=admin --admin-username=admin --no-interaction
  run ddev exec pimcore-install --mysql-username=db --mysql-password=db --mysql-host-socket=db --mysql-database=db --admin-password=admin --admin-username=admin --no-interaction
  assert_success
  # echo "web_extra_daemons:
  #   - name: consumer
  #     command: 'while true; do /var/www/html/bin/console messenger:consume pimcore_core pimcore_maintenance pimcore_scheduled_tasks pimcore_image_optimize pimcore_asset_update --memory-limit=250M --time-limit=3600; done'
  #     directory: /var/www/html" >.ddev/config.pimcore.yaml
  run echo "web_extra_daemons:
     - name: consumer
       command: 'while true; do /var/www/html/bin/console messenger:consume pimcore_core pimcore_maintenance pimcore_scheduled_tasks pimcore_image_optimize pimcore_asset_update --memory-limit=250M --time-limit=3600; done'
       directory: /var/www/html" >.ddev/config.pimcore.yaml
  assert_success
  # ddev restart -y
  run ddev restart -y
  assert_success
  # ddev launch
  run bash -c "DDEV_DEBUG=true ddev launch /admin"
  assert_output "FULLURL https://${PROJNAME}.ddev.site/admin"
  assert_success
  # validate running project
  run curl -sfI https://${PROJNAME}.ddev.site
  assert_success
  assert_output --partial "HTTP/2 200"
  assert_output --partial "x-powered-by: pimcore"
  assert_output --partial "x-pimcore-output-cache-disable-reason:"
}
