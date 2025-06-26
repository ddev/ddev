#!/usr/bin/env fish
#ddev-generated
# This script should be sourced in the context of your shell like so:
# source $HOME/.ddev/commands/host/shells/ddev.fish
# Alternatively, it can be installed into one of the directories
# that fish uses to autoload functions (e.g ~/.config/fish/functions)
# Once the ddevcd() function is defined, you can type
# "ddevcd project-name" to cd into the project directory.

function ddevcd
  cd (DDEV_VERBOSE=false DDEV_DEBUG=false ddev debug cd --get-approot -- "$argv[1]")
end

function __ddevcd_autocomplete
  DDEV_VERBOSE=false DDEV_DEBUG=false ddev debug cd --list 2>/dev/null
end

complete -c ddevcd -f -a "(__ddevcd_autocomplete)"
