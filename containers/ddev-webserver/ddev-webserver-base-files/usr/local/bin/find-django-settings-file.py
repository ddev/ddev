#!/usr/bin/env python
"""
Find out what directory might have settings file, based
on DJANGO_SETTINGS_MODULE or just finding a settings.py
"""
import os
import sys
import importlib
import inspect
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

def get_settings_filename():
    django_settings_module = os.environ.get('DJANGO_SETTINGS_MODULE')
    settings_file = ""
    if django_settings_module:
        try:
            # Import the settings module
            module = importlib.import_module(django_settings_module)

            # Get the file path of the imported module
            settings_file = inspect.getfile(module)

            return settings_file
        except ImportError:
            print(f"Could not import the specified DJANGO_SETTINGS_MODULE: {django_settings_module}")
            return None
    else:
        settings_file = find_settings_file(current_dir)

        # If settings.py is found, set the DJANGO_SETTINGS_MODULE environment variable
        if settings_file:
            sys.path.insert(0, str(settings_file.parent.parent))
        else:
            raise FileNotFoundError("Could not find the settings.py file.")

    return settings_file


# Make sure that current dir is in module path
current_dir = os.getcwd()
if str(current_dir) not in sys.path:
    # Add the current directory to sys.path
    sys.path.insert(0, str(current_dir))

# Check if DJANGO_SETTINGS_MODULE is set
f = get_settings_filename()
print(f"{f}")
