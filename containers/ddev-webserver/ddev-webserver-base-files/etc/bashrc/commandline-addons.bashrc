case ":$PATH:" in
    *":$HOME/bin:"*) ;;
    *) PATH="$HOME/bin:$PATH" ;;
esac

case ":$PATH:" in
    *":/var/www/html/bin:"*) ;;
    *) PATH="$PATH:/var/www/html/bin" ;;
esac

export PATH

[ -s "$NVM_DIR/nvm.sh" ] && source "$NVM_DIR/nvm.sh"
[ -s "$NVM_DIR/bash_completion" ] && source "$NVM_DIR/bash_completion"

([ "${DDEV_PROJECT_TYPE}" = "python" ] || [ "${DDEV_PROJECT_TYPE}" = "django4" ]) && [ -s /var/www/html/.ddev/.venv ] && source /var/www/html/.ddev/.venv/bin/activate
