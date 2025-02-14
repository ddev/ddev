#!/usr/bin/env bats

setup() {
  PROJNAME=my-contao-site
  load 'common-setup'
  _common_setup
}

# executed after each test
teardown() {
  _common_teardown
}

@test "Contao Composer quickstart with $(ddev --version)" {
  # mkdir ${PROJNAME} && cd ${PROJNAME}
  run mkdir ${PROJNAME} && cd ${PROJNAME}
  assert_success
  # ddev config --project-type=php --docroot=public --webserver-type=apache-fpm --php-version=8.2
  run ddev config --project-type=php --docroot=public --webserver-type=apache-fpm --php-version=8.2
  assert_success
  # ddev composer create contao/managed-edition:5.3
  run ddev composer create contao/managed-edition:5.3
  assert_success
  # Set DATABASE_URL and MAILER_DSN in .env.local
  # ddev dotenv set .env.local --database-url=mysql://db:db@db:3306/db --mailer-dsn=smtp://localhost:1025
  run ddev dotenv set .env.local --database-url=mysql://db:db@db:3306/db --mailer-dsn=smtp://localhost:1025
  assert_success
  # Create the database
  # ddev exec contao-console contao:migrate --no-interaction
  run ddev exec contao-console contao:migrate --no-interaction
  assert_success
  # Create backend user
  # ddev exec contao-console contao:user:create --username=admin --name=Administrator --email=admin@example.com --language=en --password=Password123 --admin
  run ddev exec contao-console contao:user:create --username=admin --name=Administrator --email=admin@example.com --language=en --password=Password123 --admin
  assert_success
  # ddev launch
  run bash -c "DDEV_DEBUG=true ddev launch contao"
  assert_output "FULLURL https://${PROJNAME}.ddev.site"
  assert_success
  # validate running project
  run curl -sfI https://${PROJNAME}.ddev.site/contao/login
  assert_success
  assert_output --partial "HTTP/2 200"
}

@test "Contao Manager quickstart with $(ddev --version)" {
  # mkdir ${PROJNAME} && cd ${PROJNAME}
  run mkdir ${PROJNAME} && cd ${PROJNAME}
  assert_success
  run ddev config --project-type=php --docroot=public --webserver-type=apache-fpm --php-version=8.2
  assert_success
  # set DATABASE_URL and MAILER_DSN in .env.local
  # ddev dotenv set .env.local --database-url=mysql://db:db@db:3306/db --mailer-dsn=smtp://localhost:1025
  run ddev dotenv set .env.local --database-url=mysql://db:db@db:3306/db --mailer-dsn=smtp://localhost:1025
  assert_success
  # ddev start
  run ddev start
  assert_success
  # ddev exec "wget -O public/contao-manager.phar.php https://download.contao.org/contao-manager/stable/contao-manager.phar"
  run ddev exec "wget -O public/contao-manager.phar.php https://download.contao.org/contao-manager/stable/contao-manager.phar"
  assert_success
  # ddev launch
  run bash -c "DDEV_DEBUG=true ddev launch contao-manager.phar.php"
  assert_output "FULLURL https://${PROJNAME}.ddev.site/contao-manager.phar.php"
  assert_success
  # validate running project
  run curl -sfI https://${PROJNAME}.ddev.site/contao-manager.phar.php
  assert_success
  assert_output --partial "HTTP/2 302"
  assert_output --partial "location: /contao-manager.phar.php/"
}
