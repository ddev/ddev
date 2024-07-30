PATH="${PATH}:/var/www/html/vendor/bin:/var/www/html/bin:${DDEV_COMPOSER_ROOT}/vendor/bin"
[ "${N_PREFIX:-}" != "" ] && PATH="$N_PREFIX/bin:$PATH"
export PATH="~/bin:${PATH}"

[ -s "$NVM_DIR/nvm.sh" ] && source "$NVM_DIR/nvm.sh"
[ -s "$NVM_DIR/bash_completion" ] && source "$NVM_DIR/bash_completion"

([ "${DDEV_PROJECT_TYPE}" = "python" ] || [ "${DDEV_PROJECT_TYPE}" = "django4" ]) && [ -s /var/www/html/.ddev/.venv ] && source /var/www/html/.ddev/.venv/bin/activate
