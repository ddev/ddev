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

  run mkdir -p ${PROJNAME} && cd ${PROJNAME}
  assert_success

  run ddev config --project-type=modx
  assert_success

  run ddev start -y
  assert_success

  _extra_info

  # Install MODX Revolution via Composer (resolves to the latest stable release)
  run ddev composer create-project modx/revolution
  assert_success

  # Perform a fresh install using the DDEV database credentials. The four
  # --context_* arguments are the installer's own defaults; passing them
  # explicitly keeps the install non-interactive.
  run ddev php setup/cli-install.php \
    --database_server=db --database=db --database_user=db --database_password=db \
    --table_prefix=modx_ --http_host=${DDEV_HOSTNAME} \
    --cmsadmin=admin --cmspassword=Admin123! --cmsadminemail=admin@example.com --language=en \
    --context_mgr_path=/var/www/html/manager/ --context_mgr_url=/manager/ \
    --context_connectors_path=/var/www/html/connectors/ --context_connectors_url=/connectors/
  assert_success

  # validate ddev launch
  DDEV_DEBUG=true run ddev launch /manager/
  assert_output "FULLURL ${PRIMARY_URL}/manager/"
  assert_success

  # validate running project
  run curl -sfIv ${PRIMARY_URL}
  assert_output --partial "server: nginx"
  assert_output --partial "HTTP/2 200"
  assert_success

  # Check the frontend renders the default MODX resource
  run curl -sfv ${PRIMARY_URL}
  assert_output --partial "<title>Home - MODX Revolution</title>"
  assert_output --partial "You have successfully installed MODX Revolution"
  assert_success

  # Check the MODX manager (backend) login page is working
  run curl -sfv ${PRIMARY_URL}/manager/
  assert_output --partial "<body id=\"login\">"
  assert_output --partial "alt=\"MODX CMS/CMF\""
  assert_output --partial "id=\"modx-login-form\""
  assert_success
}

@test "MODX ZIP Download quickstart with $(ddev --version)" {
  PROJNAME=my-modx-zip-site

  run mkdir -p ${PROJNAME} && cd ${PROJNAME}
  assert_success

  # MODX has no GitHub releases, so derive the latest production release ("-pl")
  # from the tags and download the traditional distribution zip.
  MODX_VERSION=$(_curl_github -fsSL https://api.github.com/repos/modxcms/revolution/tags | grep -o '"v[0-9][0-9.]*-pl"' | head -1 | tr -dc '0-9.')
  echo "# MODX_VERSION=${MODX_VERSION}" >&3
  [ -n "${MODX_VERSION}" ]

  run curl -fLo modx.zip "https://modx.s3.amazonaws.com/releases/${MODX_VERSION}/modx-${MODX_VERSION}-pl.zip"
  assert_success

  run unzip -q modx.zip
  assert_success
  rm -f modx.zip

  # The archive extracts into a versioned subdirectory; move its contents up
  run bash -c "shopt -s dotglob && mv 'modx-${MODX_VERSION}-pl'/* . && rmdir 'modx-${MODX_VERSION}-pl'"
  assert_success
  assert_file_exist setup/cli-install.php

  run ddev config --project-type=modx
  assert_success

  run ddev start -y
  assert_success

  _extra_info

  # Perform a fresh install using the DDEV database credentials. The four
  # --context_* arguments are the installer's own defaults; passing them
  # explicitly keeps the install non-interactive.
  run ddev php setup/cli-install.php \
    --database_server=db --database=db --database_user=db --database_password=db \
    --table_prefix=modx_ --http_host=${DDEV_HOSTNAME} \
    --cmsadmin=admin --cmspassword=Admin123! --cmsadminemail=admin@example.com --language=en \
    --context_mgr_path=/var/www/html/manager/ --context_mgr_url=/manager/ \
    --context_connectors_path=/var/www/html/connectors/ --context_connectors_url=/connectors/
  assert_success

  # validate ddev launch
  DDEV_DEBUG=true run ddev launch /manager/
  assert_output "FULLURL ${PRIMARY_URL}/manager/"
  assert_success

  # validate running project
  run curl -sfIv ${PRIMARY_URL}
  assert_output --partial "server: nginx"
  assert_output --partial "HTTP/2 200"
  assert_success

  # Check the frontend renders the default MODX resource
  run curl -sfv ${PRIMARY_URL}
  assert_output --partial "<title>Home - MODX Revolution</title>"
  assert_output --partial "You have successfully installed MODX Revolution"
  assert_success

  # Check the MODX manager (backend) login page is working
  run curl -sfv ${PRIMARY_URL}/manager/
  assert_output --partial "<body id=\"login\">"
  assert_output --partial "alt=\"MODX CMS/CMF\""
  assert_output --partial "id=\"modx-login-form\""
  assert_success
}
