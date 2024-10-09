export PATH="~/bin:$PATH"

[ -s "$NVM_DIR/nvm.sh" ] && source "$NVM_DIR/nvm.sh"
[ -s "$NVM_DIR/bash_completion" ] && source "$NVM_DIR/bash_completion"

([ "${DDEV_PROJECT_TYPE}" = "python" ] || [ "${DDEV_PROJECT_TYPE}" = "django4" ]) && [ -s /var/www/html/.ddev/.venv ] && source /var/www/html/.ddev/.venv/bin/activate
