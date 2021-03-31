## Customizing Docker Images

It's common to have a requirement for the web or db images which is not bundled in them by default. There are two easy ways to extend these docker images:

* `webimage_extra_packages` and `dbimage_extra_packages` in .ddev/config.yaml
* An add-on Dockerfile in your project's `.ddev/web-build` or `.ddev/db-build`

### Adding extra Debian packages with webimage_extra_packages and dbimage_extra_packages

You can add extra Debian packages if that's all that is needed with lines like this in `.ddev/config.yaml`:

```yaml
webimage_extra_packages: [php-yaml, php7.3-tidy]
dbimage_extra_packages: [telnet, netcat]

```

Then the additional packages will be built into the containers during `ddev start`

### How to figure out what packages you need

The web container is a Debian 10 Buster image, and its PHP distributions are packaged (thank you!) by [deb.sury.org](https://deb.sury.org/).

If you need a PHP extension, most PHP extensions are built in the deb.sury.org distribution. You can google the extension you want, or download and search the [Packages](https://packages.sury.org/php/dists/buster/main/binary-amd64/Packages) list from the sury distribution. For example, the "bcmath" PHP extension is provided by "php-bcmath". Many packages have version-specific names, for example `php7.3-tidy`.

If you need a package that is *not* a PHP package, you can view and search standard Debian packages at [packages.debian.org/stable](https://packages.debian.org/stable/), or just use google.

To test that a package will do what you want, you can `ddev ssh` and then `sudo apt-get update && sudo apt-get install <package>` to verify that you can install it and you get what you need. A php extension may require `killall -HUP php-fpm` to take effect. After you've tried that, you can add the package to `webimage_extra_packages`.

### Adding extra Dockerfiles for webimage and dbimage

For more complex requirements, you can add .ddev/web-build/Dockerfile or .ddev/db-build/Dockerfile.

Examples of possible Dockerfiles are given in `.ddev/web-build/Dockerfile.example` and `.ddev/db-build/Dockerfile.example` (These examples are created in your project when you `ddev config` the project.)

You can use the .ddev/*-build/ directory as the Docker "context" directory as well. So for example if a file named README.txt exists in .ddev/web-build, you can use `ADD README.txt /` in the Dockerfile.

An example web image `.ddev/web-build/Dockerfile` might be:

```dockerfile
ARG BASE_IMAGE
FROM $BASE_IMAGE
RUN npm install --global gulp-cli
ADD README.txt /
```

Another example would be changing the installed nodejs version to a preferred version, for example nodejs 12:

```dockerfile
ARG BASE_IMAGE
FROM $BASE_IMAGE
# Install whatever nodejs version you want
ENV NODE_VERSION=12
RUN sudo apt-get remove -y nodejs
RUN curl -sSL --fail https://deb.nodesource.com/setup_${NODE_VERSION}.x | bash -
RUN apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y -o Dpkg::Options::="--force-confold" --no-install-recommends --no-install-suggests nodejs

```

**Note that if a Dockerfile is provided, any config.yaml `webimage_extra_packages`, `dbimage_extra_packages`, or `composer_version` statements will be ignored.** If you need to add packages as well as other custom configuration, add them to your Dockerfile with a line like

```dockerfile
RUN apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y -o Dpkg::Options::="--force-confold" --no-install-recommends --no-install-suggests php7.3-tidy
```

If you need to set an explicit composer version in this situation use a command like

```dockerfile
RUN composer self-update --2
```

**Remember that the Dockerfile is building a docker image that will be used later with ddev.** At the time the Dockerfile is executing, your code is not mounted and the container is not running, it's just being built. So for example, an `npm install` in /var/www/html will not do anything useful because the code is not there at image building time.
