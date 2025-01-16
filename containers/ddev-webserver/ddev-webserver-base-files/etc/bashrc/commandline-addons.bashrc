# Add $HOME/bin as the first item to the $PATH.
case ":$PATH:" in
    # If the item is already in $PATH, don't add it again.
    *":$HOME/bin:"*) ;;
    # Otherwise, add it.
    *) PATH="$HOME/bin:$PATH" ;;
esac
# Add /var/www/html/bin as the next-to-last item to the $PATH.
case ":$PATH:" in
    # If the item is already in $PATH, don't add it again.
    *":/var/www/html/bin:"*) ;;
    # Otherwise, add it.
    *) PATH="$PATH:/var/www/html/bin" ;;
esac

# Add /mnt/ddev-global-cache/global-commands/web as the last item to the $PATH.
# This allows commands that weren't found elsewhere to be executed. For example, `xdebug on`
# can be used inside web container
case ":$PATH:" in
    # If the item is already in $PATH, don't add it again.
    *":/mnt/ddev-global-cache/global-commands/web:"*) ;;
    # Otherwise, add it.
    *) PATH="$PATH:/mnt/ddev-global-cache/global-commands/web" ;;
esac

# And don't forget to export the new $PATH.
export PATH

[ -s "$NVM_DIR/nvm.sh" ] && source "$NVM_DIR/nvm.sh"
[ -s "$NVM_DIR/bash_completion" ] && source "$NVM_DIR/bash_completion"
