#!/bin/bash

function ddev_custom_init_scripts {
  echo "Loading custom entrypoint config from ${DDEV_WEB_ENTRYPOINT}";
  if ls ${DDEV_WEB_ENTRYPOINT}/*.sh >/dev/null 2>&1; then
    for f in ${DDEV_WEB_ENTRYPOINT}/*.sh; do
      echo "sourcing $f"
      source "$f"
    done
  fi
#      touch "${DDEV_WEB_ENTRYPOINT}/.user_scripts_initialized"
}

# Set up things that gunicorn and project may need, venv
function ddev_python_setup {
  set -e
  if [ "${DDEV_WEBSERVER_TYPE}" = "nginx-gunicorn" ]; then
    python -m venv /var/www/html/.ddev/.venv
    source /var/www/html/.ddev/.venv/bin/activate
    pip install wheel
    pip install django gunicorn psycopg2-binary
    if [ -f /var/www/html/requirements.txt ]; then
      pip install -r /var/www/html/requirements.txt || true
    elif [ -f /var/www/html/pyproject.toml ]; then
      pip install /var/www/html || true
    fi
  fi
  set +e
}
