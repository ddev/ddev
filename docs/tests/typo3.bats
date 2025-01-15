#!/usr/bin/env bats

setup() {
  PROJNAME=my-typo3-site
  load 'common-setup'
  _common_setup
}

# executed after each test
teardown() {
  _common_teardown
}

@test "TYPO3 composer based quickstart with $(ddev --version)" {
  # mkdir my-typo3-site && cd my-typo3-site
  run mkdir my-typo3-site && cd my-typo3-site
  assert_success
  # ddev config --project-type=typo3 --docroot=public --php-version=8.3
  run ddev config --project-type=typo3 --docroot=public --php-version=8.3
  assert_success
  # ddev start
  run ddev start
  assert_success
  # ddev composer create "typo3/cms-base-distribution"
  run ddev composer create "typo3/cms-base-distribution"
  assert_success
  # ddev exec touch public/FIRST_INSTALL
  run ddev exec touch public/FIRST_INSTALL
  assert_success
  # ddev launch
  run bash -c "DDEV_DEBUG=true ddev launch"
  assert_output --partial "FULLURL https://${PROJNAME}.ddev.site"
  assert_success
  # validate running project
  run curl -sfI https://${PROJNAME}.ddev.site
  assert_success
  assert_output --partial "location: /typo3/install.php"
  assert_output --partial "HTTP/2 302"
}

@test "TYPO3 git based quickstart with $(ddev --version)" {
  # PROJECT_GIT_URL=https://github.com/ddev/test-typo3.git
  PROJECT_GIT_URL=https://github.com/ddev/test-typo3.git
  # git clone ${PROJECT_GIT_URL} ${PROJNAME}
  run git clone ${PROJECT_GIT_URL} ${PROJNAME}
  assert_success
  # cd my-typo3-site
  cd ${PROJNAME} || exit 2
  assert_success
  # ddev config --project-type=typo3 --docroot=public --php-version=8.3
  run ddev config --project-type=typo3 --docroot=public --php-version=8.3
  assert_success
  # ddev start
  run ddev start
  assert_success
  # ddev composer install
  run ddev composer install
  assert_success
  # ddev exec touch public/FIRST_INSTALL
  run ddev exec touch public/FIRST_INSTALL
  assert_success
  # ddev launch
  run bash -c "DDEV_DEBUG=true ddev launch"
  assert_output --partial "FULLURL https://${PROJNAME}.ddev.site"
  assert_success
  # validate running project
  run curl -sfI https://${PROJNAME}.ddev.site/typo3/install.php
  assert_success
  assert_output --partial "content-security-policy: default-src 'self'; script-src 'self'"
  assert_output --partial "HTTP/2 200"
}
