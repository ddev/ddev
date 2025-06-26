#!/usr/bin/env zsh
#ddev-generated
# This script should be sourced in the context of your shell like so:
# source $HOME/.ddev/commands/host/shells/ddev.zsh
# Once the ddevcd() function is defined, you can type
# "ddevcd project-name" to cd into the project directory.

ddevcd() {
  cd "$(DDEV_VERBOSE=false DDEV_DEBUG=false ddev debug cd --get-approot -- "$1")"
}

_ddevcd_autocomplete() {
  compadd $(DDEV_VERBOSE=false DDEV_DEBUG=false ddev debug cd --list 2>/dev/null)
}

compdef _ddevcd_autocomplete ddevcd
