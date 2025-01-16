#!/usr/bin/env bats

setup() {
  PROJNAME=my-drupal-site
  load 'common-setup'
  _common_setup
}

# executed after each test
teardown() {
  _common_teardown
}

@test "Drupal 11 quickstart with $(ddev --version)" {
  # mkdir my-drupal-site && cd my-drupal-site
  run mkdir my-drupal-site && cd my-drupal-site
  assert_success
  # ddev config --project-type=drupal11 --docroot=web
  run ddev config --project-type=drupal11 --docroot=web
  assert_success
  # ddev start
  run ddev start
  assert_success
  # ddev composer create drupal/recommended-project:^11
  run ddev composer create drupal/recommended-project:^11
  assert_success
  # ddev composer require drush/drush
  run ddev composer require drush/drush
  assert_success
  #ddev drush site:install --account-name=admin --account-pass=admin -y
  run ddev drush site:install --account-name=admin --account-pass=admin -y
  assert_success
  # ddev launch
  run bash -c "DDEV_DEBUG=true ddev launch"
  assert_output "FULLURL https://${PROJNAME}.ddev.site"
  assert_success
  # validate running project
  run curl -sfI https://${PROJNAME}.ddev.site
  assert_success
  assert_output --partial "x-generator: Drupal 11 (https://www.drupal.org)"
  assert_output --partial "HTTP/2 200"
}

@test "Drupal 10 quickstart with $(ddev --version)" {
  # mkdir my-drupal-site && cd my-drupal-site
  run mkdir my-drupal-site && cd my-drupal-site
  assert_success
  # ddev config --project-type=drupal10 --docroot=web
  run ddev config --project-type=drupal10 --docroot=web
  assert_success
  # ddev start
  run ddev start
  assert_success
  # ddev composer create drupal/recommended-project:^10
  run ddev composer create drupal/recommended-project:^10
  assert_success
  # ddev composer require drush/drush
  run ddev composer require drush/drush
  assert_success
  #ddev drush site:install --account-name=admin --account-pass=admin -y
  run ddev drush site:install --account-name=admin --account-pass=admin -y
  assert_success
  # ddev launch
  run bash -c "DDEV_DEBUG=true ddev launch"
  assert_output "FULLURL https://${PROJNAME}.ddev.site"
  assert_success
  # validate running project
  run curl -sfI https://${PROJNAME}.ddev.site
  assert_success
  assert_output --partial "x-generator: Drupal 10 (https://www.drupal.org)"
  assert_output --partial "HTTP/2 200"
}

@test "Drupal 11 git based quickstart with $(ddev --version)" {
  # PROJECT_GIT_URL=https://github.com/ddev/test-drupal11.git
  PROJECT_GIT_URL=https://github.com/ddev/test-drupal11.git
  # git clone ${PROJECT_GIT_URL} ${PROJNAME}
  run git clone ${PROJECT_GIT_URL} ${PROJNAME}
  assert_success
  # cd my-drupal-site
  cd ${PROJNAME} || exit 2
  assert_success
  # ddev config --project-type=drupal11 --docroot=web
  run ddev config --project-type=drupal11 --docroot=web
  assert_success
  # ddev start
  run ddev start
  assert_success
  # ddev composer install
  run ddev composer install
  assert_success
  #ddev drush site:install --account-name=admin --account-pass=admin -y
  run ddev drush site:install --account-name=admin --account-pass=admin -y
  assert_success
  # ddev launch
  run bash -c "DDEV_DEBUG=true ddev launch"
  assert_output "FULLURL https://${PROJNAME}.ddev.site"
  assert_success
  # validate running project
  run curl -sfI https://${PROJNAME}.ddev.site
  assert_success
  assert_output --partial "x-generator: Drupal 11 (https://www.drupal.org)"
  assert_output --partial "HTTP/2 200"
}

@test "Drupal CMS composer quickstart with $(ddev --version)" {
  # mkdir my-drupal-site && cd my-drupal-site
  run mkdir my-drupal-site && cd my-drupal-site
  assert_success
  # ddev config --project-type=drupal11 --docroot=web
  run ddev config --project-type=drupal11 --docroot=web
  assert_success
  # ddev start
  run ddev start
  assert_success
  # ddev composer create drupal/cms
  run ddev composer create drupal/cms
  assert_success
  # ddev launch
  run bash -c "DDEV_DEBUG=true ddev launch"
  assert_output "FULLURL https://${PROJNAME}.ddev.site"
  assert_success
  # validate running project
  run curl -sfI https://${PROJNAME}.ddev.site
  assert_success
  assert_output --partial "location: /core/install.php"
  assert_output --partial "HTTP/2 302"
  assert_output --partial "x-generator: Drupal 11 (https://www.drupal.org)"
}

@test "Drupal CMS zip file quickstart with $(ddev --version)" {
  skip "Skipping until script doesn't erroneously create a -1 on project name"
  # CMS_VERSION=1.0.0
  CMS_VERSION=1.0.0
  # curl -o my-drupal-site.zip -fL https://ftp.drupal.org/files/projects/cms-1.0.0-${CMS_VERSION}.zip
  run curl -o my-drupal-site.zip -fL https://ftp.drupal.org/files/projects/cms-${CMS_VERSION}.zip
  assert_success
  # unzip my-drupal-cms-zip.zip && rm my-drupal-cms-zip.zip
  run unzip my-drupal-site.zip && rm my-drupal-site.zip
  assert_success
  # mv drupal-cms my-drupal-site
  # (Not contained in quickstart but necessary to use PROJNAME in this test )
  run mv drupal-cms my-drupal-site
  assert_success
  # Change directory
  cd ${tmpdir}/${PROJNAME}
  assert_success
  # execute launch script
  run bash -c "DDEV_DEBUG=true ./launch-drupal-cms.sh"
  assert_success
  # ddev launch
  run bash -c "DDEV_DEBUG=true ddev launch"
  assert_output "FULLURL https://${PROJNAME}.ddev.site"
  assert_success
  # validate running project
  run curl -sfI https://${PROJNAME}.ddev.site
  assert_success
  assert_output --partial "location: /core/install.php"
  assert_output --partial "HTTP/2 302"
  assert_output --partial "x-generator: Drupal 11 (https://www.drupal.org)"
}
