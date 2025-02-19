#!/usr/bin/env bats

setup() {
  PROJNAME=my-civicrm-site
  load 'common-setup'
  _common_setup
}

# executed after each test
teardown() {
  _common_teardown
}

@test "CiviCRM quickstart with $(ddev --version)" {
  # mkdir ${PROJNAME} && cd ${PROJNAME}
  run mkdir ${PROJNAME} && cd ${PROJNAME}
  assert_success
  #ddev config --project-type=php --composer-root=core --upload-dirs=public/media
  run ddev config --project-type=php --composer-root=core --upload-dirs=public/media
  assert_success
  # ddev start
  run ddev start
  assert_success
  # ddev exec "curl -LsS https://download.civicrm.org/latest/civicrm-STABLE-standalone.tar.gz -o /tmp/civicrm-standalone.tar.gz"
  run ddev exec "curl -LsS https://download.civicrm.org/latest/civicrm-STABLE-standalone.tar.gz -o /tmp/civicrm-standalone.tar.gz"
  assert_success
  # ddev exec "tar --strip-components=1 -xzf /tmp/civicrm-standalone.tar.gz"
  run ddev exec "tar --strip-components=1 -xzf /tmp/civicrm-standalone.tar.gz"
  assert_success
  # ddev composer require civicrm/cli-tools --no-scripts
  run ddev composer require civicrm/cli-tools --no-scripts
  assert_success
  # ddev exec cv core:install \
  #    --cms-base-url='$DDEV_PRIMARY_URL' \
  #    --db=mysql://db:db@db/db \
  #    -m loadGenerated=1 \
  #    -m extras.adminUser=admin \
  #    -m extras.adminPass=admin \
  #    -m extras.adminEmail=admin@example.com
  run ddev exec cv core:install \
      --cms-base-url='$DDEV_PRIMARY_URL' \
      --db=mysql://db:db@db/db \
      -m loadGenerated=1 \
      -m extras.adminUser=admin \
      -m extras.adminPass=admin \
      -m extras.adminEmail=admin@example.com
  assert_success
  # ddev launch
  run bash -c "DDEV_DEBUG=true ddev launch"
  assert_output "FULLURL https://${PROJNAME}.ddev.site"
  assert_success
  # validate running project
  run curl -sfI https://${PROJNAME}.ddev.site
  assert_success
  assert_output --partial "location: /civicrm/home"
  assert_output --partial "HTTP/2 302"
}
