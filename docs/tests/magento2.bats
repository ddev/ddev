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
    skip "MAGENTO_PUBLIC_ACCESS_KEY not provided (forked PR)"
  fi

  # mkdir ${PROJNAME} && cd ${PROJNAME}
  run mkdir ${PROJNAME} && cd ${PROJNAME}
  assert_success
  # ddev config --project-type=magento2 --docroot=pub --upload-dirs=media --disable-settings-management
  run ddev config --project-type=magento2 --docroot=pub --upload-dirs=media --disable-settings-management
  assert_success

  # mkdir -p .ddev/homeadditions/.composer
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

  # ddev add-on get ddev/ddev-elasticsearch
  run ddev add-on get ddev/ddev-elasticsearch
  assert_success

  # ddev start -y
  run ddev start -y
  assert_success

  # ddev composer create --repository https://repo.magento.com/ magento/project-community-edition
  run ddev composer create --repository https://repo.magento.com/ magento/project-community-edition
  assert_success

  # Copy docker compose yaml for Elastic Search 8
  run cp .ddev/elasticsearch/docker-compose.elasticsearch8.yaml .ddev/
  assert_success

  # make sure host and container are in sync after copy
  run ddev mutagen sync
  assert_success

  # rm -f app/etc/env.php
  run rm -f app/etc/env.php
  assert_success

  # magento setup:install
  run ddev magento setup:install --base-url="https://${PROJNAME}.ddev.site/" \
      --cleanup-database --db-host=db --db-name=db --db-user=db --db-password=db \
      --elasticsearch-host=elasticsearch --search-engine=elasticsearch8 --elasticsearch-port=9200 \
      --admin-firstname=Magento --admin-lastname=User --admin-email=user@example.com \
      --admin-user=admin --admin-password=Password123 --language=en_US
  assert_success

  # ddev magento deploy:mode:set developer
  run ddev magento deploy:mode:set developer
  assert_success

  # ddev magento module:disable Magento_TwoFactorAuth Magento_AdminAdobeImsTwoFactorAuth
  run ddev magento module:disable Magento_TwoFactorAuth Magento_AdminAdobeImsTwoFactorAuth
  assert_success

  # ddev config --disable-settings-management=false
  run ddev config --disable-settings-management=false
  assert_success

  # ddev magento setup:config:set --backend-frontname="admin_ddev" --no-interaction
  run ddev magento setup:config:set --backend-frontname="admin_ddev" --no-interaction
  assert_success

  # validate ddev launch
  run bash -c "DDEV_DEBUG=true ddev launch /admin_ddev"
  assert_output "FULLURL https://${PROJNAME}.ddev.site/admin_ddev"
  assert_success
  # validate running project
  run curl -sfI https://${PROJNAME}.ddev.site
  assert_success
  assert_output --partial "server: nginx"
  assert_output --partial "HTTP/2 200"
  run curl -sf https://${PROJNAME}.ddev.site
  assert_success
  assert_output --partial "Copyright © 2013-present Magento, Inc. All rights reserved."
  run curl -sfI https://${PROJNAME}.ddev.site/index.php/admin_ddev/
  assert_success
  assert_output --partial "server: nginx"
  assert_output --partial "HTTP/2 200"
  run curl -sf https://${PROJNAME}.ddev.site/index.php/admin_ddev/
  assert_success
  assert_output --partial "Copyright &copy; 2025 Magento Commerce Inc. All rights reserved."
}
