# Helper for safely adding entries to $EXECIGNORE without duplicates.
execignore_add() {
  case ":${EXECIGNORE}:" in
    *":$1:"*) ;;
    *) EXECIGNORE="${EXECIGNORE:+${EXECIGNORE}:}$1" ;;
  esac
}

# Hide vendor/bin/composer from $PATH so `composer` resolves to the system install,
# not the project-local shim inside vendor/bin.
execignore_add "${DDEV_COMPOSER_ROOT:-/var/www/html}/vendor/bin/composer"

unset -f execignore_add

export EXECIGNORE
