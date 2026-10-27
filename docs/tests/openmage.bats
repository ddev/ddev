#!/usr/bin/env bats

setup() {
  PROJNAME=my-openmage-site
  load 'common-setup'
  _common_setup
}

# executed after each test
teardown() {
  _common_teardown
}

# Developer mode turns PHP 8.4+ #[Deprecated] notices into an error page,
# which only shows up after login
_assert_admin_dashboard() {
  local jar="${BATS_TEST_TMPDIR}/cookies" form_key
  form_key=$(curl -sSf -c "${jar}" "${PRIMARY_URL}/index.php/admin/" | grep -o 'name="form_key"[^>]*value="[^"]*"' | sed 's/.*value="//;s/"$//')
  _assert_page_contains -L "-b${jar}" "-c${jar}" "-dform_key=${form_key}" \
    "-dlogin[username]=admin" "-dlogin[password]=veryl0ngpassw0rd" \
    ${PRIMARY_URL}/index.php/admin/ \
    "<title>Dashboard / OpenMage Admin</title>"
}

@test "OpenMage git based quickstart with $(ddev --version)" {
  PROJNAME=my-openmage-git-site

  run -0 mkdir -p ${PROJNAME}
  cd ${PROJNAME}

  run -0 git clone --depth=1 https://github.com/OpenMage/magento-lts .

  run -0 ddev config --project-type=magento --php-version=8.3 --web-environment-add=MAGE_IS_DEVELOPER_MODE=1

  run -0 ddev start -y

  _extra_info

  run -0 ddev composer install

  # Silent OpenMage install with sample data
  run -0 ddev openmage-install -q

  DDEV_DEBUG=true run -0 ddev launch
  assert_output "FULLURL ${PRIMARY_URL}"

  # validate running project
  run -0 curl -sSfI ${PRIMARY_URL}
  assert_output --partial "server: nginx"
  assert_output --partial "HTTP/2 200"
  assert_output --partial "set-cookie: om_frontend"

  # Check if the frontend is working
  _assert_page_contains ${PRIMARY_URL} \
    "<title>Madison Island</title>" \
    "<meta name=\"keywords\" content=\"Magento, Varien, E-commerce\" />"

  # Check if the admin is working
  _assert_page_contains ${PRIMARY_URL}/index.php/admin/ "Log into OpenMage Admin Page"

  _assert_admin_dashboard
}

@test "OpenMage composer based quickstart with $(ddev --version)" {
  PROJNAME=my-openmage-composer-site

  run -0 mkdir -p ${PROJNAME}
  cd ${PROJNAME}

  run -0 ddev config --project-type=magento --docroot=public_test --php-version=8.3 --web-environment-add=MAGE_IS_DEVELOPER_MODE=1

  run -0 ddev start -y

  _extra_info

  # init project
  run -0 ddev composer init --name "openmage/composer-test" --description "OpenMage starter project" --type "project" -l "OSL-3.0" -s "dev" -q

  run -0 ddev composer config prefer-stable true

  run -0 ddev composer config policy.advisories.block false

  # set extra config
  run -0 ddev composer config extra.magento-root-dir "public_test"

  run -0 ddev composer config extra.magento-core-package-type "magento-source"

  # set allowed plugins
  run -0 ddev composer config allow-plugins.vaimo/composer-patches true

  run -0 ddev composer config allow-plugins.magento-hackathon/magento-composer-installer true

  run -0 ddev composer config allow-plugins.aydin-hassan/magento-core-composer-installer true

  run -0 ddev composer config allow-plugins.openmage/composer-plugin true

  # set requirements
  run -0 ddev composer require --no-update "aydin-hassan/magento-core-composer-installer":"^2.1.0" "openmage/magento-lts":"^20.19"

  # download OpenMage install command
  run -0 ddev exec wget -O .ddev/commands/web/openmage-install https://raw.githubusercontent.com/OpenMage/magento-lts/refs/heads/main/.ddev/commands/web/openmage-install

  run -0 ddev composer install

  # Silent OpenMage install with sample data
  run -0 ddev openmage-install -q

  DDEV_DEBUG=true run -0 ddev launch
  assert_output "FULLURL ${PRIMARY_URL}"

  # validate running project
  run -0 curl -sSfI ${PRIMARY_URL}
  assert_output --partial "server: nginx"
  assert_output --partial "HTTP/2 200"
  assert_output --partial "set-cookie: om_frontend"

  # Check if the frontend is working
  _assert_page_contains ${PRIMARY_URL} \
    "<title>Madison Island</title>" \
    "<meta name=\"keywords\" content=\"Magento, Varien, E-commerce\" />"

  # Check if the admin is working
  _assert_page_contains ${PRIMARY_URL}/index.php/admin/ "Log into OpenMage Admin Page"

  _assert_admin_dashboard
}
