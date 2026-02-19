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
  _skip_if_embargoed "wagtail-gunicorn"

  WAGTAIL_SITENAME=${PROJNAME}
  run mkdir ${WAGTAIL_SITENAME} && cd ${WAGTAIL_SITENAME}
  assert_success

  run ddev config --project-type=generic --webserver-type=generic \
    --webimage-extra-packages=python3-pip,python3-venv \
    --web-environment-add=DJANGO_SETTINGS_MODULE=mysite.settings.dev \
    --omit-containers=db
  assert_success

  cat <<'DOCKERFILEEND' >.ddev/web-build/Dockerfile.python-venv
RUN for file in /etc/bash.bashrc /etc/bash.nointeractive.bashrc; do \
        echo '[ -s "$DDEV_APPROOT/env/bin/activate" ] && source "$DDEV_APPROOT/env/bin/activate"' >> "$file"; \
    done
DOCKERFILEEND
  run ddev mutagen sync
  assert_success
  assert_file_exist .ddev/web-build/Dockerfile.python-venv

  run ddev start -y
  assert_success

  run ddev exec python -m venv env
  assert_success

  run ddev exec pip install wagtail gunicorn
  assert_success

  run ddev exec wagtail start mysite .
  assert_success

  run ddev exec pip install -r requirements.txt
  assert_success

  run ddev exec "echo \"SECURE_PROXY_SSL_HEADER = ('HTTP_X_FORWARDED_PROTO', 'https')\" >> mysite/settings/dev.py"
  assert_success

  run ddev exec python manage.py migrate --noinput
  assert_success

  # Create superuser non-interactively
  run ddev exec "DJANGO_SUPERUSER_PASSWORD=admin python manage.py createsuperuser --username=admin --email=admin@example.com --noinput"
  assert_success

  cat <<'EOF' > .ddev/config.wagtail.yaml
web_extra_daemons:
    - name: "wagtail"
      command: "gunicorn mysite.wsgi:application -b 0.0.0.0:8000"
      directory: /var/www/html
web_extra_exposed_ports:
    - name: "wagtail"
      container_port: 8000
      http_port: 80
      https_port: 443
EOF
  run ddev mutagen sync
  assert_success
  assert_file_exist .ddev/config.wagtail.yaml

  run ddev restart
  assert_success

  # ddev launch /admin
  DDEV_DEBUG=true run ddev launch /admin
  assert_output --partial "FULLURL https://${PROJNAME}.ddev.site/admin"
  assert_success

  # Wait for gunicorn (web_extra_daemon) to be ready - DDEV doesn't poll extra daemons for readiness
  for i in $(seq 1 15); do
    if ddev exec curl -sf --max-time 3 http://localhost:8000 >/dev/null 2>&1; then
      break
    fi
    sleep 2
  done

  # validate running project - check if Wagtail is responding
  run curl -sfI --max-time 30 https://${PROJNAME}.ddev.site
  assert_output --partial "server: gunicorn"
  assert_success

  # Verify main site is running
  run curl -sf --max-time 30 https://${PROJNAME}.ddev.site
  assert_output --partial "Welcome to your new Wagtail site"
  assert_success

  # Check if we can access the admin page (should redirect to login)
  run curl -sfL --max-time 30 https://${PROJNAME}.ddev.site/admin
  assert_output --partial "Sign in - Wagtail"
  assert_success
}
