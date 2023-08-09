# Extending and Customizing Environments

DDEV provides several ways to customize and extend project environments.

## Changing PHP Version

The project's `.ddev/config.yaml` file defines the PHP version to use. The [`php_version`](../configuration/config.md#php_version) can be changed to `5.6`, `7.0`, `7.1`, `7.2`,  `7.3`, `7.4`, `8.0`, `8.1`, `8.2`, or `8.3`. The current default is `8.1`.

### Older Versions of PHP

[Support for older versions of PHP is available on ddev-contrib](https://github.com/ddev/ddev-contrib/blob/master/docker-compose-services/old_php) via [custom docker-compose files](custom-compose-files.md).

## Changing Web Server Type

DDEV supports nginx with php-fpm by default (`nginx-fpm`), and Apache with php-fpm (`apache-fpm`). You can change this with the [`webserver_type`](../configuration/config.md#webserver_type) config option, or using the [`ddev config`](../usage/commands.md#config) command with the `--webserver-type` flag.

## Adding Services to a Project

DDEV provides everything you need to build a modern PHP application on your local machine. More complex web applications, however, often require integration with services beyond the usual requirements of a web and database server—maybe Apache Solr, Redis, Varnish, or many others. While DDEV likely won’t ever provide all of these additional services out of the box, it’s designed to provide simple ways to customize the environment and meet your project’s needs without reinventing the wheel.

A collection of vetted service configurations is available in the [Additional Services Documentation](additional-services.md).

If you need to create a service configuration for your project, see [Defining Additional Services with Docker Compose](custom-compose-files.md).

## Using Node.js with DDEV

There are many ways to deploy Node.js in any project, so DDEV tries to let you set up any possibility you can come up with.

* You can choose the Node.js version you want to use in `.ddev/config.yaml` with [`nodejs_version`](../configuration/config.md#nodejs_version).
* [`ddev nvm`](../usage/commands.md#nvm) gives you the full capabilities of [Node Version Manager](https://github.com/nvm-sh/nvm).
* [`ddev npm`](../usage/commands.md#npm) and [`ddev yarn`](../usage/commands.md#yarn) provide shortcuts to the `npm` and `yarn` commands inside the container, and their caches are persistent.
* You can run Node.js daemons using [`web_extra_daemons`](#running-extra-daemons-in-the-web-container).
* You can expose Node.js ports via `ddev-router` by using [`web_extra_exposed_ports`](#exposing-extra-ports-via-ddev-router).
* You can manually run Node.js scripts using [`ddev exec <script>`](../usage/commands.md#exec) or `ddev exec nodejs <script>`.

!!!tip "Please share your techniques!"
    There are several ways to share your favorite Node.js tips and techniques. Best are [ddev-get add-ons](additional-services.md#additional-service-configurations-and-add-ons-for-ddev), [Stack Overflow](https://stackoverflow.com/tags/ddev), and [ddev-contrib](https://github.com/ddev/ddev-contrib).

## Running Extra Daemons in the Web Container

There are several ways to run processes inside the `web` container.

1. Manually execute them as needed, with [`ddev exec`](../usage/commands.md#exec), for example.
2. Run them with a `post-start` [hook](../configuration/hooks.md).
3. Run them automatically using `web_extra_daemons`.

### Running Extra Daemons with `post-start` Hook

Daemons can be run with a `post-start` `exec` hook or automatically started using `supervisord`.

A simple `post-start` exec hook in `.ddev/config.yaml` might look like:

```yaml
hooks:
  post-start:
    - exec: "nohup php --docroot=/var/www/html/something -S 0.0.0.0:6666 &"
```

### Running Extra Daemons Using `web_extra_daemons`

If you need extra daemons to start up automatically inside the web container, you can easily add them using [`web_extra_daemons`](../configuration/config.md#web_extra_daemons) in `.ddev/config.yaml`.

You might be running Node.js daemons that serve a particular purpose, like `browsersync`, or more general daemons like a `cron` daemon.

For example, you could use this configuration to run two instances of the Node.js HTTP server for different directories:

```yaml
web_extra_daemons:
  - name: "http-1"
    command: "/var/www/html/node_modules/.bin/http-server -p 3000"
    directory: /var/www/html
  - name: "http-2"
    command: "/var/www/html/node_modules/.bin/http-server /var/www/html/sub -p 3000"
    directory: /var/www/html
```

* `directory` should be the absolute path inside the container to the directory where the daemon should run.
* `command` is best as a simple binary with its arguments, but Bash features like `cd` or `&&` work. If the program to be run is not in the `ddev-webserver` `$PATH` then it should have the absolute in-container path to the program to be run, like `/var/www/html/node_modules/.bin/http-server`.
* `web_extra_daemons` is a shortcut for adding a configuration to `supervisord`, which organizes daemons inside the web container. If the default settings are inadequate for your use, you can write a [complete config file for your daemon](#explicit-supervisord-configuration-for-additional-daemons).
* Your daemon is expected to run in the foreground, not to daemonize itself, `supervisord` will take care of that.
* To see the results of the attempt to start your daemon, see [`ddev logs`](../usage/commands.md#logs) or `docker logs ddev-<project>-web`.

## Exposing Extra Ports via `ddev-router`

If your `web` container has additional HTTP servers running inside it on different ports, those can be exposed using [`web_extra_exposed_ports`](../configuration/config.md#web_extra_exposed_ports) in `.ddev/config.yaml`. For example, this configuration would expose a `node-vite` HTTP server running on port 3000 inside the `web` container, via `ddev-router`, to ports 9998 (HTTP) and 9999 (HTTPS), so it could be accessed via `https://<project>.ddev.site:9999`:

```yaml
web_extra_exposed_ports:
  - name: node-vite
    container_port: 3000
    http_port: 9998
    https_port: 9999
```

The configuration below would expose a Node.js server running in the `web` container on port 3000 as `https://<project>.ddev.site:3000` and a “something” server running in the web container on port 4000 as `https://<project>.ddev.site:4000`:

```yaml
web_extra_exposed_ports:
  - name: nodejs
    container_port: 3000
    http_port: 2999
    https_port: 3000
  - name: something
    container_port: 4000
    https_port: 4000
    http_port: 3999
```

!!!warning "Fill in all three fields even if you don’t intend to use the `https_port`!"
    If you don’t add `https_port`, then it defaults to `0` and `ddev-router` will fail to start.

## Exposing Extra Non-HTTP Ports

While the `web_extra_exposed_ports` gracefully handles running multiple DDEV projects at the same time, it can't forward ports for non-HTTP TCP or UDP daemons. Instead, ports can be added in a `docker-compose.*.yaml` file. This file does not need to specify an additional services. For example, this configuration exposes port 5900 for a VNC server.

In `.ddev/docker-compose.vnc.yaml`:

```yaml
services:
  web:
    ports:
      - "5900:5900"
```

If multiple projects declare the same port, only the first project will be able to start successfully. Consider making services like this disabled by default, especially if they aren't needed in day to day use.

## Providing Custom Environment Variables to a Container

You can set custom environment variables in several places:

* The project’s [`web_environment`](../configuration/config.md#web_environment) setting in `.ddev/config.yaml` or `.ddev/config.*.yaml`:

    ```yaml
    web_environment:
    - MY_ENV_VAR=someval
    - MY_OTHER_ENV_VAR=someotherval
    ```

* The global `web_environment` setting in `.ddev/global_config.yaml`.

* An optional, project-level `.ddev/.env` file, which could look something like this:

    ```
    MY_ENV_VAR='someval'
    MY_OTHER_ENV_VAR='someotherval'
    ```

If you’d rather use the CLI to set the project or global `web_environment` value, you can use the [`ddev config`](../usage/commands.md#config) command:

```sh
# Set MY_ENV_VAR for the project
ddev config --web-environment-add="MY_ENV_VAR=someval"

# Set MY_ENV_VAR globally
ddev config global --web-environment-add="MY_ENV_VAR=someval
```

You can use the `--web-environment` flag to overwrite existing values rather than adding them.

!!!warning "Don’t check in sensitive values!"
    Sensitive variables like API keys should not be checked in with your project. Typically you might use an `.env` file and _not_ check that in, but offer `.env.example` with expected keys that don’t have values. Some use global configuration for sensitive values, as that’s not normally checked in either.

### Altering the In-Container `$PATH`

Sometimes it’s easiest to put the command you need into the existing `$PATH` using a symbolic link rather than changing the in-container `$PATH`. For example, the project `bin` directory is already included the `$PATH`. So if you have a command you want to run that’s not already in the `$PATH`, you can add a symlink.

Examples:

* On Craft CMS, the `craft` script is often in the project root, which is not in the `$PATH`. But if you `mkdir bin && ln -s craft bin/craft` you should be able to run `ddev exec craft`. (Note however that `ddev craft` takes care of this for you.)
* On projects where the `vendor` directory is not in the project root (Acquia projects, for example, have `composer.json` and `vendor` in the `docroot` directory), you can `mkdir bin && ln -s docroot/vendor/bin/drush bin/drush` to put `drush` in your `$PATH`. (With projects like this, make sure to set `composer_root: docroot` so that `ddev composer` works properly.)

You can also modify the `PATH` environment variable by adding a script to `<project>/.ddev/homeadditions/.bashrc.d/` or (global) `~/.ddev/homeadditions/.bashrc.d/`. For example, if your project vendor directory is not in the expected place (`/var/www/html/vendor/bin`) you can add a `<project>/.ddev/homeadditions/.bashrc.d/path.sh`:

```bash
export PATH=$PATH:/var/www/html/somewhereelse/vendor/bin
```

## Custom nginx Configuration

When you run [`ddev restart`](../usage/commands.md#restart) using `nginx-fpm`, DDEV creates a configuration customized to your project type in `.ddev/nginx_full/nginx-site.conf`. You can edit and override the configuration by removing the `#ddev-generated` line and doing whatever you need with it. After each change, run `ddev restart`. (For updates without restart, see [Troubleshooting nginx Configuration](#troubleshooting-nginx-configuration).)

You can also have more than one config file in the `.ddev/nginx_full` directory, and each will be loaded when DDEV starts. This can be used for [serving multiple docroots](#multiple-docroots-in-nginx-advanced) and other techniques.

### Troubleshooting nginx Configuration

* Any errors in your configuration may cause the `web` container to fail and try to restart. If you see that behavior, use [`ddev logs`](../usage/commands.md#logs) to diagnose.
* The configuration is copied into the container during restart. Therefore it is not possible to edit the host file for the changes to take effect. You may want to edit the file directly inside the container at `/etc/nginx/sites-enabled/`. (For example, run [`ddev ssh`](../usage/commands.md#ssh) to get into the container.)
* You can run `ddev exec nginx -t` to test whether your configuration inside the container is valid. (Or run [`ddev ssh`](../usage/commands.md#ssh) and run `nginx -t`.)
* You can reload the nginx configuration by running either [`ddev restart`](../usage/commands.md#restart) or editing the configuration inside the container at `/etc/nginx/sites-enabled/` and running `ddev exec nginx -s reload` on the host system (inside the container run `nginx -s reload`).
* The alias `Alias "/phpstatus" "/var/www/phpstatus.php"` is required for the health check script to work.

### Multiple Docroots in nginx (Advanced)

It’s easiest to have different web servers in different DDEV projects, and DDEV projects can [easily communicate with each other](../usage/faq.md), but some sites require more than one docroot for a single project codebase. Sometimes this is because there’s an API in the same codebase but using different code, or different code for different languages, etc.

The generated `.ddev/nginx_full/seconddocroot.conf.example` demonstrates how to do this. You can create as many of these as you want: change the `servername` and the `root` and customize as needed.

### nginx Snippets

To add an nginx snippet to the default config, add an nginx config file as `.ddev/nginx/<something>.conf`.

For example, to make all HTTP URLs redirect to their HTTPS equivalents you might add `.ddev/nginx/redirect.conf` with this stanza:

```
    if ($http_x_forwarded_proto = "http") {
      return 301 https://$host$request_uri;
    }
```

After adding a snippet, run `ddev restart` to make it take effect.

## Custom Apache Configuration

If you’re using [`webserver_type: apache-fpm`](../configuration/config.md#webserver_type) in your `.ddev/config.yaml`, you can override the default site configuration by editing or replacing the DDEV-provided `.ddev/apache/apache-site.conf` configuration.

* Edit the `.ddev/apache/apache-site.conf`.
* Add your configuration changes.
* Save your configuration file and run [`ddev restart`](../usage/commands.md#restart). If you encounter issues with your configuration or the project fails to start, use [`ddev logs`](../usage/commands.md#logs) to inspect the logs for possible Apache configuration errors.
* Use `ddev exec apachectl -t` to do a general Apache syntax check.
* The alias `Alias "/phpstatus" "/var/www/phpstatus.php"` is required for the health check script to work.
* Any errors in your configuration may cause the `web` container to fail. If you see that behavior, use `ddev logs` to diagnose.

!!!warning "Important!"
    Changes to `.ddev/apache/apache-site.conf` take place on a [`ddev restart`](../usage/commands.md#restart). You can also `ddev exec apachectl -k graceful` to reload the Apache configuration.

## Custom PHP Configuration (`php.ini`)

You can provide additional PHP configuration for a project by creating a directory called `.ddev/php/` and adding any number of `*.ini` PHP configuration files.

You should generally limit your override to any specific option(s) you need to customize. Every file in `.ddev/php/` will be copied into `/etc/php/[version]/(cli|fpm)/conf.d`, so it’s possible to replace files that already exist in the container. Common usage is to put custom overrides in a file called `my-php.ini`. Make sure you include the section header that goes with each item (like `[PHP]`).

One interesting implication of this behavior is that it’s possible to disable extensions by replacing the configuration file that loads them. For instance, if you were to create an empty file at `.ddev/php/20-xdebug.ini`, it would replace the configuration that loads Xdebug, which would cause Xdebug to not be loaded!

To load the new configuration, run [`ddev restart`](../usage/commands.md#restart).

An example file in `.ddev/php/my-php.ini` might look like this:

```ini
[PHP]
max_execution_time = 240;
```

## Custom MySQL/MariaDB configuration (`my.cnf`)

You can provide additional MySQL/MariaDB configuration for a project by creating a directory called `.ddev/mysql/` and adding any number of `*.cnf` MySQL configuration files. These files will be automatically included when MySQL is started. Make sure that the section header is included in the file.

An example file in `.ddev/mysql/no_utf8mb4.cnf` might be:

```
[mysqld]
collation-server = utf8_general_ci
character-set-server = utf8
innodb_large_prefix=false
```

To load the new configuration, run [`ddev restart`](../usage/commands.md#restart).

## Custom PostgreSQL Configuration

If you’re using PostgreSQL, a default `posgresql.conf` is provided in `.ddev/postgres/postgresql.conf`. If you need to alter it, remove the `#ddev-generated` line and [`ddev restart`](../usage/commands.md#restart).

## Extending `config.yaml` with Custom `config.*.yaml` Files

You may add additional `config.*.yaml` files to organize additional commands as you see fit for your project and team.

For example, many teams commit their `config.yaml` and share it throughout the team, but some team members may require overrides to the checked-in version specifically for their environment and not checked in. For example, a team member may want to use a [`router_http_port`](../configuration/config.md#router_http_port) other than the team default due to a conflict in their development environment. In this case they could add `.ddev/config.ports.yaml`:

```yaml
# My machine can’t use port 80 so override with port 8080, but don’t check this in!
router_http_port: 8080
```

Extra `config.*.yaml` files are loaded in lexicographic order, so `config.a.yaml` will be overridden by `config.b.yaml`.

Team members may choose to use `config.local.yaml` for local non-committed config changes, for example. `config.local.yaml` is gitignored by default.

`config.*.yaml` update configuration according to these rules:

1. Simple fields like [`router_http_port`](../configuration/config.md#router_http_port) or [`webserver_type`](../configuration/config.md#webserver_type) are overwritten.
2. Lists of strings like [`additional_hostnames`](../configuration/config.md#additional_hostnames) or [`additional_fqdns`](../configuration/config.md#additional_fqdns) are merged.
3. The list of environment variables in [`web_environment`](../configuration/config.md#web_environment) are “smart merged”: if you add the same environment variable with a different value, the value in the override file will replace the value from `config.yaml`.
4. Hook specifications in the [`hooks`](../configuration/config.md#hooks) variable are merged.

If you need to _override_ existing values, set [`override_config: true`](../configuration/config.md#override_config) in the `config.*.yaml` where the override behavior should take place. Since `config.*.yaml` files are normally _merged_ into the configuration, some things can’t be overridden normally. For example, if you have [`nfs_mount_enabled: true`](../configuration/config.md#nfs_mount_enabled) you can’t override it with a merge and you can’t erase existing hooks or all environment variables. However, with `override_config: true` in a particular `config.*.yaml` file,

```yaml
override_config: true
nfs_mount_enabled: false
```

can override the existing values, and

```yaml
override_config: true
hooks:
  post-start: []
```

or

```yaml
override_config: true
additional_hostnames: []
```

can have their intended affect.

[`override_config`](../configuration/config.md#override_config) affects only behavior of the `config.*.yaml` file it exists in.

To experiment with the behavior of a set of `config.*.yaml` files, use the [`ddev debug configyaml`](../usage/commands.md#debug-configyaml) file; it’s especially valuable with the `yq` command, for example `ddev debug configyaml | yq`.

## Explicit `supervisord` Configuration for Additional Daemons

Although most extra daemons (like Node.js daemons, etc.) can be configured easily using [web_extra_daemons](#running-extra-daemons-in-the-web-container), there may be situations where you want complete control of the `supervisord` configuration.

In these case you can create a `.ddev/web-build/<daemonname>.conf` with configuration like:

```
[program:daemonname]
command=/var/www/html/path/to/daemon
directory=/var/www/html/
autorestart=true
startretries=10
stdout_logfile=/var/tmp/logpipe
stdout_logfile_maxbytes=0
redirect_stderr=true
```

And create a `.ddev/web-build/Dockerfile.<daemonname>` to install the config file:

```dockerfile
ADD daemonname.conf /etc/supervisor/conf.d
```

Full details for advanced configuration possibilities are in [Supervisor docs](http://supervisord.org/configuration.html#program-x-section-settings).
