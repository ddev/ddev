#!/bin/bash

# This script tries to diagnose mutagen issues. Please run it in the project
# directory where you're having trouble and provide its
# output in a gist at gist.github.com with any issue you open about mutagen.

if command -v mutagen >/dev/null ; then
  echo "mutagen additionally installed in PATH at $(command -v mutagen), version $(mutagen version)"
fi
if killall -0 mutagen 2>/dev/null; then
  echo "mutagen is running on this system:"
  ps -ef | grep mutagen
fi

ddev list
ddev describe
~/.ddev/bin/mutagen sync list -l
~/.ddev/bin/mutagen version
