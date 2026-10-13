#!/usr/bin/env bats

setup() {
  PROJNAME=my-backdrop-site
  load 'common-setup'
  _common_setup
}

# executed after each test
teardown() {
  _common_teardown
}

@test "backdrop new-project quickstart with $(ddev --version)" {
  run -0 mkdir -p ${PROJNAME} && cd ${PROJNAME}

  run -0 ddev config --project-type=backdrop

  # Add the official Bee CLI add-on
  run -0 ddev add-on get backdrop-ops/ddev-backdrop-bee

  run -0 ddev start -y

  # Download Backdrop core
  local token
  token=$(_github_token)
  run -0 ddev bee download-core ${token:+--github-token=${token}}
  # bee exits 0 even when the GitHub API rate limit stops the download
  run -0 ddev exec test -f index.php

  # Create admin user
  run -0 ddev bee si --username=admin --password=Password123 --db-name=db --db-user=db --db-pass=db --db-host=db --auto

  _extra_info

  DDEV_DEBUG=true run -0 ddev launch
  assert_output "FULLURL ${PRIMARY_URL}"
  # Check direct http/https URLs
  run -0 curl -sSfI ${HOST_HTTP_URL}
  assert_output --partial "HTTP/1.1 200"
  run -0 curl -sSfI ${HOST_HTTPS_URL}
  assert_output --partial "HTTP/1.1 200"

  # validate running project
  run -0 curl -sSfI ${PRIMARY_HTTPS_URL}
  assert_output --partial "HTTP/2 200"
  run -0 curl -sSfI ${PRIMARY_HTTP_URL}
  assert_output --partial "HTTP/1.1 200"
  _assert_page_contains ${PRIMARY_HTTPS_URL} "Welcome to My Backdrop Site!"
}

@test "backdrop existing project with $(ddev --version)" {
  run -0 mkdir -p ${PROJNAME} && cd ${PROJNAME}

  run -0 git clone https://github.com/ddev/test-backdrop.git .

  run -0 ddev config --project-type=backdrop

  # Add the official Bee CLI add-on
  run -0 ddev add-on get backdrop-ops/ddev-backdrop-bee

  DDEV_DEBUG=true run -0 ddev start -y >&3
#  echo "# DEBUG: ddev start completed at $(date), exit code: $status" >&3
#  echo "# DEBUG: ddev start output: " >&3
#  echo "# $output" >&3

  run -0 _curl_github -fLO https://github.com/ddev/test-backdrop/releases/download/1.32.1/db.sql.gz

  run -0 ddev import-db --file=db.sql.gz

  run -0 _curl_github -fLO https://github.com/ddev/test-backdrop/releases/download/1.32.1/files.tgz

  run -0 ddev import-files --source=files.tgz

  # Clear cache after importing db and files to ensure Backdrop is ready
  run -0 ddev bee cc all

  _extra_info

  # Check direct http/https URLs
  run -0 curl -sSfI ${HOST_HTTP_URL}
  assert_output --partial "HTTP/1.1 200"
  run -0 curl -sSfI ${HOST_HTTPS_URL}
  assert_output --partial "HTTP/1.1 200"

  DDEV_DEBUG=true run -0 ddev launch
  assert_output "FULLURL ${PRIMARY_URL}"

  # validate running project
  run -0 curl -sSfI ${PRIMARY_HTTPS_URL}
  assert_output --partial "HTTP/2 200"
  run -0 curl -sSfI ${PRIMARY_HTTP_URL}
  assert_output --partial "HTTP/1.1 200"
  _assert_page_contains ${PRIMARY_HTTPS_URL} "Welcome to My Backdrop Site!"
}
