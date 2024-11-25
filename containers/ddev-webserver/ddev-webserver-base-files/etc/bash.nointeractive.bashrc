# This file is loaded in non-interactive bash shells through $BASH_ENV

# Add ${DDEV_COMPOSER_ROOT:-/var/www/html}/vendor/bin as the first item to the $PATH.
case ":$PATH:" in
    # If the item is already in $PATH, don't add it again.
    *":${DDEV_COMPOSER_ROOT:-/var/www/html}/vendor/bin:"*) ;;
    # Otherwise, add it.
    *) PATH="${DDEV_COMPOSER_ROOT:-/var/www/html}/vendor/bin:$PATH" ;;
esac
# And don't forget to export the new $PATH.
export PATH
# Hide vendor/bin/composer from $PATH.
export EXECIGNORE="${DDEV_COMPOSER_ROOT:-/var/www/html}/vendor/bin/composer"

for f in /etc/bashrc/*.bashrc; do
    source $f;
done
unset f

for i in $(\ls $HOME/.bashrc.d/* 2>/dev/null); do
    source $i;
done
unset i
