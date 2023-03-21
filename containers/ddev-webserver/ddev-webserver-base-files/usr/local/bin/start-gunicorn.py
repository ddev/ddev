import os
import importlib.util
import subprocess


def find_settings_files(start_dir):
    settings_files = []
    for root, dirs, files in os.walk(start_dir):
        for file in files:
            if file == "settings.py":
                settings_files.append(os.path.join(root, file))
    return settings_files


def import_module_from_path(file_path):
    spec = importlib.util.spec_from_file_location("settings_module", file_path)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


def launch_gunicorn(wsgi_application, bind_address):
    command = f"gunicorn  -b {bind_address} '{wsgi_application}'"
    print(command)
    process = subprocess.Popen(command, shell=True)
    return process

def convert_import_path(import_path):
    parts = import_path.split(".")
    gunicorn_path = ".".join(parts[:-1]) + ":" + parts[-1]
    return gunicorn_path

def main():
    start_dir = "../.."
    bind_address_base = "0.0.0.0"
    base_port = 8000

    settings_files = find_settings_files(start_dir)
    processes = []

    for index, settings_file in enumerate(settings_files):

        settings_module = import_module_from_path(settings_file)
        if hasattr(settings_module, 'WSGI_APPLICATION'):
            if index > 0:
                print("More than one app found. Launching only the first detected, skipping  {settings_module.WSGI_APPLICATION}.")
                continue

            print(f"{index}. WSGI_APPLICATION for {settings_file}: {settings_module.WSGI_APPLICATION}")

            gunicorn_wsgi_application = convert_import_path(settings_module.WSGI_APPLICATION)
            bind_address = f"{bind_address_base}:{base_port + index}"
            process = launch_gunicorn(gunicorn_wsgi_application, bind_address)
            processes.append(process)
            print(f"Launched Gunicorn for {settings_file} at {bind_address}")

        else:
            print(f"No WSGI_APPLICATION found in {settings_file}")

    print("Press Ctrl+C to stop all Gunicorn processes.")

    try:
        while True:
            pass
    except KeyboardInterrupt:
        print("Stopping Gunicorn processes...")
        for process in processes:
            process.terminate()


if __name__ == "__main__":
    main()
