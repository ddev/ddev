#!/usr/bin/env bats

setup() {
  PROJNAME=my-magento-site
  load 'common-setup'
  _common_setup
}

# executed after each test
teardown() {
  _common_teardown
}

@test "Magento 2 quickstart with $(ddev --version)" {
  if [ "${MAGENTO2_PUBLIC_ACCESS_KEY}" = "" ]; then
    skip "MAGENTO2_PUBLIC_ACCESS_KEY not provided (forked PR)"
  fi

  run -0 mkdir -p ${PROJNAME} && cd ${PROJNAME}

  run -0 ddev config --project-type=magento2 --docroot=pub --upload-dirs=media --disable-settings-management

  mkdir -p ./.ddev/homeadditions/.composer

  # add the env variable credentials to auth.json
  cat <<EOF > .ddev/homeadditions/.composer/auth.json
{
    "http-basic": {
        "repo.magento.com": {
            "username": "${MAGENTO2_PUBLIC_ACCESS_KEY}",
            "password": "${MAGENTO2_PRIVATE_ACCESS_KEY}"
        }
    }
}
EOF

  run -0 ddev add-on get ddev/ddev-opensearch

  run -0 ddev start -y

  _extra_info

  run -0 ddev composer create-project --repository https://repo.magento.com/ magento/project-community-edition

  # Copy the auth.json into var/composer_home for the deploying the sample data sep
  run -0 ddev exec "mkdir -p var/composer_home && cp ~/.composer/auth.json var/composer_home/auth.json"

  run -0 rm -f app/etc/env.php
  assert_file_not_exist app/etc/env.php

  run -0 ddev magento setup:install --base-url="${PRIMARY_URL}/" \
      --cleanup-database --db-host=db --db-name=db --db-user=db --db-password=db \
      --opensearch-host=opensearch --search-engine=opensearch --opensearch-port=9200 \
      --admin-firstname=Magento --admin-lastname=User --admin-email=user@example.com \
      --admin-user=admin --admin-password=Password123 --language=en_US

  run -0 ddev magento deploy:mode:set developer

  run -0 ddev magento module:disable Magento_TwoFactorAuth Magento_AdminAdobeImsTwoFactorAuth

  run -0 ddev config --disable-settings-management=false

  run -0 ddev magento setup:config:set --backend-frontname="admin_ddev" --no-interaction

  # Add sample data for Magento
  run -0 ddev magento sampledata:deploy

  # Re-run the setup script to include the sample data
  run -0 ddev magento setup:upgrade

  # validate ddev launch
  DDEV_DEBUG=true run -0 ddev launch /admin_ddev
  assert_output "FULLURL ${PRIMARY_URL}/admin_ddev"
  # validate running project
  run -0 curl -sSfI ${PRIMARY_URL}
  assert_output --partial "server: nginx"
  assert_output --partial "HTTP/2 200"
  _assert_page_contains ${PRIMARY_URL} \
    "Copyright © 2013-present Magento, Inc. All rights reserved." \
    "Here is what\`s trending on Luma right now" \
    "title=\"Argus All-Weather Tank\""
  run -0 curl -sSfI ${PRIMARY_URL}/index.php/admin_ddev/
  assert_output --partial "server: nginx"
  assert_output --partial "HTTP/2 200"
  _assert_page_contains ${PRIMARY_URL}/index.php/admin_ddev/ "Copyright &copy; $(date +%Y) Magento Commerce Inc. All rights reserved."
  # OpenSearch Dashboards is exposed on its own port, independent of router_https_port
  _assert_page_contains https://${DDEV_HOSTNAME}:5602/app/home#/ "<title>OpenSearch Dashboards</title>"
}
