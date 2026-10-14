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
  PROJNAME=my-drupal12-site

  run -0 mkdir -p my-drupal12-site && cd my-drupal12-site

  run -0 ddev config --project-type=drupal12 --docroot=web

  run -0 ddev start -y

  _extra_info

  run -0 ddev composer create-project drupal/recommended-project:main-dev@dev

  run -0 ddev composer require drush/drush

  run -0 ddev drush site:install --account-name=admin --account-pass=admin -y

  DDEV_DEBUG=true run -0 ddev launch
  assert_line "FULLURL ${PRIMARY_URL}"

  # validate the return header of the running project
  run -0 curl -sI ${PRIMARY_URL}
  assert_output --partial "x-generator: Drupal 12 (https://www.drupal.org)"
  assert_output --partial "HTTP/2 403"
  assert_output --partial "expires: Sun, 19 Nov 1978 05:00:00 GMT"

  # validate the DOM of the running project
  run -0 curl -s ${PRIMARY_URL}
  assert_output --partial "<meta name=\"Generator\" content=\"Drupal 12"
  assert_output --partial "<h1 class=\"page-title\">Access denied</h1>"
}

@test "Drupal 11 quickstart with $(ddev --version)" {
  PROJNAME=my-drupal11-site

  run -0 mkdir -p my-drupal11-site && cd my-drupal11-site

  run -0 ddev config --project-type=drupal11 --docroot=web

  run -0 ddev start -y

  _extra_info

  run -0 ddev composer create-project "drupal/recommended-project"

  run -0 ddev composer require drush/drush

  run -0 ddev drush site:install --account-name=admin --account-pass=admin -y

  DDEV_DEBUG=true run -0 ddev launch
  assert_line "FULLURL ${PRIMARY_URL}"

  # validate running project
  run -0 curl -sSfI ${PRIMARY_URL}
  assert_output --partial "x-generator: Drupal 11 (https://www.drupal.org)"
  assert_output --partial "HTTP/2 200"
}

@test "Drupal 10 quickstart with $(ddev --version)" {
  PROJNAME=my-drupal10-site

  run -0 mkdir -p my-drupal10-site && cd my-drupal10-site

  run -0 ddev config --project-type=drupal10 --docroot=web

  run -0 ddev start -y

  _extra_info

  run -0 ddev composer create-project "drupal/recommended-project:^10"

  run -0 ddev composer require drush/drush

  run -0 ddev drush site:install --account-name=admin --account-pass=admin -y

  DDEV_DEBUG=true run -0 ddev launch
  assert_line "FULLURL ${PRIMARY_URL}"

  # validate running project
  run -0 curl -sSfI ${PRIMARY_URL}
  assert_output --partial "x-generator: Drupal 10 (https://www.drupal.org)"
  assert_output --partial "HTTP/2 200"
}

@test "Drupal 11 git based quickstart with $(ddev --version)" {
  PROJNAME=my-drupal11-site

  PROJECT_GIT_URL=https://github.com/ddev/test-drupal11.git
  run -0 git clone ${PROJECT_GIT_URL} ${PROJNAME}

  cd ${PROJNAME} || exit 2

  run -0 ddev config --project-type=drupal11 --docroot=web

  run -0 ddev start -y

  _extra_info

  run -0 ddev composer install

  run -0 ddev drush site:install --account-name=admin --account-pass=admin -y

  DDEV_DEBUG=true run -0 ddev launch
  assert_line "FULLURL ${PRIMARY_URL}"

  # validate running project
  run -0 curl -sSfI ${PRIMARY_URL}
  assert_output --partial "x-generator: Drupal 11 (https://www.drupal.org)"
  assert_output --partial "HTTP/2 200"
}

@test "Drupal CMS composer quickstart with $(ddev --version)" {
  PROJNAME=my-drupalcms-site

  run -0 mkdir -p my-drupalcms-site && cd my-drupalcms-site

  run -0 ddev config --project-type=drupal11 --docroot=web

  run -0 ddev start -y

  _extra_info

  run -0 ddev composer create-project drupal/cms
  # This check shows that post-create-project-cmd event ran successfully
  assert_output --partial "You're all set to get started with Drupal CMS"

  # Note: recipe-unpack runs automatically in DDEV v1.25.0+
  run -0 ddev composer drupal:recipe-unpack
  assert_output --partial "No recipes to unpack."

  # drush si makes the installer composer-require drupal_cms_helper mid-run, and
  # its davereid/drupal-environment dependency is not autoloadable in that process.
  run -0 ddev composer require drupal/drupal_cms_helper

  # Run Drush site install to set up the site
  run -0 ddev drush si --account-name=admin --account-pass=admin -y

  DDEV_DEBUG=true run ddev launch
  assert_line "FULLURL ${PRIMARY_URL}"

  # validate running project
  run -0 curl -sSfI ${PRIMARY_URL}
  assert_output --partial "HTTP/2 200"
  assert_output --partial "expires: Sun, 19 Nov 1978 05:00:00 GMT"
  assert_output --partial "x-generator: Drupal 11 (https://www.drupal.org)"
  _assert_page_contains ${PRIMARY_URL} \
    "Home | Drush Site-Install" \
    "Bring your site to life, starting here."
}
