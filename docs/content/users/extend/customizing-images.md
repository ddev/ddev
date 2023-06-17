# Customizing Docker Images

It’s common to have a requirement for the `web` or `db` images which isn’t bundled with them by default. There are two ways to extend these Docker images:

1. `webimage_extra_packages` and `dbimage_extra_packages` in `.ddev/config.yaml`.
2. An add-on Dockerfile in your project’s `.ddev/web-build` or `.ddev/db-build`.

## Adding Extra Debian Packages with `webimage_extra_packages` and `dbimage_extra_packages`

You can add extra Debian packages with lines like this in `.ddev/config.yaml`:

```yaml
webimage_extra_packages: [php-yaml, php8.2-tidy]
dbimage_extra_packages: [telnet, netcat]
```

Then the additional packages will be built into the containers during [`ddev start`](../usage/commands.md#start).

## Determining What Packages You Need

The `web` container is a Debian image, and its PHP distributions are packaged (thank you!) by [`deb.sury.org`](https://deb.sury.org/).

Most PHP extensions are built within the `deb.sury.org` distribution. You can Google the extension you want, or download and search the [Packages](https://packages.sury.org/php/dists/buster/main/binary-amd64/Packages) list from the `sury` distribution. For example, the `bcmath` PHP extension is provided by `php-bcmath`. Many packages have version-specific names, like `php7.3-tidy`.

If you need a package that is *not* a PHP package, you can view and search standard Debian packages at [packages.debian.org/stable](https://packages.debian.org/stable/), or use Google.

To test that a package will do what you want, you can [`ddev ssh`](../usage/commands.md#ssh) and `sudo apt-get update && sudo apt-get install <package>` to verify that you can install it and you get what you need. A PHP extension may require `killall -USR2 php-fpm` to take effect. After you’ve tried that, you can add the package to [`webimage_extra_packages`](../configuration/config.md#webimage_extra_packages).

## Adding Extra Dockerfiles for `webimage` and `dbimage`

For more complex requirements, you can add:

* `.ddev/web-build/Dockerfile`
* `.ddev/web-build/Dockerfile.*`
* `.ddev/db-build/Dockerfile`
* `.ddev/db-build/Dockerfile.*`

These files’ content will be inserted into the constructed Dockerfile for each image. They are inserted *after* most of the rest of the things that are done to build the image, and are done in alphabetical order, so `Dockerfile` is inserted first, followed by `Dockerfile.*` in alphabetical order.

For certain use cases, you might need to add directives very early on the Dockerfile like proxy settings or SSL termination. You can use `pre.` variants for this that are inserted *before* everything else:

* `.ddev/web-build/pre.Dockerfile.*`
* `.ddev/db-build/pre.Dockerfile.*`

Examine the resultant generated Dockerfile (which you will never edit directly), at `.ddev/.webimageBuild/Dockerfile`. You can force a rebuild with [`ddev debug refresh`](../usage/commands.md#debug-refresh).

Examples of possible Dockerfiles are `.ddev/web-build/Dockerfile.example` and `.ddev/db-build/Dockerfile.example`, created in your project when you run [`ddev config`](../usage/commands.md#config).

You can use the `.ddev/*-build` directory as the Docker “context” directory as well. So for example, if a file named `README.txt` exists in `.ddev/web-build`, you can use `ADD README.txt /` in the Dockerfile.

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

* `$BASE_IMAGE`: the base image, like `ddev/ddev-webserver:v1.21.4`
* `$username`: the username inferred from your host-side username
* `$uid`: the user ID inferred from your host-side user ID
* `$gid`: the group ID inferred from your host-side group ID
* `$DDEV_PHP_VERSION`: the PHP version declared in your project configuration (provided in versions after v1.21.4)

For example, a Dockerfile might want to build an extension for the configured PHP version like this:

```Dockerfile
ENV extension=xhprof
ENV extension_repo=https://github.com/longxinH/xhprof
ENV extension_version=v2.3.8
# For versions <= DDEV v1.21.4 you must also declare DDEV_PHP_VERSION yourself: ENV DDEV_PHP_VERSION=8.1

RUN apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y -o Dpkg::Options::="--force-confnew" --no-install-recommends --no-install-suggests autoconf build-essential libc-dev php-pear php${DDEV_PHP_VERSION}-dev pkg-config zlib1g-dev
RUN mkdir -p /tmp/php-${extension} && cd /tmp/php-${extension} && git clone ${extension_repo} .
WORKDIR /tmp/php-${extension}/extension
RUN git checkout ${extension_version}
RUN phpize
RUN ./configure
RUN make install
RUN echo "extension=${extension}.so" > /etc/php/${DDEV_PHP_VERSION}/mods-available/${extension}.ini
```

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
3. If you can’t figure out what’s failing or why, running `ddev debug refresh` will show the full output of the build process. You can also run `export DDEV_VERBOSE=true && ddev start` to see what’s happening during the `ddev start` Dockerfile build.
