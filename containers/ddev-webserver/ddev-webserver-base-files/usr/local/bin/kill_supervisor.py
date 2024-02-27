#!/usr/bin/env python3
import sys
import os
import signal
import subprocess


# From https://blog.zhaw.ch/icclab/process-management-in-docker-containers/
# http://supervisord.org/events.html#example-event-listener-implementation

def write_stdout(s):
    sys.stdout.write(s)
    sys.stdout.flush()
def write_stderr(s):
    sys.stderr.write(s)
    sys.stderr.flush()
# Function to check if a process is running using ps, grep, and awk
def is_process_running(process_identifier):
    try:
        # Use ps and grep to search for the process, excluding the grep command itself
        output = subprocess.check_output(
            f"ps aux | grep '[{process_identifier[0]}]{process_identifier[1:]}' | grep -v grep", shell=True)
        # If the output is not empty, then the process is considered running
        return bool(output.decode('utf-8').strip())
    except subprocess.CalledProcessError:
        # If the grep command does not find anything, it will raise a CalledProcessError
        return False
# Function to parse the config.yaml file for the webserver_type
def read_webserver_type_from_yaml(file_path):
    webserver_type = None
    try:
        with open(file_path, 'r') as file:
            for line in file:
                if line.strip().startswith('webserver_type:'):
                    webserver_type = line.split(':', 1)[1].strip()
                    break
    except Exception as e:
        write_stderr(f"Error reading webserver_type from YAML: {str(e)}\n")
    return webserver_type
def main():
    # Read the webserver_type from the config.yaml
    webserver_type = read_webserver_type_from_yaml('/var/www/html/.ddev/config.yaml')
    critical_processes = {
        'nginx-fpm': ['nginx: ', 'php-fpm: '],
        'apache-fpm': ['apache2 ', 'php-fpm: '],
        'nginx-gunicorn': ['nginx: ', 'start-gunicorn.py'],
    }.get(webserver_type, [])

    while 1:
        write_stdout('READY\n')
        line = sys.stdin.readline()
        try:
            # Check if any critical process is not running
            process_failure = False
            for process in critical_processes:
                if not is_process_running(process):
                    process_failure = True
                    write_stderr(f'Critical process {process} for {webserver_type} is not running. Action required.\n')
                    break

            if process_failure:
                # Proceed with original behavior if a critical process is not running
                write_stdout('This line kills supervisor: ' + line)
                pidfile = open('/var/run/supervisord.pid', 'r')
                pid = int(pidfile.readline())
                os.kill(pid, signal.SIGQUIT)
                write_stderr('Supervisor killed because a critical process is not running.\n')

        except Exception as e:
            write_stdout('Could not kill supervisor: ' + e.strerror + '\n')

        write_stdout('RESULT 2\nOK')
if __name__ == '__main__':
    main()
    import sys
