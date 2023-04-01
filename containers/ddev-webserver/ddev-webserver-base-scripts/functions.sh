#!/bin/bash

function ddev_custom_init_scripts {
  echo "Loading custom entrypoint config from ${ENTRYPOINT}";
  if ls ${ENTRYPOINT}/*.sh >/dev/null 2>&1; then
    for f in ${ENTRYPOINT}/*.sh; do
      echo "sourcing web-entrypoint.d/$f"
      . "$f"
    done
  fi
#      touch "${ENTRYPOINT}/.user_scripts_initialized"
}

# Set up things that gunicorn and project may need, venv
function ddev_python_setup {
  if [ "${DDEV_WEBSERVER_TYPE}" = "nginx-gunicorn" ]; then
    python -m venv /var/www/html/.ddev/.venv
    source /var/www/html/.ddev/.venv/bin/activate
    pip install wheel gunicorn psycopg2
    if [ -f /var/www/html/requirements.txt ]; then
      pip install -r /var/www/html/requirements.txt
    fi
    if [ -f /var/www/html/pyproject.toml ]; then
      pip install /var/www/html
    fi
  fi
}
