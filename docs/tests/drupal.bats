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

@test "Drupal 12 quickstart with $(ddev --version)" {
  _skip_test_if_needed "drupal12-composer"

  run mkdir my-drupal-site && cd my-drupal-site
  assert_success

  run ddev config --project-type=drupal12 --docroot=web
  assert_success

  run ddev start -y
  assert_success

  run ddev composer create-project drupal/recommended-project:main-dev@dev
  assert_success

  run ddev composer require drush/drush
  assert_success

  run ddev drush site:install --account-name=admin --account-pass=admin -y
  assert_success

  DDEV_DEBUG=true run ddev launch
  assert_output "FULLURL https://${PROJNAME}.ddev.site"
  assert_success

  # validate running project
  run curl -sfIv https://${PROJNAME}.ddev.site
  assert_output --partial "x-generator: Drupal 12 (https://www.drupal.org)"
  assert_output --partial "HTTP/2 200"
  assert_success
}

@test "Drupal 11 quickstart with $(ddev --version)" {
  _skip_test_if_needed "drupal11-composer"

  run mkdir my-drupal-site && cd my-drupal-site
  assert_success

  run ddev config --project-type=drupal11 --docroot=web
  assert_success

  run ddev start -y
  assert_success

  run ddev composer create-project "drupal/recommended-project:^11"
  assert_success

  run ddev composer require drush/drush
  assert_success

  run ddev drush site:install --account-name=admin --account-pass=admin -y
  assert_success

  DDEV_DEBUG=true run ddev launch
  assert_output "FULLURL https://${PROJNAME}.ddev.site"
  assert_success

  # validate running project
  run curl -sfIv https://${PROJNAME}.ddev.site
  assert_output --partial "x-generator: Drupal 11 (https://www.drupal.org)"
  assert_output --partial "HTTP/2 200"
  assert_success
}

@test "Drupal 10 quickstart with $(ddev --version)" {
  _skip_test_if_needed "drupal10-composer"

  run mkdir my-drupal-site && cd my-drupal-site
  assert_success

  run ddev config --project-type=drupal10 --docroot=web
  assert_success

  run ddev start -y
  assert_success

  run ddev composer create-project "drupal/recommended-project:^10"
  assert_success

  run ddev composer require drush/drush
  assert_success

  run ddev drush site:install --account-name=admin --account-pass=admin -y
  assert_success

  DDEV_DEBUG=true run ddev launch
  assert_output "FULLURL https://${PROJNAME}.ddev.site"
  assert_success

  # validate running project
  run curl -sfIv https://${PROJNAME}.ddev.site
  assert_output --partial "x-generator: Drupal 10 (https://www.drupal.org)"
  assert_output --partial "HTTP/2 200"
  assert_success
}

@test "Drupal 11 git based quickstart with $(ddev --version)" {
  _skip_test_if_needed "drupal11-git"

  PROJECT_GIT_URL=https://github.com/ddev/test-drupal11.git
  run git clone ${PROJECT_GIT_URL} ${PROJNAME}
  assert_success

  cd ${PROJNAME} || exit 2
  assert_success

  run ddev config --project-type=drupal11 --docroot=web
  assert_success

  run ddev start -y
  assert_success

  run ddev composer install
  assert_success

  run ddev drush site:install --account-name=admin --account-pass=admin -y
  assert_success

  DDEV_DEBUG=true run ddev launch
  assert_output "FULLURL https://${PROJNAME}.ddev.site"
  assert_success

  # validate running project
  run curl -sfIv https://${PROJNAME}.ddev.site
  assert_output --partial "x-generator: Drupal 11 (https://www.drupal.org)"
  assert_output --partial "HTTP/2 200"
  assert_success
}

@test "Drupal CMS composer quickstart with $(ddev --version)" {
  _skip_test_if_needed "drupal-cms-composer"

  run mkdir my-drupal-site && cd my-drupal-site
  assert_success

  run ddev config --project-type=drupal11 --docroot=web
  assert_success

  run ddev start -y
  assert_success

  run ddev composer create-project drupal/cms
  assert_success
  # This check shows that post-create-project-cmd event ran successfully
  assert_output --partial "Congratulations, you’ve installed Drupal CMS!"

  # Note: recipe-unpack runs automatically in DDEV v1.25.0+
  run ddev composer drupal:recipe-unpack
  assert_success
  assert_output --partial "No recipes to unpack."

  # Run Drush site install to set up the site
  run ddev drush si --account-name=admin --account-pass=admin -y
  assert_success

  DDEV_DEBUG=true run ddev launch
  assert_output "FULLURL https://${PROJNAME}.ddev.site"
  assert_success

  # validate running project
  run curl -sfIv https://${PROJNAME}.ddev.site
  assert_output --partial "HTTP/2 200"
  assert_output --partial "expires: Sun, 19 Nov 1978 05:00:00 GMT"
  assert_output --partial "x-generator: Drupal 11 (https://www.drupal.org)"
  assert_success
  run curl -sfv https://${PROJNAME}.ddev.site
  assert_success
  assert_output --partial "Home | Drush Site-Install"
  assert_output --partial "Bring your site to life, starting here."
}
