# Customizing Docker Images

It's common to have a requirement for the web or db images which is not bundled in them by default. There are two easy ways to extend these docker images:

* `webimage_extra_packages` and `dbimage_extra_packages` in .ddev/config.yaml
* An add-on Dockerfile in your project's `.ddev/web-build` or `.ddev/db-build`

## Adding extra Debian packages with webimage_extra_packages and dbimage_extra_packages

You can add extra Debian packages if that's all that is needed with lines like this in `.ddev/config.yaml`:

```yaml
webimage_extra_packages: [php-yaml, php7.3-tidy]
dbimage_extra_packages: [telnet, netcat]

```

Then the additional packages will be built into the containers during `ddev start`

## How to figure out what packages you need

The web container is a Debian image, and its PHP distributions are packaged (thank you!) by [`deb.sury.org`](https://deb.sury.org/).

If you need a PHP extension, most PHP extensions are built in the `deb.sury.org` distribution. You can google the extension you want, or download and search the [Packages](https://packages.sury.org/php/dists/buster/main/binary-amd64/Packages) list from the `sury` distribution. For example, the `bcmath` PHP extension is provided by "php-bcmath". Many packages have version-specific names, for example `php7.3-tidy`.

If you need a package that is *not* a PHP package, you can view and search standard Debian packages at [packages.debian.org/stable](https://packages.debian.org/stable/), or just use google.

To test that a package will do what you want, you can `ddev ssh` and then `sudo apt-get update && sudo apt-get install <package>` to verify that you can install it and you get what you need. A php extension may require `killall -USR2 php-fpm` to take effect. After you've tried that, you can add the package to `webimage_extra_packages`.

## Adding extra Dockerfiles for webimage and dbimage

For more complex requirements, you can add:

* `.ddev/web-build/Dockerfile`
* `.ddev/web-build/Dockerfile.*`
* `.ddev/db-build/Dockerfile`
* `.ddev/db-build/Dockerfile.*`

These files' content will be appended to ddev's own Dockerfile for each image.

Examples of possible Dockerfiles are given in `.ddev/web-build/Dockerfile.example` and `.ddev/db-build/Dockerfile.example` (these examples are created in your project when you `ddev config` the project).

You can use the `.ddev/*-build` directory as the Docker "context" directory as well. So for example if a file named README.txt exists in `.ddev/web-build`, you can use `ADD README.txt /` in the Dockerfile.

An example web image `.ddev/web-build/Dockerfile` might be:

```dockerfile
RUN npm install -g gatsby-cli
```

Another example would be installing phpcs globally (see [Stack Overflow answer](https://stackoverflow.com/questions/61870801/add-global-phpcs-and-drupal-coder-to-ddev-in-custom-dockerfile/61870802#61870802)):

```dockerfile
ENV COMPOSER_HOME=/usr/local/composer

# We try to avoid when possible relying on composer to download global, so in PHPCS case we can use the phar.
RUN curl -L https://squizlabs.github.io/PHP_CodeSniffer/phpcs.phar -o /usr/local/bin/phpcs && chmod +x /usr/local/bin/phpcs
RUN curl -L https://squizlabs.github.io/PHP_CodeSniffer/phpcbf.phar -o /usr/local/bin/phpcbf && chmod +x /usr/local/bin/phpcbf

# If however we need to download a package, we use `cgr` for that.
RUN composer global require consolidation/cgr
RUN $COMPOSER_HOME/vendor/bin/cgr drupal/coder:^8.3.1
RUN $COMPOSER_HOME/vendor/bin/cgr dealerdirect/phpcodesniffer-composer-installer

# Register Drupal's code sniffer rules.
RUN phpcs --config-set installed_paths $COMPOSER_HOME/global/drupal/coder/vendor/drupal/coder/coder_sniffer --verbose
# Make Codesniffer config file writable for ordinary users in container.
RUN chmod 666 /usr/local/bin/CodeSniffer.conf
# Make COMPOSER_HOME writable if regular users need to use it.
RUN chmod -R ugo+rw $COMPOSER_HOME
# Now turn it off, because ordinary users will want to be using the default
ENV COMPOSER_HOME=""
```

**Remember that the Dockerfile is building a docker image that will be used later with ddev.** At the time the Dockerfile is executing, your code is not mounted and the container is not running, it's just being built. So for example, an `npm install` in /var/www/html will not do anything useful because the code is not there at image building time.

### Debugging the Dockerfile build

It can be complicated to figure out what's going on when building a Dockerfile, and even more complicated when you're seeing it go by as part of `ddev start`.

1. Use `ddev ssh` first of all to pioneer the steps you want to take. You can just do all the things you need to do there, and see if it works. If doing something that affects PHP you may need to `sudo killall -USR2 php-fpm` for it to take effect.
2. Put the steps you pioneered into `.ddev/web-build/Dockerfile` as above.
3. If you can't figure out what's failing or why, then `~/.ddev/bin/docker-compose -f .ddev/.ddev-docker-compose-full.yaml build web --no-cache --progress=plain` to see what's happening during the Dockerfile build.
