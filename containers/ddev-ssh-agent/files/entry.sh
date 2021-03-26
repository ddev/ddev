#!/bin/bash
# Copyright (c) Andreas Urbanski, 2018
#
# Permission is hereby granted, free of charge, to any person obtaining a
# copy of this software and associated documentation files (the "Software"),
# to deal in the Software without restriction, including without limitation
# the rights to use, copy, modify, merge, publish, distribute, sublicense,
# and/or sell copies of the Software, and to permit persons to whom the
# Software is furnished to do so, subject to the following conditions:
#
# The above copyright notice and this permission notice shall be included
# in all copies or substantial portions of the Software.
#
# THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS
# OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
# FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL
# THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
# LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
# OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
# THE SOFTWARE.


set -eo pipefail
rm -f /tmp/healthy

# Print a debug message if debug mode is on ($DEBUG is not empty)
# @param message
debug_msg ()
{
  if [ -n "$DEBUG" ]; then
    echo "$@"
  fi
}

mkdir -p $SSH_KEY_DIR && ( chmod 700 $SSH_KEY_DIR || true )
mkdir -p $SOCKET_DIR

case "$1" in
  # Start ssh-agent
  ssh-agent)

  # Create proxy-socket for ssh-agent (to give everyone acceess to the ssh-agent socket)
  echo "Creating a proxy socket..."
  rm -f ${SSH_AUTH_SOCK} ${SSH_AUTH_PROXY_SOCK}
  echo "Running socat UNIX-LISTEN:${SSH_AUTH_PROXY_SOCK},perm=0666,fork UNIX-CONNECT:${SSH_AUTH_SOCK}"
  socat UNIX-LISTEN:${SSH_AUTH_PROXY_SOCK},perm=0666,fork UNIX-CONNECT:${SSH_AUTH_SOCK} &

  echo "Launching ssh-agent..."
  exec /usr/bin/ssh-agent -a ${SSH_AUTH_SOCK} -d
  ;;

	# Manage SSH identities
  ssh-add)
  shift # remove argument from array

  # Add keys id_rsa and id_dsa from /root/.ssh using cat so it will work regardless of permissions
  # docker toolbox mounts files as 0777, which ruins the normal technique.
  set +o errexit
  keyfiles=$(file ~/.ssh/* | awk -F: '/private key/ {  print $1 }')
  set -o errexit
  if [ ! -z "$keyfiles" ] ; then
      for key in $keyfiles; do
        perm=$(stat -c %a "$key")
        if [ $perm = "777" ] ; then
            echo "Please add password for ${key}..."
            cat $key | ssh-add -k -
        else
            ssh-add $key
        fi
      done
  else
    echo "No private keys were found in the directory."
  fi

  # Return first command exit code
  exit ${PIPESTATUS[0]}
  ;;
	*)
  exec $@
  ;;
esac
