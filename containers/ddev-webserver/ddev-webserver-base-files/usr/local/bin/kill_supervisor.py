#!/usr/bin/env python3
import sys
import os
import signal

# From https://blog.zhaw.ch/icclab/process-management-in-docker-containers/

def write_stdout(s):
   sys.stdout.write(s)
   sys.stdout.flush()

def write_stderr(s):
   sys.stderr.write(s)
   sys.stderr.flush()

def check_pid(pid):        
    """ Check For the existence of a unix pid. """
    try:
        os.kill(pid, 0)
    except OSError:
        return False
    else:
        return True

def check_by_pid_file(pidFile):
    if(not os.path.isfile(pidFile)):
        return False
    
    pid = int(open(pidFile,'r').readline())
    return check_pid(pid)
    
def main():
    while 1:
        write_stdout('READY\n')
        line = sys.stdin.readline()
        write_stdout('This line kills supervisor: ' + line)
        try:
            supervisorPidFile = '/var/run/supervisord.pid'
            isSupervisordAlive = check_by_pid_file(supervisorPidFile)
            isApacheAlive = check_by_pid_file('/var/run/apache2/apache2.pid')
            isNginxAlive = check_by_pid_file('/var/run/nginx.pid')

            if (not isApacheAlive and not isNginxAlive and isSupervisordAlive):
                supervisordPid = int(open(supervisorPidFile,'r').readline())
                os.kill(supervisordPid, signal.SIGQUIT)
        except Exception as e:
                write_stdout('Could not kill supervisor: ' + e.strerror + '\n')
        write_stdout('RESULT 2\nOK')
if __name__ == '__main__':
    main()
