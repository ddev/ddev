import os
import sys
import subprocess
from pathlib import Path

# Function to search for the settings.py file
def find_settings_file(path: Path):
    for root, dirs, files in os.walk(path):
        if "settings.py" in files:
            return Path(root) / "settings.py"
    return None

def convert_import_path(import_path):
    parts = import_path.split(".")
    gunicorn_path = ".".join(parts[:-1]) + ":" + parts[-1]
    return gunicorn_path

def launch_gunicorn(wsgi_application, bind_address):
    command = f"gunicorn  -b {bind_address} {wsgi_application}"
    print(command)
    process = subprocess.Popen(command, shell=True)
    return process


# Make sure that current dir is in module path
current_dir = os.getcwd()
if str(current_dir) not in sys.path:
    # Add the current directory to sys.path
    sys.path.insert(0, str(current_dir))

print(f"sys.path={sys.path}")
# Check if DJANGO_SETTINGS_MODULE is set
if not os.environ.get("DJANGO_SETTINGS_MODULE"):

    # Search for settings.py
    settings_file = find_settings_file(current_dir)

    # If settings.py is found, set the DJANGO_SETTINGS_MODULE environment variable
    if settings_file:
        sys.path.insert(0, str(settings_file.parent.parent))
        os.environ["DJANGO_SETTINGS_MODULE"] = f"{settings_file.parent.name}.settings"
    else:
        raise FileNotFoundError("Could not find the settings.py file.")

print(f"DJANGO_SETTINGS_MODULE={os.environ.get('DJANGO_SETTINGS_MODULE')}")
from django.conf import settings
wsgi_application = settings.WSGI_APPLICATION
if not wsgi_application:
    raise ValueError("WSGI_APPLICATION is not set in the settings module.")
wsgi_application = convert_import_path(wsgi_application)
print(f"wsgi_application is set to: {wsgi_application}")

bind_address = "0.0.0.0:8000"
process = launch_gunicorn(wsgi_application, bind_address)
print(f"Launched Gunicorn for {wsgi_application} at {bind_address}")


print("Press Ctrl+C to stop all Gunicorn processes.")

try:
    while True:
        pass
except KeyboardInterrupt:
    print("Stopping Gunicorn process...")
    process.terminate()

