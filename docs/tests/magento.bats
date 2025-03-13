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
  # mkdir ${PROJNAME} && cd ${PROJNAME}
  run mkdir ${PROJNAME} && cd ${PROJNAME}
  assert_success
  # ddev config --project-type=magento2 --docroot=pub --upload-dirs=media --disable-settings-management
  run ddev config --project-type=magento2 --docroot=pub --upload-dirs=media --disable-settings-management
  assert_success

  # mkdir -p .ddev/homeadditions/.composer
  run mkdir -p ./.ddev/homeadditions/.composer
  assert_success

  # touch .ddev/homeadditions/.composer/auth.json
  run touch ./.ddev/homeadditions/.composer/auth.json
  assert_success

  # add the env variable credentials to auth.json
  run cat <<EOF > .ddev/homeadditions/.composer/auth.json
{
    "http-basic": {
        "repo.magento.com": {
            "username": "'"$COMPOSER_USERNAME"'",
            "password": "'"$COMPOSER_PASSWORD"'"
        }
    }
}
EOF
  assert_success

  # ddev add-on get ddev/ddev-elasticsearch
  run ddev add-on get ddev/ddev-elasticsearch
  assert_success

  # ddev start -y
  run ddev start -y
  assert_success

  # ddev composer create --repository https://repo.magento.com/ magento/project-community-edition
  run ddev composer create --repository https://repo.magento.com/ magento/project-community-edition
  assert_success

  # rm -f app/etc/env.php
  run rm -f app/etc/env.php
  assert_success

  # magento setup:install
  run ddev magento setup:install --base-url="https://${PROJNAME}.ddev.site/" \
      --cleanup-database --db-host=db --db-name=db --db-user=db --db-password=db \
      --elasticsearch-host=elasticsearch --search-engine=elasticsearch7 --elasticsearch-port=9200 \
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
