# Extends PATH to include the composer bin directory.

# set COMPOSER_BIN_DIR to $WEBSERVER_DOCROOT/vendor/bin if unset
if [[ ! -v $COMPOSER_BIN_DIR ]]; then
    export COMPOSER_BIN_DIR="$WEBSERVER_DOCROOT/vendor/bin"
fi

# add composer bin dir to PATH
export PATH="$PATH:$COMPOSER_BIN_DIR"
