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
  run -0 mkdir -p ${WAGTAIL_SITENAME} && cd ${WAGTAIL_SITENAME}

  run -0 ddev config --project-type=generic --webserver-type=generic \
    --webimage-extra-packages=python3-pip,python3-venv \
    --web-environment-add=DJANGO_SETTINGS_MODULE=mysite.settings.dev \
    --omit-containers=db

  cat <<'DOCKERFILEEND' >.ddev/web-build/Dockerfile.python-venv
RUN for file in /etc/bash.bashrc /etc/bash.nointeractive.bashrc; do \
        echo '[ -s "$DDEV_APPROOT/env/bin/activate" ] && source "$DDEV_APPROOT/env/bin/activate"' >> "$file"; \
    done
DOCKERFILEEND
  run -0 ddev mutagen sync
  assert_file_exist .ddev/web-build/Dockerfile.python-venv

  run -0 ddev start -y

  run -0 ddev exec python -m venv env

  run -0 ddev exec pip install wagtail

  run -0 ddev exec wagtail start mysite .

  run -0 ddev exec pip install -r requirements.txt

  run -0 ddev exec "echo \"SECURE_PROXY_SSL_HEADER = ('HTTP_X_FORWARDED_PROTO', 'https')\" >> mysite/settings/dev.py"

  run -0 ddev exec python manage.py migrate --noinput

  # Create superuser non-interactively
  run -0 ddev exec "DJANGO_SUPERUSER_PASSWORD=admin python manage.py createsuperuser --username=admin --email=admin@example.com --noinput"

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
  run -0 ddev mutagen sync
  assert_file_exist .ddev/config.wagtail.yaml

  run -0 ddev restart

  # web_extra_exposed_ports only takes effect after this restart, so the
  # generic webserver has no primary URL to report until now (see
  # GetPrimaryRouterHTTP(S)Port in pkg/ddevapp/ddevapp.go).
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
}
