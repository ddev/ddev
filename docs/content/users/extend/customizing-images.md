# Customizing Docker Images

It’s common to have a requirement for the `web` or `db` images which isn’t bundled with them by default. There are two ways to extend these Docker images:

1. `webimage_extra_packages` and `dbimage_extra_packages` in `.ddev/config.yaml`.
2. An add-on Dockerfile in your project’s `.ddev/web-build` or `.ddev/db-build`.

## Adding Extra Debian Packages with `webimage_extra_packages` and `dbimage_extra_packages`

You can add extra Debian packages with lines like this in `.ddev/config.yaml`:

```yaml
webimage_extra_packages: ["php${DDEV_PHP_VERSION}-yaml", "php${DDEV_PHP_VERSION}-tidy"]
dbimage_extra_packages: [telnet, netcat, sudo]
```

Then the additional packages will be built into the containers during [`ddev start`](../usage/commands.md#start).

## Adding PHP Extensions

### PHP Extensions supported by `deb.sury.org`

If a PHP extension is supported by the upstream package management from `deb.sury.org`, you'll be able to add it with minimal effort. Test to see if it's available using `ddev exec '(sudo apt-get update || true) && sudo apt-get install php${DDEV_PHP_VERSION}-<extension>'`, for example, `ddev exec '(sudo apt-get update || true) && sudo apt-get install php${DDEV_PHP_VERSION}-imap'`. If that works, then the extension is supported, and you can add `webimage_extra_packages: ["php${DDEV_PHP_VERSION}-<extension>"]` to your `.ddev/config.yaml` file.

### PECL PHP Extensions not supported by `deb.sury.org`

!!!tip "Few people need pecl extensions"
    Most people don't need to install PHP extensions that aren't supported by `deb.sury.org`, so you only need to go down this path if you have very particular needs.

If a PHP extension is not supported by the upstream package management from `deb.sury.org`, you'll install it via pecl using a `.ddev/web-build/Dockerfile`. You can search for the extension on [pecl.php.net](https://pecl.php.net/) to find the package name. (This technique can also be used to get newer versions of PHP extensions than are available in the `deb.sury.org` distribution.)

For example, a `.ddev/web-build/Dockerfile.mcrypt` might look like this:

```dockerfile
ENV extension=mcrypt
SHELL ["/bin/bash", "-c"]
# Install the needed development packages
RUN (apt-get update || true) && DEBIAN_FRONTEND=noninteractive apt-get install -y -o Dpkg::Options::="--force-confnew" --no-install-recommends --no-install-suggests build-essential php-pear php${DDEV_PHP_VERSION}-dev
# mcrypt happens to require libmcrypt-dev
RUN apt-get install -y libmcrypt-dev
RUN pecl install ${extension}
RUN echo "extension=${extension}.so" > /etc/php/${DDEV_PHP_VERSION}/mods-available/${extension}.ini && chmod 666 /etc/php/${DDEV_PHP_VERSION}/mods-available/${extension}.ini
RUN phpenmod ${extension}
```

A `.ddev/web-build/Dockerfile.xlswriter` to add `xlswriter` might be:

```dockerfile
ENV extension=xlswriter
SHELL ["/bin/bash", "-c"]
# Install the needed development packages
RUN (apt-get update || true) && DEBIAN_FRONTEND=noninteractive apt-get install -y -o Dpkg::Options::="--force-confnew" --no-install-recommends --no-install-suggests build-essential php-pear php${DDEV_PHP_VERSION}-dev
# xlswriter requires libz-dev
RUN sudo apt-get install -y libz-dev
RUN echo | pecl install ${extension}
RUN echo "extension=${extension}.so" > /etc/php/${DDEV_PHP_VERSION}/mods-available/${extension}.ini && chmod 666 /etc/php/${DDEV_PHP_VERSION}/mods-available/${extension}.ini
RUN phpenmod ${extension}

```

A `.ddev/web-build/Dockerfile.xdebug` (overriding the `deb.sury.org` version) might look like this:

```dockerfile
# This example installs xdebug from pecl instead of the standard deb.sury.org package
ENV extension=xdebug
SHELL ["/bin/bash", "-c"]
RUN phpdismod xdebug
# Install the needed development packages
RUN (apt-get update || true) && DEBIAN_FRONTEND=noninteractive apt-get install -y -o Dpkg::Options::="--force-confnew" --no-install-recommends --no-install-suggests build-essential php-pear php${DDEV_PHP_VERSION}-dev
# Remove the standard xdebug provided by deb.sury.org
RUN apt-get remove php${DDEV_PHP_VERSION}-xdebug || true
RUN pecl install ${extension}
# Use the standard xdebug.ini from source
ADD https://raw.githubusercontent.com/ddev/ddev/main/containers/ddev-php-base/ddev-php-files/etc/php/8.2/mods-available/xdebug.ini /etc/php/${DDEV_PHP_VERSION}/mods-available
RUN chmod 666 /etc/php/${DDEV_PHP_VERSION}/mods-available/xdebug.ini
# ddev xdebug handles enabling module so we don't enable here
#RUN phpenmod ${extension}
```

## Adding Locales

The web image ships by default with a small number of locales, which work for most usages, including
`en_CA`, `en_US`, `en_GB`, `es_ES`, `es_MX`, `pt_BR`, `pt_PT`, `de_DE`, `de_AT`, `fr_CA`, `fr_FR`, `ja_JP`, and `ru_RU`.

If you need other locales, you can install all of them by adding `locales-all` to your `webimage_extra_packages`. For example, in `.ddev/config.yaml`:

```yaml
webimage_extra_packages: [locales-all]
```

## Adding Extra Dockerfiles for `webimage` and `dbimage`

For more complex requirements, you can add:

* `.ddev/web-build/Dockerfile`
* `.ddev/web-build/Dockerfile.*`
* `.ddev/db-build/Dockerfile`
* `.ddev/db-build/Dockerfile.*`

These files’ content will be inserted into the constructed Dockerfile for each image. They are inserted *after* most of the rest of the things that are done to build the image, and are done in alphabetical order, so `Dockerfile` is inserted first, followed by `Dockerfile.*` in alphabetical order.

For certain use cases, you might need to add directives very early on the Dockerfile like proxy settings or SSL termination. You can use `pre.` variants for this that are inserted *before* everything else:

* `.ddev/web-build/pre.Dockerfile.*`
* `.ddev/web-build/pre.Dockerfile`
* `.ddev/db-build/pre.Dockerfile.*`
* `.ddev/db-build/pre.Dockerfile`

Examine the resultant generated Dockerfile (which you will never edit directly), at `.ddev/.webimageBuild/Dockerfile`. You can force a rebuild with [`ddev debug rebuild`](../usage/commands.md#debug-rebuild).

Examples of possible Dockerfiles are `.ddev/web-build/Dockerfile.example` and `.ddev/db-build/Dockerfile.example`, created in your project when you run [`ddev config`](../usage/commands.md#config).

You can use the `.ddev/*-build` directory as the Docker “context” directory as well. So for example, if a file named `file.txt` exists in `.ddev/web-build`, you can use `ADD file.txt /` in the Dockerfile.

An example web image `.ddev/web-build/Dockerfile` might be:

```dockerfile
RUN npm install -g gatsby-cli
```

Another example would be installing `phpcs` globally (see [Stack Overflow answer](https://stackoverflow.com/questions/61870801/add-global-phpcs-and-drupal-coder-to-ddev-in-custom-dockerfile/61870802#61870802)):

```dockerfile
ENV COMPOSER_HOME=/usr/local/composer

# We try to avoid relying on Composer to download global, so in `phpcs` case we can use the PHAR.
RUN curl -L https://squizlabs.github.io/PHP_CodeSniffer/phpcs.phar -o /usr/local/bin/phpcs && chmod +x /usr/local/bin/phpcs
RUN curl -L https://squizlabs.github.io/PHP_CodeSniffer/phpcbf.phar -o /usr/local/bin/phpcbf && chmod +x /usr/local/bin/phpcbf

# If however we need to download a package, we use `cgr` for that.
RUN composer global require consolidation/cgr
RUN $COMPOSER_HOME/vendor/bin/cgr drupal/coder:^8.3.1
RUN $COMPOSER_HOME/vendor/bin/cgr dealerdirect/phpcodesniffer-composer-installer

# Register Drupal’s code sniffer rules.
RUN phpcs --config-set installed_paths $COMPOSER_HOME/global/drupal/coder/vendor/drupal/coder/coder_sniffer --verbose
# Make Codesniffer config file writable for ordinary users in container.
RUN chmod 666 /usr/local/bin/CodeSniffer.conf
# Make `COMPOSER_HOME` writable if regular users need to use it.
RUN chmod -R ugo+rw $COMPOSER_HOME
# Now turn it off, because ordinary users will want to be using the default.
ENV COMPOSER_HOME=""
```

**Remember that the Dockerfile is building a Docker image that will be used later with DDEV.** At the time the Dockerfile is executing, your code is not mounted and the container is not running, the image is being built. So for example, an `npm install` in `/var/www/html` will not do anything to your project because the code is not there at image building time.

### Build Time Environment Variables

The following environment variables are available for the web Dockerfile to use at build time:

* `$BASE_IMAGE`: the base image, like `ddev/ddev-webserver:v1.24.0`
* `$username`: the username inferred from your host-side username
* `$uid`: the user ID inferred from your host-side user ID
* `$gid`: the group ID inferred from your host-side group ID
* `$DDEV_PHP_VERSION`: the PHP version declared in your project configuration
* `$TARGETARCH`: The build target architecture, like `arm64` or `amd64`
* `$TARGETOS`: The build target operating system (always `linux`)
* `$TARGETPLATFORM`: `linux/amd64` or `linux/arm64` depending on the machine it's been executed on

For example, a Dockerfile might want to build an extension for the configured PHP version like this using `$DDEV_PHP_VERSION` to specify the proper version:

```dockerfile
ENV extension=xhprof
ENV extension_repo=https://github.com/longxinH/xhprof
ENV extension_version=v2.3.8

RUN (apt-get update || true) && DEBIAN_FRONTEND=noninteractive apt-get install -y -o Dpkg::Options::="--force-confnew" --no-install-recommends --no-install-suggests autoconf build-essential libc-dev php-pear php${DDEV_PHP_VERSION}-dev pkg-config zlib1g-dev
RUN mkdir -p /tmp/php-${extension} && cd /tmp/php-${extension} && git clone ${extension_repo} .
WORKDIR /tmp/php-${extension}/extension
RUN git checkout ${extension_version}
RUN phpize
RUN ./configure
RUN make install
RUN echo "extension=${extension}.so" > /etc/php/${DDEV_PHP_VERSION}/mods-available/${extension}.ini
```

An example of using `$TARGETARCH` would be:

```dockerfile
RUN curl --fail -JL -s -o /usr/local/bin/mkcert "https://dl.filippo.io/mkcert/latest?for=linux/${TARGETARCH}"
```

## Adding EOL Versions of PHP

If your project requires multiple versions of PHP—such as using PHP 8.3 but also needing an older, unsupported, unmaintained version like PHP 7.4 for specific scripts—and you don’t want to fully switch to PHP 7.4 with `ddev config --php-version=7.4`, you can install it using the `pre.Dockerfile.*` technique from the previous section.

Create a `.ddev/web-build/pre.Dockerfile.php7.4` file with the following content:

```dockerfile
RUN /usr/local/bin/install_php_extensions.sh "php7.4" "${TARGETARCH}"
```

After restarting the project, you can use PHP 7.4 with the command `ddev exec php7.4 -v`.

## Installing into the home directory

The in-container home directory is rebuilt when you run `ddev restart`, so if you have something that installs into the home directory (like `~/.cache`) you'll want to switch users in the Dockerfile. In this example, `npx playwright install` installs a number of things into `~/.cache`, so we'll switch to the proper user before executing it, and switch back to the `root` user after installation to avoid surprises with any other Dockerfile that may follow.

```Dockerfile
USER $username
# This is an example of creating a file in the home directory
RUN touch ~/${username}-was-here
# `npx playwright` installs lots of things in ~/.cache
RUN npx playwright install
RUN npx playwright install-deps
USER root
```

### Debugging the Dockerfile Build

It can be complicated to figure out what’s going on when building a Dockerfile, and even more complicated when you’re seeing it go by as part of [`ddev start`](../usage/commands.md#start).

1. Use [`ddev ssh`](../usage/commands.md#ssh) first of all to pioneer the steps you want to take. You can do all the things you need to do there and see if it works. If you’re doing something that affects PHP, you may need to `sudo killall -USR2 php-fpm` for it to take effect.
2. Put the steps you pioneered into `.ddev/web-build/Dockerfile` as above.
3. If you can’t figure out what’s failing or why, running `ddev debug rebuild` will show the full output of the build process. You can also run `export DDEV_VERBOSE=true && ddev start` to see what’s happening during the `ddev start` Dockerfile build.
