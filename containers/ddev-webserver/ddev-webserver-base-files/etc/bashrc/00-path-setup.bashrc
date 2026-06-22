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
path_remove() {
  local p result= IFS=:
  for p in $PATH; do
    [ "$p" = "$1" ] || result="${result:+$result:}$p"
  done
  PATH="$result"
}

# Add vendor/bin, then user-owned dirs in front of it (prepend order is last-wins)
path_prepend "${DDEV_COMPOSER_ROOT:-/var/www/html}/vendor/bin"
path_prepend "$HOME/bin"
path_prepend "$HOME/.local/bin"

# N_PREFIX for Node.js
path_append "/usr/local/n/bin"

# Add /var/www/html/bin as the next-to-last item to the $PATH.
path_append "/var/www/html/bin"

# Add /mnt/ddev-global-cache/global-commands/web as a fallback for commands like `xdebug on`,
# but drop it when this shell is running a script from that dir, so a wrapper calling a
# same-named binary (e.g. yarn -> `yarn "$@"`) doesn't recurse into itself.
case "$0" in
  /mnt/ddev-global-cache/global-commands/web/*)
    path_remove "/mnt/ddev-global-cache/global-commands/web"
    ;;
  *)
    path_append "/mnt/ddev-global-cache/global-commands/web"
    ;;
esac

unset -f path_prepend path_append path_remove

export PATH
