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
  run -0 mkdir -p ${PROJNAME} && cd ${PROJNAME}

  # Download the latest version of Joomla
  run -0 curl -o joomla.zip -L "https://www.joomla.org/latest"

  run -0 unzip joomla.zip && rm -f joomla.zip

  run -0 ddev config --project-type=joomla

  run -0 echo "display_errors = off" > .ddev/php/joomla.ini

  run -0 echo "output_buffering = off" >> .ddev/php/joomla.ini
  assert_file_exist .ddev/php/joomla.ini

  run -0 ddev start -y

  _extra_info

  run -0 ddev php installation/joomla.php install --site-name="My Joomla Site" --admin-user="Administrator" --admin-username=admin --admin-password=AdminAdmin1! --admin-email=admin@example.com --db-type=mysql --db-encryption=0 --db-host=db --db-user=db --db-pass="db" --db-name=db --db-prefix=ddev_ --public-folder=""

  DDEV_DEBUG=true run -0 ddev launch /administrator/
  assert_output "FULLURL ${PRIMARY_URL}/administrator/"

  # validate running project
  _assert_page_contains ${PRIMARY_URL} \
    "<meta name=\"generator\" content=\"Joomla! - Open Source Content Management\">" \
    "alt=\"My Joomla Site\""

  run -0 curl -sSfI ${PRIMARY_URL}/administrator/
  assert_output --partial "HTTP/2 200"

  _assert_page_contains ${PRIMARY_URL}/administrator/ "<meta name=\"generator\" content=\"Joomla! - Open Source Content Management\">"
}
