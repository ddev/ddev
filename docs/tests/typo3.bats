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

@test "TYPO3 v14 'ddev typo3 setup' composer test with $(ddev --version)" {
  PROJNAME=my-typo3-site
  run mkdir -p ${PROJNAME} && cd ${PROJNAME}
  assert_success
  run ddev config --project-type=typo3 --docroot=public
  assert_success
  run ddev start -y >/dev/null
  assert_success
  run ddev composer create-project "typo3/cms-base-distribution:^14" >/dev/null
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
    --project-name="My TYPO3 site" \
    --force
  assert_success

  DDEV_DEBUG=true run ddev launch
  assert_output --partial "FULLURL https://${PROJNAME}.ddev.site"
  assert_success

  run curl -sfIv "https://${PROJNAME}.ddev.site/"
  assert_output --partial "HTTP/2 200"
  assert_success
  run curl -sfLv "https://${PROJNAME}.ddev.site/"
  assert_output --partial "Welcome to a default website made with "
  assert_success
  run curl -sfLv "https://${PROJNAME}.ddev.site/typo3/"
  assert_output --partial "TYPO3 CMS Login:"
  assert_success
}

@test "TYPO3 v13 'ddev typo3 setup' composer test with $(ddev --version)" {
  PROJNAME=my-typo3-site
  run mkdir -p ${PROJNAME} && cd ${PROJNAME}
  assert_success
  run ddev config --project-type=typo3 --docroot=public
  assert_success
  run ddev start -y >/dev/null
  assert_success
  run ddev composer create-project "typo3/cms-base-distribution:^13" >/dev/null
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
    --project-name="My TYPO3 site" \
    --force
  assert_success

  DDEV_DEBUG=true run ddev launch
  assert_output --partial "FULLURL https://${PROJNAME}.ddev.site"
  assert_success

  run bats_pipe curl -sfIv https://${PROJNAME}.ddev.site/ \| grep "HTTP/2 200"
  assert_success
  run bats_pipe curl -sfLv https://${PROJNAME}.ddev.site/ \| grep "Welcome to a default website made with <a href=\"https://typo3.org\">TYPO3</a>"
  assert_success
  run bats_pipe curl -sfLv https://${PROJNAME}.ddev.site/typo3/ \| grep "TYPO3 CMS Login:"
  assert_success
}

@test "TYPO3 v12 'ddev typo3 setup' composer test with $(ddev --version)" {
  PROJNAME=my-typo3-site
  run mkdir -p ${PROJNAME} && cd ${PROJNAME}
  assert_success
  run ddev config --project-type=typo3 --docroot=public --php-version=8.1
  assert_success
  run ddev start -y >/dev/null
  assert_success
  run ddev composer create-project "typo3/cms-base-distribution:^12" >/dev/null
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
    --project-name="My TYPO3 site" \
    --force
  assert_success

  DDEV_DEBUG=true run ddev launch
  assert_output --partial "FULLURL https://${PROJNAME}.ddev.site"
  assert_success

  run bats_pipe curl -sfIv https://${PROJNAME}.ddev.site/ \| grep "HTTP/2 200"
  assert_success
  run bats_pipe curl -sfLv https://${PROJNAME}.ddev.site/ \| grep "Welcome to a default website made with <a href=\"https://typo3.org\">TYPO3</a>"
  assert_success
  run bats_pipe curl -sfLv https://${PROJNAME}.ddev.site/typo3/ \| grep "TYPO3 CMS Login:"
  assert_success
}

@test "TYPO3 v11 'web installer' composer test with $(ddev --version)" {
  PROJNAME=my-typo3-site
  run mkdir -p ${PROJNAME} && cd ${PROJNAME}
  assert_success
  run ddev config --project-type=typo3 --docroot=public --php-version=8.1
  assert_success
  run ddev start -y >/dev/null
  assert_success
  run ddev composer create-project "typo3/cms-base-distribution:^11" >/dev/null
  assert_success
  run ddev exec touch public/FIRST_INSTALL
  assert_success

  DDEV_DEBUG=true run ddev launch /typo3/install.php
  assert_output --partial "FULLURL https://${PROJNAME}.ddev.site/typo3/install.php"
  assert_success

  run bats_pipe curl -sfIv https://${PROJNAME}.ddev.site/typo3/install.php \| grep "HTTP/2 200"
  assert_success
  run bats_pipe curl -sfLv https://${PROJNAME}.ddev.site/typo3/install.php \| grep "data-init=\"TYPO3/CMS/Install/Installer\""
  assert_success
}

@test "TYPO3 git based quickstart with $(ddev --version)" {
  PROJECT_GIT_URL=https://github.com/ddev/test-typo3.git
  PROJNAME=my-typo3-site
  run git clone ${PROJECT_GIT_URL} ${PROJNAME}
  assert_success
  cd ${PROJNAME} || exit 2
  assert_success
  run ddev config --project-type=typo3 --docroot=public
  assert_success
  run ddev start -y >/dev/null
  assert_success
  run ddev composer install >/dev/null
  assert_success
  run ddev exec touch public/FIRST_INSTALL
  assert_success
  DDEV_DEBUG=true run ddev launch
  assert_output --partial "FULLURL https://${PROJNAME}.ddev.site"
  assert_success
  # validate running project
  run curl -sfIv https://${PROJNAME}.ddev.site/typo3/install.php
  assert_output --partial "content-security-policy: default-src 'self'; script-src 'self'"
  assert_output --partial "HTTP/2 200"
  assert_success
}

@test "TYPO3 XHGui composer test with $(ddev --version)" {
  run mkdir my-typo3-site && cd my-typo3-site
  assert_success

  run ddev config --project-type=typo3 --docroot=public --xhprof-mode=xhgui
  assert_success

  run ddev start -y >/dev/null
  assert_success

  run ddev composer create-project typo3/cms-base-distribution:^13 >/dev/null
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
  run curl -sfIv https://${PROJNAME}.ddev.site/typo3/install.php
  assert_output --partial "HTTP/2 200"
  assert_success

  run curl -sfv https://${PROJNAME}.ddev.site/typo3/install.php
  assert_output --partial "Installing TYPO3 CMS"
  assert_success

  # wait for XHGui
  sleep 5

  # Ensure there a profiling data link
  run ddev exec "curl -s xhgui:80 | grep -q '<a href=\"/?server_name=${PROJNAME}.ddev.site\">'"
  assert_success
}
