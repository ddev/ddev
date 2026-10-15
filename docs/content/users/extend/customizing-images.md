# Customizing Docker Images

It’s common to have a requirement for the `web` or `db` images which isn’t bundled with them by default. There are two ways to extend these Docker images:

1. `webimage_extra_packages` and `dbimage_extra_packages` in `.ddev/config.yaml`.
2. An add-on Dockerfile in your project’s `.ddev/web-build` or `.ddev/db-build`.

## Adding Extra Debian Packages with `webimage_extra_packages` and `dbimage_extra_packages`

You can add extra Debian packages with lines like this in `.ddev/config.yaml`:

```yaml
# .ddev/config.yaml
webimage_extra_packages: ['php${DDEV_PHP_VERSION}-tidy', 'php${DDEV_PHP_VERSION}-yac']
dbimage_extra_packages: [netcat, telnet, sudo]
```

Then the additional packages will be built into the containers during [`ddev start`](../usage/commands.md#start).

## Adding PHP Extensions

### PHP Extensions supported by `deb.sury.org`

If a PHP extension is supported by the upstream package management from `deb.sury.org`, you'll be able to add it with minimal effort. Test to see if it's available using `ddev exec '(sudo apt-get update || true) && sudo apt-get install php${DDEV_PHP_VERSION}-<extension>'`, for example, `ddev exec '(sudo apt-get update || true) && sudo apt-get install php${DDEV_PHP_VERSION}-imap'`. If that works, then the extension is supported, and you can add `webimage_extra_packages: ["php${DDEV_PHP_VERSION}-<extension>"]` to your `.ddev/config.yaml` file.

### PIE PHP Extensions not supported by `deb.sury.org`

!!!tip "Few people need PIE or PECL extensions"
    Most people don't need to install PHP extensions that aren't supported by `deb.sury.org`, so you only need to go down this path if you have very particular needs.

If a PHP extension is not supported by `deb.sury.org`, you can install it with [PIE](https://github.com/php/pie) (PHP Installer for Extensions) in a `.ddev/web-build/Dockerfile.*` file. Find the extension's package name on [packagist.org/extensions](https://packagist.org/extensions). (This technique can also be used to get newer versions of PHP extensions than are available in the `deb.sury.org` distribution.)

PIE requires PHP 8.1 or later. For older PHP versions, or an extension that isn't on Packagist, use [PECL](#pecl-php-extensions-not-supported-by-debsuryorg) instead.

For example, this `.ddev/web-build/Dockerfile.brotli` installs [brotli](https://packagist.org/packages/kjdev/brotli). To install another extension, change `package`, and list any Debian packages it needs to build in `build_packages`:

```dockerfile
# .ddev/web-build/Dockerfile.brotli
RUN --mount=type=secret,id=github_token <<EOF
    set -eu -o pipefail
    # Package name, optionally with a version constraint, like kjdev/brotli:^0.21
    package=kjdev/brotli
    # Debian packages needed to build it, like "libmemcached-dev zlib1g-dev"
    build_packages=""
    (apt-get update || true)
    DEBIAN_FRONTEND=noninteractive apt-get install -y \
        -o Dpkg::Options::="--force-confold" \
        --no-install-recommends --no-install-suggests \
        build-essential pkg-config "php${DDEV_PHP_VERSION}-dev" ${build_packages}
    curl -fsSL -o /usr/local/bin/pie \
        https://github.com/php/pie/releases/latest/download/pie.phar
    chmod +x /usr/local/bin/pie
    if [ -s /run/secrets/github_token ]; then
        token=$(cat /run/secrets/github_token)
        export COMPOSER_AUTH="{\"github-oauth\":{\"github.com\":\"${token}\"}}"
    fi
    pie install --with-php-config="/usr/bin/php-config${DDEV_PHP_VERSION}" \
        "${package}"
EOF
```

* `--with-php-config` builds the extension for your project's PHP version rather than the image's default one. PIE enables the extension itself.
* For `build_packages`, check the extension's installation instructions and add the `-dev` packages of the libraries it needs. If the build still fails, the `configure` error names what's missing, like `memcached support requires ZLIB` for `zlib1g-dev`, and [packages.debian.org](https://packages.debian.org/) finds the package that provides a missing file.
* DDEV passes your GitHub token, from `DDEV_GITHUB_TOKEN`, `GH_TOKEN`, or `GITHUB_TOKEN`, as the `github_token` [build secret](https://docs.docker.com/build/building/secrets/). PIE reads it from `COMPOSER_AUTH` to avoid GitHub API rate limits. The secret is empty when no token is set. Keep the token in an environment variable of the `RUN` step, as above: writing it to a file would store it in the image.
* `--force-confold` keeps DDEV's PHP configuration files if installing `php${DDEV_PHP_VERSION}-dev` upgrades the PHP packages.

A `.ddev/web-build/Dockerfile.xdebug` that replaces the `deb.sury.org` Xdebug with one built by PIE might look like this:

```dockerfile
# .ddev/web-build/Dockerfile.xdebug
RUN --mount=type=secret,id=github_token <<EOF
    set -eu -o pipefail
    (apt-get update || true)
    apt-get remove -y "php${DDEV_PHP_VERSION}-xdebug" || true
    DEBIAN_FRONTEND=noninteractive apt-get install -y \
        -o Dpkg::Options::="--force-confold" \
        --no-install-recommends --no-install-suggests \
        build-essential pkg-config "php${DDEV_PHP_VERSION}-dev"
    curl -fsSL -o /usr/local/bin/pie \
        https://github.com/php/pie/releases/latest/download/pie.phar
    chmod +x /usr/local/bin/pie
    if [ -s /run/secrets/github_token ]; then
        token=$(cat /run/secrets/github_token)
        export COMPOSER_AUTH="{\"github-oauth\":{\"github.com\":\"${token}\"}}"
    fi
    # ddev xdebug enables it with DDEV's own xdebug.ini, so there is no phpenmod
    pie install --with-php-config="/usr/bin/php-config${DDEV_PHP_VERSION}" \
        --skip-enable-extension xdebug/xdebug
EOF
```

### PECL PHP Extensions not supported by `deb.sury.org`

Use PECL for PHP versions older than 8.1, which aren't supported by PIE, or for an extension release that isn't published on Packagist. Find the extension's package name on [pecl.php.net](https://pecl.php.net/).

For example, this `.ddev/web-build/Dockerfile.brotli` installs [brotli](https://pecl.php.net/package/brotli). To install another extension, change `extension`, and list any Debian packages it needs to build in `build_packages`:

```dockerfile
# .ddev/web-build/Dockerfile.brotli
RUN <<EOF
    set -eu -o pipefail
    # Package name, optionally with a version, like brotli-0.21.0
    extension=brotli
    # Debian packages needed to build it, like "libmemcached-dev zlib1g-dev"
    build_packages=""
    (apt-get update || true)
    DEBIAN_FRONTEND=noninteractive apt-get install -y \
        -o Dpkg::Options::="--force-confold" \
        --no-install-recommends --no-install-suggests \
        build-essential php-pear "php${DDEV_PHP_VERSION}-dev" ${build_packages}
    # yes '' accepts the default answer to every question pecl asks
    pecl -d php_suffix="${DDEV_PHP_VERSION}" install "${extension}" < <(yes '') \
        | tee /tmp/pecl.log
    # pecl prints the line that enables the extension, like extension=brotli.so
    ini=$(grep -o '[a-z_]*extension=[^"]*\.so' /tmp/pecl.log | tail -1)
    name=$(basename "${ini#*=}" .so)
    echo "${ini}" > "/etc/php/${DDEV_PHP_VERSION}/mods-available/${name}.ini"
    phpenmod -v "${DDEV_PHP_VERSION}" "${name}"
EOF
```

`-d php_suffix` builds the extension for your project's PHP version rather than the image's default one. Find what goes in `build_packages` [the same way as for PIE](#pie-php-extensions-not-supported-by-debsuryorg). To answer questions from `pecl` with something other than the defaults, pass them with `--configureoptions`, like `pecl install --configureoptions 'enable-redis-igbinary="yes"' redis`.

To replace an extension that DDEV enables itself, Xdebug with [`ddev xdebug`](../usage/commands.md#xdebug) or XHProf with [`ddev xhprof`](../usage/commands.md#xhprof), remove the `deb.sury.org` package first. For example, this `.ddev/web-build/Dockerfile.xdebug` pins Xdebug 3.1.6, the last release supporting PHP 7.2 to 7.4:

```dockerfile
# .ddev/web-build/Dockerfile.xdebug
RUN <<EOF
    set -eu -o pipefail
    # xdebug or xhprof, optionally with a version
    extension=xdebug-3.1.6
    (apt-get update || true)
    apt-get remove -y "php${DDEV_PHP_VERSION}-${extension%%-*}" || true
    DEBIAN_FRONTEND=noninteractive apt-get install -y \
        -o Dpkg::Options::="--force-confold" \
        --no-install-recommends --no-install-suggests \
        build-essential php-pear "php${DDEV_PHP_VERSION}-dev"
    pecl -d php_suffix="${DDEV_PHP_VERSION}" install "${extension}" < <(yes '')
    # ddev xdebug and ddev xhprof enable it with DDEV's own ini file,
    # so there is no phpenmod
EOF
```

## Adding Locales

The web image ships by default with a small number of locales, which work for most usages, including
`en_CA`, `en_US`, `en_GB`, `es_ES`, `es_MX`, `pt_BR`, `pt_PT`, `de_DE`, `de_AT`, `fr_CA`, `fr_FR`, `ja_JP`, and `ru_RU`.

If you need other locales, you can install all of them by adding `locales-all` to your `webimage_extra_packages`. For example, in `.ddev/config.yaml`:

```yaml
# .ddev/config.yaml
webimage_extra_packages: [locales-all]
```

## Adding Extra Dockerfiles for `webimage` and `dbimage`

For more complex requirements, add your own Dockerfile content to `.ddev/web-build` (for `webimage`) or `.ddev/db-build` (for `dbimage`). DDEV merges those files with its own build steps into one generated Dockerfile per image, at `.ddev/.webimageBuild/Dockerfile` and `.ddev/.dbimageBuild/Dockerfile`, which you never edit directly.

DDEV writes example files for you to copy and rename:

* `.ddev/web-build/Dockerfile.example` and `.ddev/db-build/Dockerfile.example` in the project, when you run [`ddev config`](../usage/commands.md#config)
* `$HOME/.ddev/web-build/pre.Dockerfile.example` and `$HOME/.ddev/db-build/pre.Dockerfile.example` for [global Dockerfiles](#global-dockerfiles)

!!!warning "The Dockerfile builds an image, it doesn’t run in your project"
    While the Dockerfile is executing, your code is not mounted and the container is not running, the image is being built. So for example, an `npm install` in `/var/www/html` will not do anything to your project because the code is not there at image building time.

### Insertion Order

The filename decides where its content lands in the generated Dockerfile, and [global files](#global-dockerfiles) in `$HOME/.ddev/*-build` are always inserted before the project’s files in `.ddev/*-build`:

| # | Content                                                                                       | Notes                                                                                                                                                                                    |
|---|-----------------------------------------------------------------------------------------------|------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| 1 | `prepend.Dockerfile`, `prepend.Dockerfile.*`<br>([global](#global-dockerfiles), then project) | Above DDEV’s `FROM` line, for [multi-stage builds](#multi-stage-builds). Only `$BASE_IMAGE` is declared this early, see [Build Time Environment Variables](#build-time-environment-variables) |
| 2 | DDEV’s `FROM $BASE_IMAGE`, build arguments, and user creation                                  | The in-image user mirrors your host user, using `$username`, `$uid`, and `$gid`                                                                                                                                                                                        |
| 3 | `pre.Dockerfile`, `pre.Dockerfile.*`<br>([global](#global-dockerfiles), then project)         | For directives that have to come early, like proxy settings, SSL termination, [CA certificates](#global-dockerfiles), or [EOL PHP versions](#adding-eol-versions-of-php)                 |
| 4 | DDEV’s own build steps                                                                        | PHP version, `webimage_extra_packages` or `dbimage_extra_packages`, Composer update, database clients                                                                                     |
| 5 | `Dockerfile`, `Dockerfile.*`<br>([global](#global-dockerfiles), then project)                 | The usual choice for most customizations                                                                                                                                                 |
| 6 | DDEV’s finishing steps                                                                        | Permission fixes on the web image, which have to run after everything else                                                                                                               |

Within each group, files are inserted in alphabetical order, so `Dockerfile` comes first, then `Dockerfile.*` alphabetically.

To see the result, read the generated Dockerfile, or force a rebuild with [`ddev restart --no-cache`](../usage/commands.md#restart) or [`ddev utility rebuild`](../usage/commands.md#utility-rebuild), which shows the whole build output for debugging.

### Rebuilding Only What Changed

Docker caches each step of the image build. When a step changes, Docker rebuilds it and every step after it, and reuses the rest. `ddev restart --no-cache` and `ddev utility rebuild` skip the cache and rebuild everything, including `webimage_extra_packages`, which can take minutes.

A step that installs the latest version of a tool doesn't change when a new version comes out, so the cache keeps the old one. Pin the version instead. When you change it, `ddev restart` rebuilds only from that step on:

```dockerfile
# .ddev/web-build/Dockerfile.gatsby
ARG GATSBY_CLI_VERSION=5.14.0
RUN npm install -g gatsby-cli@${GATSBY_CLI_VERSION}
```

### Copying Files into the Image

The `.ddev/*-build` directory is the Docker "context", so if a file named `file.txt` exists in `.ddev/web-build`, you can use `COPY file.txt /` in the Dockerfile.

### Examples

This `.ddev/web-build/Dockerfile.phpcs` installs `phpcs` and `phpcbf` globally, with the Drupal coding standards from [`drupal/coder`](https://packagist.org/packages/drupal/coder):

```dockerfile
# .ddev/web-build/Dockerfile.phpcs
ARG COMPOSER_HOME=/usr/local/composer
ARG DRUPAL_CODER_VERSION=9.0.1
RUN composer global config --no-interaction \
        allow-plugins.dealerdirect/phpcodesniffer-composer-installer true && \
    composer global require --no-interaction \
        "drupal/coder:${DRUPAL_CODER_VERSION}" && \
    ln -s "${COMPOSER_HOME}/vendor/bin/phpcs" \
        "${COMPOSER_HOME}/vendor/bin/phpcbf" /usr/local/bin/
```

### Multi-Stage Builds

[Multi-stage builds](https://docs.docker.com/build/building/multi-stage/) let you copy files from another image, so the web image gets a tool without what it took to build or package it. They need a `prepend.` file, because the extra `FROM` statement has to be on top of the generated Dockerfile. For example, to copy [Bun](https://bun.sh/) from its official image, add a `.ddev/web-build/prepend.Dockerfile.bun`:

```dockerfile
# .ddev/web-build/prepend.Dockerfile.bun
ARG BUN_VERSION=1.4.3
FROM oven/bun:${BUN_VERSION} AS bun
```

And then a `.ddev/web-build/Dockerfile.bun`:

```dockerfile
# .ddev/web-build/Dockerfile.bun
COPY --from=bun /usr/local/bin/bun /usr/local/bin/bun
RUN ln -s bun /usr/local/bin/bunx
```

### Project-Specific Utilities

DDEV installs these utilities when building the web image for the configured project type:

| Project type | Utilities |
| --- | --- |
| `wordpress`, `wp-bedrock` | WP-CLI (`wp`, `wp-cli`) |
| `magento` | `magerun` and Bash completion |
| `magento2` | `magerun2` and Bash completion |
| `drupal6`, `drupal7` | Drush 8 |
| `backdrop` | Drush 8 and the Backdrop Drush extension |
| `symfony` | Symfony CLI |
| `shopware6` | Shopware CLI |

Building without a cached image downloads the required utilities. Subsequent starts reuse the built image. After changing the project type, run `ddev restart` to rebuild with the corresponding utilities. Modern Drupal projects should install Drush through their project's Composer dependencies.

To use a utility with another project type, install it in a custom Dockerfile. For example, add `.ddev/web-build/Dockerfile.wp-cli` to make WP-CLI available in a `php` project:

```dockerfile
# .ddev/web-build/Dockerfile.wp-cli
RUN curl --fail -sSL -o /usr/local/bin/wp-cli \
    https://raw.githubusercontent.com/wp-cli/builds/gh-pages/phar/wp-cli.phar && \
    chmod +x /usr/local/bin/wp-cli && \
    ln -sf /usr/local/bin/wp-cli /usr/local/bin/wp
```

Run `ddev restart`, then use `ddev exec wp --info`. The `ddev wp` command is available for the WordPress and Bedrock project types.

### Global Dockerfiles

The same files work in `$HOME/.ddev/web-build/` and `$HOME/.ddev/db-build/`, where they apply to every project on the machine, which is handy for things like corporate certificates or private package registries. Global files are inserted _before_ the project’s own files, and the global directory is a Docker “context” too, where a project file of the same name wins.

!!!tip "Your global directory may live elsewhere"
    See [Global Files](../usage/architecture.md#global-files), it isn’t always `$HOME/.ddev`.

For example, to install a custom CA certificate in every project’s web image:

```bash
cat > $HOME/.ddev/web-build/pre.Dockerfile.vpn << 'EOF'
COPY my-ca.crt /usr/local/share/ca-certificates/
RUN update-ca-certificates
EOF
cp /path/to/my-ca.crt $HOME/.ddev/web-build/my-ca.crt
ddev restart
```

A `pre.` file is used here so the certificate is in place before DDEV’s own build steps run.

### Build Time Environment Variables

The following environment variables are available for the web Dockerfile to use at build time:

* `$BASE_IMAGE`: the base image, like `ddev/ddev-webserver:v1.24.0` ([global scope](https://docs.docker.com/build/building/variables/#scoping))
* `$username`: the username inferred from your host-side username
* `$uid`: the user ID inferred from your host-side user ID
* `$gid`: the group ID inferred from your host-side group ID
* `$DDEV_PHP_VERSION`: the PHP version declared in your project configuration
* `$TARGETARCH`: The build target architecture, like `arm64` or `amd64`
* `$TARGETOS`: The build target operating system (always `linux`)
* `$TARGETPLATFORM`: `linux/amd64` or `linux/arm64` depending on the machine it's been executed on

!!!warning "Only `$BASE_IMAGE` is automatically available in `prepend.Dockerfile*` variants"
    If you need to use any of the other variables you will need to manually add them to your `prepend.Dockerfile*` files using [ARG](https://docs.docker.com/reference/dockerfile/#arg) instructions.

The [PIE](#pie-php-extensions-not-supported-by-debsuryorg) and [PECL](#pecl-php-extensions-not-supported-by-debsuryorg) examples use `$DDEV_PHP_VERSION` to build an extension for the configured PHP version.

For example, this `.ddev/web-build/Dockerfile.hugo` uses `$TARGETARCH` to download [Hugo](https://gohugo.io/) for the architecture being built:

```dockerfile
# .ddev/web-build/Dockerfile.hugo
ARG HUGO_VERSION=0.168.0
RUN url="https://github.com/gohugoio/hugo/releases/download/v${HUGO_VERSION}" && \
    curl -fsSL "${url}/hugo_${HUGO_VERSION}_linux-${TARGETARCH}.tar.gz" \
    | tar -xz -C /usr/local/bin hugo
```

## DDEV Labels on Images

A DDEV image can carry two labels that look similar and mean different things. `docker inspect` on a locally built web image will show both, usually with different values, which is expected:

| Label | Set by | Means |
| -- | -- | -- |
| `com.ddev.image-tag` | baked in when the image is built | The tag this image was published as. Inherited through `FROM`, so a derived image keeps the tag of the DDEV image it descends from, even after it is given its own name and tag. |
| `com.ddev.webtag` | the running DDEV CLI | Which DDEV built or started this resource. Applied to containers, networks, and the build sections DDEV generates, so it always reflects the DDEV you are running rather than the image's origin. |

So a project's own web image, built from `.ddev/web-build`, inherits `com.ddev.image-tag` from the `ddev-webserver` base it was built on, and gets `com.ddev.webtag` from whichever DDEV built it. When you pin a [`webimage`](../configuration/config.md#webimage) or [`dbimage`](../configuration/config.md#dbimage), only `com.ddev.image-tag` tells you which generation of DDEV images it came from, which is what DDEV checks to warn you that a pinned image has gone stale.

## Seeding a Custom Starter Database in `dbimage`

If you publish a derived `dbimage` (for example a CI-built image with your production dataset baked in, so teammates or preview environments get a ready-to-use database with no import step), bake it in as `/mysqlbase/custom/base_db.zst` using a `.ddev/db-build/Dockerfile.seed`:

```dockerfile
# .ddev/db-build/Dockerfile.seed
COPY base_db.zst /mysqlbase/custom/base_db.zst
```

The database container uses this seed the first time its data volume is created (a brand-new project, or after `ddev delete` and `ddev start`), instead of the stock DDEV starter database. A project-level [seed snapshot](../usage/database-management.md#seeding-a-fresh-database-from-a-snapshot) wins over it — so a teammate can still override your baked-in seed for their own project just by dropping a `seed` snapshot into `.ddev/db_snapshots`, without rebuilding the image.

A seed can also be baked in uncompressed, as `base_db.mbstream` (MariaDB) or `base_db.xbstream` (MySQL, or MariaDB 5.5/10.0) — the raw `mariabackup`/`xtrabackup` stream with no compression stage. This is an unusual, deliberate trade-off (see [Uncompressed Snapshots](../usage/database-management.md#uncompressed-snapshots)): it trades a much larger image layer for zero first-boot decompression cost, and only makes sense if you've already decided that trade-off is worth it.

When `ddev start` seeds a fresh database volume from a baked-in seed, it says so and warns that a large one may take a while:

```
Initializing new database volume from /mysqlbase/custom/base_db.zst baked into dbimage ddev/ddev-dbserver-mariadb-11.8:v1.25.0...
With a large database this may take a long time.
```

!!!note
    Only supported for `mysql` and `mariadb` database types (excluding the very old, EOL `mysql:5.5` and `mariadb:5.5`). PostgreSQL projects use the stock upstream `postgres` image and its own startup process, which this seeding mechanism doesn't hook into, so a baked-in seed is silently ignored.

## Adding EOL Versions of PHP

If your project requires multiple versions of PHP—such as using PHP 8.3 but also needing an older, unsupported, unmaintained version like PHP 7.4 for specific scripts—and you don’t want to fully switch to PHP 7.4 with `ddev config --php-version=7.4`, you can install it using the `pre.Dockerfile.*` technique from the previous section.

Create a `.ddev/web-build/pre.Dockerfile.php7.4` file with the following content:

```dockerfile
# .ddev/web-build/pre.Dockerfile.php7.4
RUN /usr/local/bin/install_php_extensions.sh "php7.4" "${TARGETARCH}"
```

After restarting the project, you can use PHP 7.4 with the command `ddev exec php7.4 -v`.

## Installing into the home directory

The in-container home directory is rebuilt when you run `ddev restart`, so if you have something that installs into the home directory (like `~/.cache`) you'll want to switch users in the Dockerfile. In this example, `npx playwright install` puts browsers into `~/.cache`, so this `.ddev/web-build/Dockerfile.playwright` switches to the in-container user before running it, and back to `root` afterwards to avoid surprises with any other Dockerfile that may follow. `--with-deps` installs the Debian packages the browsers need with `sudo`. For files that change while the container runs, see [Persisting Changes Across Restarts](in-container-configuration.md#persisting-changes-across-restarts).

```dockerfile
# .ddev/web-build/Dockerfile.playwright
# The version in your project's package.json, so the browsers match it
ARG PLAYWRIGHT_VERSION=1.64.1
USER $username
RUN npx --yes "playwright@${PLAYWRIGHT_VERSION}" install --with-deps
USER root
```

### Debugging the Dockerfile Build

It can be complicated to figure out what’s going on when building a Dockerfile, and even more complicated when you’re seeing it go by as part of [`ddev start`](../usage/commands.md#start).

1. Use [`ddev ssh`](../usage/commands.md#ssh) first of all to pioneer the steps you want to take. You can do all the things you need to do there and see if it works. If you’re doing something that affects PHP, you may need to `sudo killall -USR2 php-fpm` for it to take effect.
2. Put the steps you pioneered into a `.ddev/web-build/Dockerfile.*` file as above.
3. If you can’t figure out what’s failing or why, running `ddev utility rebuild` will show the full output of the build process. You can also run `export DDEV_VERBOSE=true && ddev start` to see what’s happening during the `ddev start` Dockerfile build.
