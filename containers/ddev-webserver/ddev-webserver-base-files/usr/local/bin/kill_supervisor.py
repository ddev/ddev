#!/usr/bin/env python3
import sys
import os
import signal
import pdb

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

def check_supervisord():
    if(not os.path.isfile('/var/run/supervisord.pid')):
        return False
    
    supervisordPid = int(open('/var/run/supervisord.pid','r').readline())
    if(check_pid(supervisordPid)):
        write_stdout("supervisord alive\n")
    else:
        write_stdout("supervisord dead\n")
    return check_pid(supervisordPid)

def check_apache():
    if(not os.path.isfile('/var/run/apache2/apache2.pid')):
        return False
    
    apachePid = int(open('/var/run/apache2/apache2.pid','r').readline())
    if(check_pid(apachePid)):
        write_stdout("apache alive\n")
    else:
        write_stdout("apache dead\n")
    return check_pid(apachePid)

def check_nginx():
    if(not os.path.isfile('/var/run/nginx.pid')):
        return False
    
    nginxPid = int(open('/var/run/nginx.pid','r').readline())
    if(check_pid(nginxPid)):
        write_stdout("nginx alive\n")
    else:
        write_stdout("nginx dead\n")
    return check_pid(nginxPid)
    
        
def main():
    while 1:
        write_stdout('READY\n')
        line = sys.stdin.readline()
        write_stdout('This line kills supervisor: ' + line)
        try:
            isSupervisordAlive = check_supervisord()
            isApacheAlive = check_apache()
            isNginxAlive = check_nginx()

            if (not isApacheAlive and not isNginxAlive and isSupervisordAlive):
                pdb.set_trace()
                supervisordPid = int(open('/var/run/supervisord.pid','r').readline())
                os.kill(supervisordPid, signal.SIGQUIT)
        except Exception as e:
                write_stdout('Could not kill supervisor: ' + e.strerror + '\n')
        write_stdout('RESULT 2\nOK')
if __name__ == '__main__':
    main()
