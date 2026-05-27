# Helper functions for safely adding directories to $PATH without duplicates.
path_prepend() {
  case ":$PATH:" in
    *":$1:"*) ;;
    *) PATH="$1:$PATH" ;;
  esac
}
path_append() {
  case ":$PATH:" in
    *":$1:"*) ;;
    *) PATH="$PATH:$1" ;;
  esac
}

# Add vendor/bin, then user-owned dirs in front of it (prepend order is last-wins).
path_prepend "${DDEV_COMPOSER_ROOT:-/var/www/html}/vendor/bin"
path_prepend "$HOME/bin"
path_prepend "$HOME/.local/bin"

# Add /var/www/html/bin as the next-to-last item to the $PATH.
path_append "/var/www/html/bin"

# Add /mnt/ddev-global-cache/global-commands/web as the last item to the $PATH.
# This allows commands that weren't found elsewhere to be executed. For example, `xdebug on`
# can be used inside web container
path_append "/mnt/ddev-global-cache/global-commands/web"

unset -f path_prepend path_append

export PATH
