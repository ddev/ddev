#!/bin/bash

function ddev_custom_init_scripts {
  echo "Loading custom entrypoint config from ${ENTRYPOINT}";
  for f in ${ENTRYPOINT}/*.sh; do
    echo "sourcing web-entrypoint.d/$f"
    . "$f"
  done
#      touch "${ENTRYPOINT}/.user_scripts_initialized"
}

# Set up things that gunicorn and project may need, venv
function ddev_python_setup {
  python -m venv /var/www/html/.ddev/.venv
  source /var/www/html/.ddev/.venv/bin/activate
  pip install psycopg2 gunicorn
  if [ -f requirements.txt ]; then
    pip install -r requirements.txt
  fi
  if [ -f pyproject.toml ]; then
    pip install .
  fi
}
