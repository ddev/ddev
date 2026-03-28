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
  _skip_if_embargoed "joomla-zip"

  run mkdir -p ${PROJNAME} && cd ${PROJNAME}
  assert_success
  # Download the latest version of Joomla
  run curl -o joomla.zip -L "https://www.joomla.org/latest"
  assert_success
  # unzip joomla.zip && rm -f joomla.zip
  run unzip joomla.zip && rm -f joomla.zip
  assert_success
  run ddev config --project-type=joomla --upload-dirs=images
  assert_success
  run echo "display_errors = off" > .ddev/php/joomla.ini
  run echo "output_buffering = off" >> .ddev/php/joomla.ini
  assert_success
  run ddev start -y
  assert_success
  run ddev php installation/joomla.php install --site-name="My Joomla Site" --admin-user="Administrator" --admin-username=admin --admin-password=AdminAdmin1! --admin-email=admin@example.com --db-type=mysql --db-encryption=0 --db-host=db --db-user=db --db-pass="db" --db-name=db --db-prefix=ddev_ --public-folder=""
  assert_success
  DDEV_DEBUG=true run ddev launch /administrator/
  assert_output "FULLURL https://${PROJNAME}.ddev.site/administrator/"
  assert_success
  # validate running project
  run curl -sfv https://${PROJNAME}.ddev.site
  assert_output --partial "<meta name=\"generator\" content=\"Joomla! - Open Source Content Management\">"
  assert_output --partial "alt=\"My Joomla Site\""
  assert_success
  run curl -sfIv https://${PROJNAME}.ddev.site/administrator/
  assert_output --partial "HTTP/2 200"
  run curl -sfv https://${PROJNAME}.ddev.site/administrator/
  assert_success
  assert_output --partial "<meta name=\"generator\" content=\"Joomla! - Open Source Content Management\">"
  assert_success
}
