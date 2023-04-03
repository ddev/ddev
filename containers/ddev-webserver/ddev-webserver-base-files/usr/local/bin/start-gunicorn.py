#!/usr/bin/env python
"""
Start up gunicorn, with
1. WSGI_APP environment variable setting if it exists
2. WSGI_APP derived from django settings if it exists
3. Fail but run a placeholder tail -f /dev/null
"""
import os
import sys
import subprocess
from pathlib import Path

bind_address = "0.0.0.0:8000"

def convert_import_path(import_path):
    parts = import_path.split(".")
    gunicorn_path = ".".join(parts[:-1]) + ":" + parts[-1]
    return gunicorn_path

def launch_gunicorn(wsgi_application, bind_address):
    command = f"gunicorn  -b {bind_address} -w 4 {wsgi_application}"
    print(command)
    process = subprocess.Popen(command, shell=True)
    return process


# Make sure that current dir is in module path
current_dir = os.getcwd()
if str(current_dir) not in sys.path:
    # Add the current directory to sys.path
    sys.path.insert(0, str(current_dir))

print(f"sys.path={sys.path}")

wsgi_app = os.getenv("WSGI_APP")
django_settings_module = os.getenv("DJANGO_SETTINGS_MODULE")
ddev_project_type = os.getenv("DDEV_PROJECT_TYPE")
print(f"WSGI_APP environment variable={wsgi_app} DJANGO_SETTINGS_MODULE={django_settings_module}")


if wsgi_app:
    process = launch_gunicorn(wsgi_app, bind_address)
    print(f"Launched Gunicorn for {wsgi_app} at {bind_address}")

elif ddev_project_type == "django4":
    settings_file = subprocess.run(['python', '/usr/local/bin/find-django-settings-file.py'], stdout=subprocess.PIPE, text=True)
    print(f"settings_file='{settings_file}'")
    from django.conf import settings
    wsgi_application = settings.WSGI_APPLICATION
    print(f"wsgi_application from django='{wsgi_application}'")
    if wsgi_application:
        wsgi_app = convert_import_path(wsgi_application)
        process = launch_gunicorn(wsgi_app, bind_address)
        print(f"Launched Gunicorn for {wsgi_app} at {bind_address}")
else:
        print("wsgi_application not found in the settings module, just running tail -f /dev/null instead")

subprocess.run(['tail', "-f", "/dev/null"])

