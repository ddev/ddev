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
  PROJNAME=my-contao-composer-site

  run -0 mkdir -p ${PROJNAME} && cd ${PROJNAME}

  run -0 ddev config --project-type=php --docroot=public --webserver-type=apache-fpm --php-version=8.4

  run -0 ddev start -y

  _extra_info

  run -0 ddev composer create-project contao/managed-edition:5.7
  # Set DATABASE_URL and MAILER_DSN in .env.local
  run -0 ddev dotenv set .env.local --database-url=mysql://db:db@db:3306/db --mailer-dsn=smtp://localhost:1025
  # Create the database
  run -0 ddev exec contao-console contao:migrate --no-interaction
  # Create backend user
  run -0 ddev exec contao-console contao:user:create --username=admin --name=Administrator --email=admin@example.com --language=en --password=Password123 --admin
  DDEV_DEBUG=true run -0 ddev launch contao
  assert_output "FULLURL ${PRIMARY_URL}/contao"
  # validate running project
  run -0 curl -sSfI ${PRIMARY_URL}/contao/login
  assert_output --partial "HTTP/2 200"
}

@test "Contao Manager quickstart with $(ddev --version)" {
  PROJNAME=my-contao-manager-site

  run -0 mkdir -p ${PROJNAME} && cd ${PROJNAME}
  run -0 ddev config --project-type=php --docroot=public --webserver-type=apache-fpm --php-version=8.4
  # set DATABASE_URL and MAILER_DSN in .env.local
  run -0 ddev dotenv set .env.local --database-url=mysql://db:db@db:3306/db --mailer-dsn=smtp://localhost:1025
  run -0 ddev start -y

  _extra_info

  run -0 ddev exec "wget -O public/contao-manager.phar.php https://download.contao.org/contao-manager/stable/contao-manager.phar"
  DDEV_DEBUG=true run -0 ddev launch contao-manager.phar.php
  assert_output "FULLURL ${PRIMARY_URL}/contao-manager.phar.php"
  # validate running project
  run -0 curl -sSfI ${PRIMARY_URL}/contao-manager.phar.php
  assert_output --partial "HTTP/2 302"
  assert_output --partial "location: /contao-manager.phar.php/"
}
