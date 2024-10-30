#!/usr/bin/env sh
#ddev-generated
# This script should be sourced in the context of your shell like so:
# source $HOME/.ddev/commands/host/shells/ddev.sh
# Once the ddev() function is defined, you can type
# "ddev cd project-name" to cd into the project directory.

ddev() {
  if [ "$#" -eq 2 ] && [ "$1" = "cd" ]; then
    case "$2" in
      -*) command ddev "$@" ;;
      *) cd "$(DDEV_VERBOSE=false command ddev cd "$2" --get-approot)" ;;
    esac
  else
    command ddev "$@"
  fi
}
