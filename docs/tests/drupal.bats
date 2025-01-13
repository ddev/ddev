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
  run ddev config --project-type=drupal11 --docroot=web --project-name=${PROJNAME}
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
# Test for the git based quickstart. Has to be uncommented
# as soon as the Drupal test repo is created on GitHub
#@test "Drupal 11 git based quickstart with $(ddev --version)" {
#  # mkdir my-drupal-site && cd my-drupal-site
#  run mkdir my-drupal-site && cd my-drupal-site
#  assert_success
#  # PROJECT_GIT_URL=https://github.com/ddev/test-drupal.git
#  PROJECT_GIT_URL=https://github.com/ddev/test-drupal.git
#  # git clone ${PROJECT_GIT_URL} my-backdrop-site
#  run git clone ${PROJECT_GIT_URL} .
#  assert_success
#  # ddev config --project-type=drupal10 --docroot=web
#  run ddev config
#  assert_success
#  # ddev composer create drupal/recommended-project:^10
#  run ddev composer install
#  assert_success
#  # ddev launch
#  run bash -c "DDEV_DEBUG=true ddev launch"
#  assert_output "FULLURL https://${PROJNAME}.ddev.site"
#  assert_success
#  # validate running project
#  run curl -sfI https://${PROJNAME}.ddev.site
#  assert_success
#  assert_output --partial "x-generator: Drupal 10 (https://www.drupal.org)"
#  assert_output --partial "HTTP/2 200"
#}

@test "Drupal CMS composer quickstart with $(ddev --version)" {
  # mkdir my-drupal-site && cd my-drupal-site
  run mkdir my-drupal-site && cd my-drupal-site
  assert_success
  # ddev config --project-type=drupal11 --docroot=web
  run ddev config --project-type=drupal11 --docroot=web
  assert_success
  # ddev composer create --stability="RC" drupal/cms
  run ddev composer create --stability="RC" drupal/cms
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
  # curl -o my-drupal-site.zip -fL https://ftp.drupal.org/files/projects/cms-1.0.0-rc2.zip
  run curl -o my-drupal-site.zip -fL https://ftp.drupal.org/files/projects/cms-1.0.0-rc2.zip
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
  run DDEV_DEBUG=true ./launch-drupal-cms.sh
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
