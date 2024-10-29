#ddev-generated
# This script should be sourced in the context of your shell like so:
# source $HOME/.ddev/commands/host/shells/ddev.fish
# Alternatively, it can be installed into one of the directories
# that fish uses to autoload functions (e.g ~/.config/fish/functions)
# Once the ddev() function is defined, you can type
# "ddev cd project-name" to cd into the project directory.

function ddev
  if test (count $argv) -eq 2 -a "$argv[1]" = "cd"
    switch "$argv[2]"
      case '-*'
        command ddev $argv
      case '*'
        cd (DDEV_VERBOSE=false command ddev cd $argv[2] --get-approot)
    end
  else
    command ddev $argv
  end
end
