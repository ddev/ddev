#!/usr/bin/env bats

setup() {
  PROJNAME=my-modx-site
  load 'common-setup'
  _common_setup
}

# executed after each test
teardown() {
  _common_teardown
}

@test "MODX Composer quickstart with $(ddev --version)" {
  PROJNAME=my-modx-composer-site

  run -0 mkdir -p ${PROJNAME}
  cd ${PROJNAME}

  run -0 ddev config --project-type=modx

  run -0 ddev start -y

  _extra_info

  # Install MODX Revolution via Composer (resolves to the latest stable release)
  run -0 _retry ddev composer create-project modx/revolution

  # Perform a fresh install using the DDEV database credentials. The four
  # --context_* arguments are the installer's own defaults; passing them
  # explicitly keeps the install non-interactive.
  run -0 ddev php setup/cli-install.php \
    --database_server=db --database=db --database_user=db --database_password=db \
    --table_prefix=modx_ --http_host=${DDEV_HOSTNAME} \
    --cmsadmin=admin --cmspassword=Admin123! --cmsadminemail=admin@example.com --language=en \
    --context_mgr_path=/var/www/html/manager/ --context_mgr_url=/manager/ \
    --context_connectors_path=/var/www/html/connectors/ --context_connectors_url=/connectors/

  # validate ddev launch
  DDEV_DEBUG=true run -0 ddev launch /manager/
  assert_output "FULLURL ${PRIMARY_URL}/manager/"

  # validate running project
  run -0 curl -sSfI ${PRIMARY_URL}
  assert_output --partial "server: nginx"
  assert_output --partial "HTTP/2 200"

  # Check the frontend renders the default MODX resource
  _assert_page_contains ${PRIMARY_URL} \
    "<title>Home - MODX Revolution</title>" \
    "You have successfully installed MODX Revolution"

  # Check the MODX manager (backend) login page is working
  _assert_page_contains ${PRIMARY_URL}/manager/ \
    "<body id=\"login\">" \
    "alt=\"MODX CMS/CMF\"" \
    "id=\"modx-login-form\""
}

@test "MODX ZIP Download quickstart with $(ddev --version)" {
  PROJNAME=my-modx-zip-site

  run -0 mkdir -p ${PROJNAME}
  cd ${PROJNAME}

  # MODX has no GitHub releases, so derive the latest production release ("-pl")
  # from the tags and download the traditional distribution zip.
  # git ls-remote avoids the GitHub API's rate limit and 5xx responses.
  MODX_VERSION=$(git ls-remote --tags --refs https://github.com/modxcms/revolution 'v*-pl' | sed -E 's|.*refs/tags/v([0-9.]+)-pl$|\1|' | sort -V | tail -1)
  echo "# MODX_VERSION=${MODX_VERSION}" >&3
  [ -n "${MODX_VERSION}" ]

  run -0 curl -fLo modx.zip "https://modx.s3.amazonaws.com/releases/${MODX_VERSION}/modx-${MODX_VERSION}-pl.zip"

  run -0 unzip -q modx.zip
  rm -f modx.zip

  # The archive extracts into a versioned subdirectory; move its contents up
  run -0 bash -c "shopt -s dotglob && mv 'modx-${MODX_VERSION}-pl'/* . && rmdir 'modx-${MODX_VERSION}-pl'"
  assert_file_exist setup/cli-install.php

  run -0 ddev config --project-type=modx

  run -0 ddev start -y

  _extra_info

  # Perform a fresh install using the DDEV database credentials. The four
  # --context_* arguments are the installer's own defaults; passing them
  # explicitly keeps the install non-interactive.
  run -0 ddev php setup/cli-install.php \
    --database_server=db --database=db --database_user=db --database_password=db \
    --table_prefix=modx_ --http_host=${DDEV_HOSTNAME} \
    --cmsadmin=admin --cmspassword=Admin123! --cmsadminemail=admin@example.com --language=en \
    --context_mgr_path=/var/www/html/manager/ --context_mgr_url=/manager/ \
    --context_connectors_path=/var/www/html/connectors/ --context_connectors_url=/connectors/

  # validate ddev launch
  DDEV_DEBUG=true run -0 ddev launch /manager/
  assert_output "FULLURL ${PRIMARY_URL}/manager/"

  # validate running project
  run -0 curl -sSfI ${PRIMARY_URL}
  assert_output --partial "server: nginx"
  assert_output --partial "HTTP/2 200"

  # Check the frontend renders the default MODX resource
  _assert_page_contains ${PRIMARY_URL} \
    "<title>Home - MODX Revolution</title>" \
    "You have successfully installed MODX Revolution"

  # Check the MODX manager (backend) login page is working
  _assert_page_contains ${PRIMARY_URL}/manager/ \
    "<body id=\"login\">" \
    "alt=\"MODX CMS/CMF\"" \
    "id=\"modx-login-form\""
}
