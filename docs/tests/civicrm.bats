#!/usr/bin/env bats

setup() {
  PROJNAME=my-civicrm-site
  load 'common-setup'
  _common_setup
}

# executed after each test
teardown() {
  _common_teardown
}

@test "CiviCRM quickstart with $(ddev --version)" {
  run -0 mkdir -p ${PROJNAME}
  cd ${PROJNAME}

  run -0 ddev config --project-type=php --composer-root=core --upload-dirs=public/media

  # GitHub Actions always fail on "ddev composer require civicrm/cli-tools --no-scripts"
  # We can use newer curl from backports for testing to avoid this problem.
  # More info: https://github.com/ddev/ddev/pull/7897
  cat << 'EOF' > .ddev/web-build/pre.Dockerfile.backports
RUN printf "Types: deb\nURIs: http://deb.debian.org/debian\nSuites: trixie-backports\nComponents: main\nSigned-By: /usr/share/keyrings/debian-archive-keyring.pgp\n" > /etc/apt/sources.list.d/debian-backports.sources
EOF
  assert_file_exist .ddev/web-build/pre.Dockerfile.backports
  run -0 ddev config --webimage-extra-packages="curl/trixie-backports"

  run -0 ddev start

  _extra_info

  run -0 ddev exec "curl -fsSL --retry 3 --retry-all-errors https://download.civicrm.org/latest/civicrm-STABLE-standalone.tar.gz -o /tmp/civicrm-standalone.tar.gz"

  run -0 ddev exec "tar --strip-components=1 -xzf /tmp/civicrm-standalone.tar.gz"

  run -0 ddev composer require civicrm/cli-tools --no-scripts

  run -0 ddev exec cv core:install \
      --cms-base-url='$DDEV_PRIMARY_URL' \
      --db=mysql://db:db@db/db \
      -m loadGenerated=1 \
      -m extras.adminUser=admin \
      -m extras.adminPass=admin \
      -m extras.adminEmail=admin@example.com

  # ddev launch
  DDEV_DEBUG=true run -0 ddev launch
  assert_output "FULLURL ${PRIMARY_URL}"

  # validate running project
  run -0 curl -sSfI ${PRIMARY_URL}
  assert_output --partial "location: /civicrm/home"
  assert_output --partial "HTTP/2 302"
}
