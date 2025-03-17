#!/usr/bin/env bats

setup() {
  PROJNAME=my-craft-site
  load 'common-setup'
  _common_setup
}

# executed after each test
#teardown() {
#  _common_teardown
#}

@test "Craft CMS New Projects quickstart with $(ddev --version)" {
  # mkdir ${PROJNAME} && cd ${PROJNAME}
  run mkdir ${PROJNAME} && cd ${PROJNAME}
  assert_success

  # ddev config --project-type=craftcms --docroot=web
  run ddev config --project-type=craftcms --docroot=web
  assert_success

  # ddev start -y
  run ddev start -y
  assert_success

  # ddev composer create craftcms/craft
  #run ddev composer create craftcms/craft

  run bash -c 'expect -d -c "
    log_file -noappend /Users/rkoller/Desktop/craft.log
    spawn ddev composer create craftcms/craft
    expect \"Username: \[admin\] \"
    send \"\r\"
    expect \"Email:\"
    send \"admin@mail.com\r\"
    sleep 1
    expect \"Password:\"
    send \"admin123\r\"
    expect \"Confirm:\"
    send \"admin123\r\"
    expect \"Site name:\"
    send \"CraftCMS Site\r\"
    expect \"Site URL: \[https:\/\/my-craft-site.ddev.site\] \"
    send \"\r\"
    expect \"Site language: \[en-US\]\"
    send \"\r\"
  "'
  assert_success

  # validate ddev launch
  run bash -c "DDEV_DEBUG=true ddev launch"
  assert_output "FULLURL https://${PROJNAME}.ddev.site"
  assert_success
  run bash -c "DDEV_DEBUG=true ddev launch /admin/login"
  assert_output "FULLURL https://${PROJNAME}.ddev.site/admin/login"
  assert_success


  ## validate running project
  run curl -sfI https://${PROJNAME}.ddev.site
  assert_success
  assert_output --partial "server: nginx"
  assert_output --partial "HTTP/2 200"
  assert_output --partial "x-powered-by: Craft CMS"

  run curl -sf https://${PROJNAME}.ddev.site
  assert_success
  assert_output --partial "<title>Welcome to Craft CMS</title>"
  assert_output --partial "<h2>Popular Resources</h2>"
  run curl -sf https://${PROJNAME}.ddev.site/admin/login
  assert_success
  assert_output --partial "<title>Sign In - CraftCMS Site</title>"

}

@test "Craft CMS Existing Projects quickstart with $(ddev --version)" {
  skip "Does not have a test yet"
}
