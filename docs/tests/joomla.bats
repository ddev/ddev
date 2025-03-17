#!/usr/bin/env bats

setup() {
  PROJNAME=my-joomla-site
  load 'common-setup'
  _common_setup
}

# executed after each test
teardown() {
  _common_teardown
}

@test "Joomla quickstart with $(ddev --version)" {
  # mkdir ${PROJNAME} && cd ${PROJNAME}
  run mkdir ${PROJNAME} && cd ${PROJNAME}
  assert_success
  # Download the latest version of Joomla
  run curl -o joomla.zip -L $(curl -sL https://api.github.com/repos/joomla/joomla-cms/releases/latest | docker run -i --rm ddev/ddev-utilities jq -r '.assets | map(select(.name | test("^Joomla.*Stable-Full_Package\\.zip$")))[0].browser_download_url')
  assert_success
  # unzip ./joomla.zip && rm -f joomla.zip
  run unzip joomla.zip && rm -f joomla.zip
  assert_success
  # ddev config --project-type=php --webserver-type=apache-fpm --upload-dirs=images
  run ddev config --project-type=php --webserver-type=apache-fpm --upload-dirs=images
  assert_success
  # ddev start -y
  run ddev start -y
  assert_success
  # ddev php installation/joomla.php install --site-name="My Joomla Site" --admin-user="Administrator" --admin-username=admin --admin-password=AdminAdmin1! --admin-email=admin@example.com --db-type=mysql --db-encryption=0 --db-host=db --db-user=db --db-pass="db" --db-name=db --db-prefix=ddev_ --public-folder=""
  run ddev php installation/joomla.php install --site-name="My Joomla Site" --admin-user="Administrator" --admin-username=admin --admin-password=AdminAdmin1! --admin-email=admin@example.com --db-type=mysql --db-encryption=0 --db-host=db --db-user=db --db-pass="db" --db-name=db --db-prefix=ddev_ --public-folder=""
  assert_success
  # ddev launch
  run bash -c "DDEV_DEBUG=true ddev launch /administrator/"
  assert_output "FULLURL https://${PROJNAME}.ddev.site/administrator/"
  assert_success
  # validate running project
  run curl -sf https://${PROJNAME}.ddev.site
  assert_success
  assert_output --partial "<meta name=\"generator\" content=\"Joomla! - Open Source Content Management\">"
  assert_output --partial "alt=\"My Joomla Site\""
  run curl -sfI https://${PROJNAME}.ddev.site/administrator/
  assert_success
  assert_output --partial "HTTP/2 200"
  run curl -sf https://${PROJNAME}.ddev.site/administrator/
  assert_success
  assert_output --partial "<meta name=\"generator\" content=\"Joomla! - Open Source Content Management\">"
}
