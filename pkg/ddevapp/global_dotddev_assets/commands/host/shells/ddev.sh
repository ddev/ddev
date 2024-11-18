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

iterm2_ddev_status_user_var() {
  # Warn if DDEV isn't installed.
  command -v ddev > /dev/null 2>&1 || { echo "ᙌ DDEV isn't installed"; return; }

  # Get the DDEV status in the current directory.
  ddev_describe_raw=$(ddev describe --json-output 2>/dev/null | jq -r .raw 2>/dev/null) || return
  project_name=$(echo "$ddev_describe_raw" | jq -r .name 2>/dev/null)
  project_status=$(echo "$ddev_describe_raw" | jq -r .status 2>/dev/null)

  # Exit silently if not in a DDEV project.
  [ -z "$project_name" ] && return

  # Output the project name and status.
  echo "ᙌ $project_name: $project_status"
}
