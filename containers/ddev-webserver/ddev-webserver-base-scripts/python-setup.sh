#!/bin/bash

# Do required setup for nginx-python webserver_type

set -o errexit nounset pipefail

python -m venv /var/www/html/.ddev/.venv
source /var/www/html/.ddev/.venv/bin/activate
pip install wheel
pip install django gunicorn psycopg2-binary
if [ -f /var/www/html/requirements.txt ]; then
  pip install -r /var/www/html/requirements.txt || true
elif [ -f /var/www/html/pyproject.toml ]; then
  pip install /var/www/html || true
fi
