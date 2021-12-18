#!/bin/bash

# Used only on Windows to chmod files uploaded using docker api where the
# executable bit is lost.
set -eu -o pipefail
if [ $# = 0 ]; then
  echo "Requires one of more file or directory args" && exit 23
fi
find $* -type f | xargs file -i  | awk -F: '/text.x-shellscript/ { print $1 }' >/tmp/scripts.txt
if [ -s /tmp/scripts.txt ]; then
  chmod +x $(cat /tmp/scripts.txt)
fi
