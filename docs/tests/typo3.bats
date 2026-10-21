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
  run -0 mkdir -p ${PROJNAME}
  cd ${PROJNAME}
  run -0 ddev config --project-type=typo3 --docroot=public
  run -0 ddev start -y >/dev/null

  _extra_info

  run -0 _retry ddev composer create-project "typo3/cms-base-distribution:^14" >/dev/null
  run -0 ddev composer require typo3/theme-camino >/dev/null

  run -0 ddev typo3 setup \
    --admin-user-password="Demo123*" \
    --driver=mysqli \
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

  DDEV_DEBUG=true run -0 ddev launch
  assert_output --partial "FULLURL ${PRIMARY_URL}"

  run -0 curl -sSfI "${PRIMARY_URL}/camino/"
  assert_output --partial "HTTP/2 200"
  _assert_page_contains -L "${PRIMARY_URL}/camino/" "Camino de Compostela"
  _assert_page_contains -L "${PRIMARY_URL}/typo3/" "TYPO3 CMS Login:"
}

@test "TYPO3 v13 'ddev typo3 setup' composer test with $(ddev --version)" {
  PROJNAME=my-typo3-site
  run -0 mkdir -p ${PROJNAME}
  cd ${PROJNAME}
  run -0 ddev config --project-type=typo3 --docroot=public
  run -0 ddev start -y >/dev/null
  _extra_info

  run -0 _retry ddev composer create-project "typo3/cms-base-distribution:^13" >/dev/null

  run -0 ddev typo3 setup \
    --admin-user-password="Demo123*" \
    --driver=mysqli \
    --create-site=${PRIMARY_URL} \
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

  DDEV_DEBUG=true run -0 ddev launch
  assert_output --partial "FULLURL ${PRIMARY_URL}"

  run -0 bats_pipe curl -sSfI ${PRIMARY_URL}/ \| grep "HTTP/2 200"
  run -0 bats_pipe curl -sSfL ${PRIMARY_URL}/ \| grep "Welcome to a default website made with <a href=\"https://typo3.org\">TYPO3</a>"
  run -0 bats_pipe curl -sSfL ${PRIMARY_URL}/typo3/ \| grep "TYPO3 CMS Login:"
}

@test "TYPO3 v12 'ddev typo3 setup' composer test with $(ddev --version)" {
  PROJNAME=my-typo3-site
  run -0 mkdir -p ${PROJNAME}
  cd ${PROJNAME}
  run -0 ddev config --project-type=typo3 --docroot=public --php-version=8.1
  run -0 ddev start -y >/dev/null
  _extra_info

  run -0 _retry ddev composer create-project "typo3/cms-base-distribution:^12" >/dev/null

  run -0 ddev typo3 setup \
    --admin-user-password="Demo123*" \
    --driver=mysqli \
    --create-site=${PRIMARY_URL} \
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

  DDEV_DEBUG=true run -0 ddev launch
  assert_output --partial "FULLURL ${PRIMARY_URL}"

  run -0 bats_pipe curl -sSfI ${PRIMARY_URL}/ \| grep "HTTP/2 200"
  run -0 bats_pipe curl -sSfL ${PRIMARY_URL}/ \| grep "Welcome to a default website made with <a href=\"https://typo3.org\">TYPO3</a>"
  run -0 bats_pipe curl -sSfL ${PRIMARY_URL}/typo3/ \| grep "TYPO3 CMS Login:"
}

@test "TYPO3 v11 'web installer' composer test with $(ddev --version)" {
  PROJNAME=my-typo3-site
  run -0 mkdir -p ${PROJNAME}
  cd ${PROJNAME}
  run -0 ddev config --project-type=typo3 --docroot=public --php-version=8.1
  run -0 ddev start -y >/dev/null
  _extra_info

  run -0 _retry ddev composer create-project "typo3/cms-base-distribution:^11" >/dev/null
  run -0 ddev exec touch public/FIRST_INSTALL

  DDEV_DEBUG=true run -0 ddev launch /typo3/install.php
  assert_output --partial "FULLURL ${PRIMARY_URL}/typo3/install.php"

  run -0 bats_pipe curl -sSfI ${PRIMARY_URL}/typo3/install.php \| grep "HTTP/2 200"
  run -0 bats_pipe curl -sSfL ${PRIMARY_URL}/typo3/install.php \| grep "data-init=\"TYPO3/CMS/Install/Installer\""
}

@test "TYPO3 git based quickstart with $(ddev --version)" {
  PROJECT_GIT_URL=https://github.com/ddev/test-typo3.git
  PROJNAME=my-typo3-site
  run -0 git clone ${PROJECT_GIT_URL} ${PROJNAME}
  cd ${PROJNAME} || exit 2
  run -0 ddev config --project-type=typo3 --docroot=public
  run -0 ddev start -y >/dev/null

  _extra_info

  run -0 ddev composer install >/dev/null
  run -0 ddev exec touch public/FIRST_INSTALL
  DDEV_DEBUG=true run -0 ddev launch
  assert_output --partial "FULLURL ${PRIMARY_URL}"
  # validate running project
  run -0 curl -sSfI ${PRIMARY_URL}/typo3/install.php
  assert_output --partial "content-security-policy: default-src 'self'; script-src 'self'"
  assert_output --partial "HTTP/2 200"
}

@test "TYPO3 XHGui composer test with $(ddev --version)" {
  run -0 mkdir -p my-typo3-site
  cd my-typo3-site

  run -0 ddev config --project-type=typo3 --docroot=public --xhprof-mode=xhgui

  run -0 ddev start -y >/dev/null

  _extra_info

  run -0 _retry ddev composer create-project typo3/cms-base-distribution:^13 >/dev/null

  run -0 ddev exec touch public/FIRST_INSTALL

  run -0 ddev xhgui on
  assert_output --partial "Started optional compose profiles"
  assert_output --partial "xhgui"

  # Ensure XHGui answers, so the grep failure below means there's no profiling data link
  run -0 _retry ddev exec "curl -sf xhgui:80 -o /dev/null"
  run ddev exec "curl -s xhgui:80 | grep -q '<a href=\"/?server_name=${DDEV_HOSTNAME}\">'"
  assert_failure

  # Profile site
  run -0 curl -sSfI ${PRIMARY_URL}/typo3/install.php
  assert_output --partial "HTTP/2 200"

  _assert_page_contains ${PRIMARY_URL}/typo3/install.php "Installing TYPO3 CMS"

  # wait for XHGui
  sleep 5

  # Ensure there a profiling data link
  run -0 ddev exec "curl -s xhgui:80 | grep -q '<a href=\"/?server_name=${DDEV_HOSTNAME}\">'"
}
