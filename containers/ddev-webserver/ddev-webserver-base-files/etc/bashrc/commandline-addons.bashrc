# Add $HOME/bin as the first item to the $PATH.
case ":$PATH:" in
    # If the item is already in $PATH, don't add it again.
    *":$HOME/bin:"*) ;;
    # Otherwise, add it.
    *) PATH="$HOME/bin:$PATH" ;;
esac
# Add /var/www/html/bin as the last item to the $PATH.
case ":$PATH:" in
    # If the item is already in $PATH, don't add it again.
    *":/var/www/html/bin:"*) ;;
    # Otherwise, add it.
    *) PATH="$PATH:/var/www/html/bin" ;;
esac
# And don't forget to export the new $PATH.
export PATH

[ -s "$NVM_DIR/nvm.sh" ] && source "$NVM_DIR/nvm.sh"
[ -s "$NVM_DIR/bash_completion" ] && source "$NVM_DIR/bash_completion"

([ "${DDEV_PROJECT_TYPE}" = "python" ] || [ "${DDEV_PROJECT_TYPE}" = "django4" ]) && [ -s /var/www/html/.ddev/.venv ] && source /var/www/html/.ddev/.venv/bin/activate
