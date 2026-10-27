#!/usr/bin/env bats

setup() {
  PROJNAME=my-wagtail-site
  load 'common-setup'
  _common_setup
}

# executed after each test
teardown() {
  _common_teardown
}

@test "Wagtail quickstart with $(ddev --version)" {
  WAGTAIL_SITENAME=${PROJNAME}
  run -0 mkdir -p ${WAGTAIL_SITENAME}
  cd ${WAGTAIL_SITENAME}

  run -0 ddev config --project-type=generic --webserver-type=generic \
    --webimage-extra-packages=python3-venv \
    --web-environment-add=DJANGO_SETTINGS_MODULE=mysite.settings.dev \
    --omit-containers=db

  echo wagtail > requirements.txt
  mkdir -p mysite/settings
  echo 'SECURE_PROXY_SSL_HEADER = ("HTTP_X_FORWARDED_PROTO", "https")' > mysite/settings/local.py
  cat <<'EOF' > .ddev/docker-compose.wagtail.yaml
services:
  web:
    build:
      additional_contexts:
        project: ${DDEV_APPROOT}
EOF
  cat <<'EOF' > .ddev/web-build/Dockerfile.wagtail
COPY --from=project --chmod=644 requirements.txt /tmp/requirements.txt
RUN install -d -o "$username" /opt/wagtail \
    && runuser -u "$username" -- python3 -m venv /opt/wagtail \
    && runuser -u "$username" -- /opt/wagtail/bin/pip install --no-cache-dir -r /tmp/requirements.txt \
    && echo 'export PATH="/opt/wagtail/bin:$PATH"' > /etc/bashrc/wagtail.bashrc
EOF
  cat <<'EOF' > .ddev/config.wagtail.yaml
web_extra_daemons:
    - name: "wagtail"
      command: "python manage.py runserver 0.0.0.0:8000"
      directory: /var/www/html
web_extra_exposed_ports:
    - name: "wagtail"
      container_port: 8000
      http_port: 80
      https_port: 443
EOF
  assert_file_exist requirements.txt
  assert_file_exist mysite/settings/local.py
  assert_file_exist .ddev/docker-compose.wagtail.yaml
  assert_file_exist .ddev/web-build/Dockerfile.wagtail
  assert_file_exist .ddev/config.wagtail.yaml

  run -0 ddev start -y

  run -0 ddev exec "rm -f requirements.txt && wagtail start mysite ."

  run -0 ddev exec python manage.py migrate --noinput

  # Create superuser non-interactively
  run -0 ddev exec "DJANGO_SUPERUSER_PASSWORD=admin python manage.py createsuperuser --username=admin --email=admin@example.com --noinput"

  run -0 ddev restart

  run -0 ddev exec command -v python
  assert_output "/opt/wagtail/bin/python"

  # The virtual environment belongs to the user, so temporary installs work
  run -0 ddev exec pip install --no-cache-dir django-debug-toolbar

  _extra_info

  # ddev launch /admin
  DDEV_DEBUG=true run -0 ddev launch /admin
  assert_output --partial "FULLURL ${PRIMARY_URL}/admin"

  # validate running project - check if Wagtail is responding
  run -0 curl -sfI ${PRIMARY_URL}
  assert_line --regexp 'server:.*WSGIServer.*CPython.*'

  # Verify main site is running
  run -0 curl -sf ${PRIMARY_URL}
  assert_output --partial "Welcome to your new Wagtail site"

  # Check if we can access the admin page (should redirect to login)
  run -0 curl -sfL ${PRIMARY_URL}/admin
  assert_output --partial "Sign in - Wagtail"

  # Logging in over HTTPS passes Django's CSRF origin check only with SECURE_PROXY_SSL_HEADER
  cookies=${tmpdir}/cookies.txt
  run -0 curl -sf -c ${cookies} ${PRIMARY_URL}/admin/login/
  csrftoken=$(awk '$6 == "csrftoken" {print $7}' ${cookies})
  run -0 curl -sfL -b ${cookies} -c ${cookies} -H "Origin: ${PRIMARY_URL}" -H "Referer: ${PRIMARY_URL}/admin/login/" \
    -d "csrfmiddlewaretoken=${csrftoken}&username=admin&password=admin" ${PRIMARY_URL}/admin/login/
  assert_output --partial "Dashboard - Wagtail"
}
