#!/usr/bin/env sh
#ddev-generated
# This script should be sourced in the context of your shell like so:
# source $HOME/.ddev/commands/host/shells/ddev.sh
# Once the ddev() function is defined, you can type
# "ddev cd project-name" to cd into the project directory.

ddev() {
  if [ "$1" = "cd" ] && [ -n "$2" ]; then
    cd "$(DDEV_VERBOSE=false command ddev cd "$2")"
  else
    command ddev "$@"
  fi
}
