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
  # ddev start -y
  run ddev start -y
  assert_success
  # ddev composer create-project "drupal/recommended-project:^11"
  run ddev composer create-project "drupal/recommended-project:^11"
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
  # ddev start -y
  run ddev start -y
  assert_success
  # ddev composer create-project "drupal/recommended-project:^10"
  run ddev composer create-project "drupal/recommended-project:^10"
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
  # ddev start -y
  run ddev start -y
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
  # ddev start -y
  run ddev start -y
  assert_success
  # ddev composer create-project drupal/cms
  run ddev composer create-project drupal/cms
  assert_success
  # Run Drush site install to set up the site
  run ddev drush si --account-name=admin --account-pass=admin -y
  assert_success
  # Apply one of the additional recipes Drupal CMS ships with
  run ddev drush recipe ../recipes/drupal_cms_blog
  assert_success
  # ddev launch
  run bash -c "DDEV_DEBUG=true ddev launch"
  assert_output "FULLURL https://${PROJNAME}.ddev.site"
  assert_success
  # validate running project
  run curl -sfI https://${PROJNAME}.ddev.site
  assert_success
  assert_output --partial "HTTP/2 200"
  assert_output --partial "expires: Sun, 19 Nov 1978 05:00:00 GMT"
  assert_output --partial "x-generator: Drupal 11 (https://www.drupal.org)"
  run curl -sf https://${PROJNAME}.ddev.site
  assert_success
  assert_output --partial "<title>Home | Drush Site-Install</title>"
  assert_output --partial "<span>A celebration of Drupal and open-source contributions</span>"
}
