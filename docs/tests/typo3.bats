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
  PROJNAME=my-typo3-site
  run mkdir -p ${PROJNAME} && cd ${PROJNAME}
  assert_success
  run ddev config --project-type=typo3 --docroot=public --php-version=8.3
  assert_success
  run ddev start -y
  assert_success
  run ddev composer create-project "typo3/cms-base-distribution"
  assert_success
  run ddev exec touch public/FIRST_INSTALL
  assert_success
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
  PROJECT_GIT_URL=https://github.com/ddev/test-typo3.git
  PROJNAME=my-typo3-site
  run git clone ${PROJECT_GIT_URL} ${PROJNAME}
  assert_success
  # cd my-typo3-site
  cd ${PROJNAME} || exit 2
  assert_success
  # ddev config --project-type=typo3 --docroot=public --php-version=8.3
  run ddev config --project-type=typo3 --docroot=public --php-version=8.3
  assert_success
  # ddev start -y
  run ddev start -y
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

@test "TYPO3 XHGui composer test with $(ddev --version)" {
  run mkdir my-typo3-site && cd my-typo3-site
  assert_success

  run ddev config --project-type=typo3 --docroot=public --php-version=8.3 --xhprof-mode=xhgui
  assert_success

  run ddev start -y >/dev/null
  assert_success

  run ddev composer create-project typo3/cms-base-distribution >/dev/null
  assert_success

  run ddev exec touch public/FIRST_INSTALL
  assert_success

  run ddev xhgui on
  assert_success
  assert_output --partial "Started optional compose profiles"
  assert_output --partial "xhgui"

  # Ensure there's no profiling data link
  run ddev exec "curl -s xhgui:80 | grep -q '<a href=\"/?server_name=${PROJNAME}.ddev.site\">'"
  assert_failure

  # Profile site
  run curl -sfI https://${PROJNAME}.ddev.site/typo3/install.php
  assert_success
  assert_output --partial "HTTP/2 200"

  run curl -sf https://${PROJNAME}.ddev.site/typo3/install.php
  assert_success
  assert_output --partial "Installing TYPO3 CMS"

  # wait for XHGui
  sleep 5

  # Ensure there a profiling data link
  run ddev exec "curl -s xhgui:80 | grep -q '<a href=\"/?server_name=${PROJNAME}.ddev.site\">'"
  assert_success
}

# bats test_tags=typo3-setup,t3v13
@test "TYPO3 v13 'ddev typo3 setup' composer test with $(ddev --version)" {
  PROJNAME=my-typo3-site
  run mkdir -p ${PROJNAME} && cd ${PROJNAME}
  assert_success

  run ddev config --project-type=typo3 --docroot=public --php-version=8.3
  assert_success

  run ddev start -y >/dev/null
  assert_success

  run ddev composer create-project typo3/cms-base-distribution >/dev/null
  assert_success

  run ddev exec touch public/FIRST_INSTALL
  assert_success

  run ddev typo3 setup \
    --admin-user-password="Demo123*" \
    --driver=mysqli \
    --create-site=https://${PROJNAME}.ddev.site \
    --server-type=other \
    --dbname=db \
    --username=db \
    --password=db \
    --port=3306 \
    --host=db \
    --admin-username=admin \
    --admin-email=admin@example.com \
    --project-name="demo TYPO3 site" \
    --force
  assert_success

  run bats_pipe curl -sfL https://${PROJNAME}.ddev.site/ \| grep "Welcome to a default website made with"
  assert_success
  run bats_pipe curl s-sfL https://${PROJNAME}.ddev.site/typo3/ \| grep "TYPO3 CMS Login:"
  assert_success
}

# This test is for the future, when we have a v14 quickstart. For now, it's
# to ensure compatibility with upcoming v14
# bats test_tags=typo3-setup,t3v14
@test "TYPO3 v14 DEV 'ddev typo3 setup' composer test with $(ddev --version)" {
  PROJNAME=my-typo3-site
  run mkdir -p ${PROJNAME} && cd ${PROJNAME}
  assert_success

  run git clone https://github.com/TYPO3/TYPO3.CMS.BaseDistribution.git .
  assert_success

  run ddev config --project-type=typo3 --docroot=public --php-version=8.3
  assert_success

  run ddev start -y >/dev/null
  assert_success

  run ddev composer install >/dev/null
  assert_success

  run ddev exec touch public/FIRST_INSTALL
  assert_success

  run ddev typo3 setup \
    --admin-user-password="Demo123*" \
    --driver=mysqli \
    --create-site=https://${PROJNAME}.ddev.site \
    --server-type=other \
    --dbname=db \
    --username=db \
    --password=db \
    --port=3306 \
    --host=db \
    --admin-username=admin \
    --admin-email=admin@example.com \
    --project-name="demo TYPO3 site" \
    --force
  assert_success

  # Restart to get the additional.php settings in there
  run ddev restart -y
  assert_success

  ddev mutagen sync
  sleep 2

  run bats_pipe curl -sfL https://${PROJNAME}.ddev.site/ \| grep "Welcome to a default website made with"
  assert_success
  run bats_pipe curl -sfL https://${PROJNAME}.ddev.site/typo3/ \| grep "TYPO3 CMS Login:"
  assert_success

  # Now try it with /admin as the BE entrypoint
  echo '$GLOBALS["TYPO3_CONF_VARS"]["BE"]["entryPoint"] = "/admin";' >> config/system/additional.php
  run ddev mutagen sync
  sleep 2
  
  run curl -If https://${PROJNAME}.ddev.site/typo3/
  assert_failure
  run curl -If https://${PROJNAME}.ddev.site/admin/
  assert_success

  run bats_pipe curl -sfL https://${PROJNAME}.ddev.site/admin/ \| grep "TYPO3 CMS Login:"
  assert_success
}
