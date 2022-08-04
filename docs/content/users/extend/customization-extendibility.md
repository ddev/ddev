# Extending and Customizing Environments

ddev provides several ways in which the environment for a project using ddev can be customized and extended.

## Changing PHP version

The project's `.ddev/config.yaml` file defines the PHP version to use. This can be changed, and the php_version can be set there to `5.6`, `7.0`, `7.1`, `7.2`,  `7.3`, `7.4`, `8.0` or `8.1`. The current default is php 7.4.

### Older versions of PHP

[Support for older versions of PHP is available on ddev-contrib](https://github.com/drud/ddev-contrib/blob/master/docker-compose-services/old_php) via [custom Docker compose files](custom-compose-files.md).

## Changing webserver type

DDEV-Local supports nginx with php-fpm by default ("nginx-fpm") and also apache2 with php-fpm ("apache-fpm"). These can be changed using the "webserver_type" value in .ddev/config.yaml, for example `webserver_type: apache-fpm`.

## Adding services to a project

For most standard web applications, ddev provides everything you need to successfully provision and develop a web application on your local machine out of the box. More complex and sophisticated web applications, however, often require integration with services beyond the standard requirements of a web and database server. Examples of these additional services are Apache Solr, Redis, Varnish, etc. While ddev likely won't ever provide all of these additional services out of the box, it is designed to provide simple ways for the environment to be customized and extended to meet the needs of your project.

A collection of vetted service configurations is available in the [Additional Services Documentation](additional-services.md).

If you need to create a service configuration for your project, see [Defining an additional service with Docker Compose](custom-compose-files.md)

## Using nodejs with DDEV

There are many, many ways to deploy nodejs in any project, so DDEV tries to let you set up any possibility you can come up with.

* You can choose the nodejs version you want to use in `.ddev/config.yaml` with `nodejs_version`
* `ddev nvm` gives you the full capabilities of nvm.
* `ddev npm` and `ddev yarn` provide shortcuts to the npm and yarn commands inside the container, and their caches are persistent.
* You can run nodejs daemons using [Running extra daemons](#running-extra-daemons-in-the-web-container).
* You can expose nodejs ports via ddev-router by using [Exposing extra ports](#exposing-extra-ports-via-ddev-router).
* You can just manually run particular nodejs scripts as needed using `ddev exec <script>` or `ddev exec nodejs <script>`.

Please share your techniques using the many ways to share. Best are [ddev-get add-ons](additional-services.md#additional-service-configurations-and-add-ons-for-ddev), but [Stack Overflow](https://stackoverflow.com/tags/ddev) and [ddev-contrib](https://github.com/drud/ddev-contrib) are others.

## Running extra daemons in the web container

You can run processes inside the web container a number of ways.

1. You can manually execute them when you need them, with `ddev exec`, for example.
2. You can run them with a post-start hook.
3. You can run them automatically using `web_extra_daemons`.

### Running extra daemons with post-start hook

Needed daemons can be run either with a `post-start` `exec` hook or they can be automatically started using supervisord.

A simple `post-start` exec hook in `.ddev/config.yaml` might look like:

```yaml
hooks:
  post-start:
    - exec: "nohup php --docroot=/var/www/html/something -S 0.0.0.0:6666 &"
```

### Running extra daemons using `web_extra_daemons`

If you need extra daemons to start up automatically inside the web container, you can easily add them using `web_extra_daemons` in `.ddev/config.yaml`.

You might be running node daemons that serve a particular purpose (like `browsersync`) or daemons like a `cron` daemon, etc.

For example, you could use this configuration to run two instances of the nodejs http-server serving different directories:

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
* `command` is best as a simple binary with its arguments, but `bash` features like `cd` or `&&` do work. If the program to be run is not in the `ddev-webserver` `$PATH` then it should have the absolute in-container path to the program to be run, like `/var/www/html/node_modules/.bin/http-server`.
* `web_extra_daemons` is a shortcut for adding a configuration to `supervisord`, which organizes daemons inside the web container. If the default settings are inadequate for your use, you can write a [complete config file for your daemon](#explicit-supervisord-configuration-for-additional-daemons).
* Your daemon is expected to run in the foreground, not to daemonize itself, `supervisord` will take care of that.
* To see the results of the attempt to start your daemon, see `ddev logs` or `docker logs ddev-<project>-web`.

## Exposing extra ports via ddev-router

If your web container has additional HTTP servers running inside it on different ports, those can be exposed using `web_extra_exposed_ports` in `.ddev/config.yaml`. For example, this configuration would expose a `node-vite` HTTP server running on port 3000 inside the web container via ddev-router to ports 9998 (http) and 9999 (https), so it could be accessed via `https://<project>.ddev.site:9999`:

```yaml
web_extra_exposed_ports:
  - name: node-vite
    container_port: 3000
    http_port: 9998
    https_port: 9999
```

The configuration below would expose a nodejs server running in the web container on port 3000 as `https://<project>.ddev.site:4000` and a "something" server running in the web container on port 4000 as `https://<project>.ddev.site:4000`:

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

**Please make sure you fill in all three fields, even if you don't intend to use the `https_port`.** If you don't add `https_port`, then it will be a default `0` and the `ddev-router` will fail to start.

## Providing custom environment variables to a container

Custom environment variables may be set in the project config.yaml or the ~/.ddev/global_config.yaml with the `web_environment` key, for example

```yaml
web_environment:
- SOMEENV=someval
- SOMEOTHERENV=someotherval
```

You can also use `ddev config global --web-environment-add="SOMEENV=someval"` or `ddev config --web-environment-add="SOMEENV=someval"` for the same purpose. The command just sets the values in the configuration files. (The `--web-environment` option is also available, but it overwrites existing contents. Alternately, edit `.ddev/config.yaml` or `~/.ddev/global_config.yaml`.)

The docker-compose web environment can also provide env file support. To enable this you would create a new docker-compose yaml partial, for example `.ddev/docker-compose.env-file.yaml` with contents:

```
services:
  web:
    env_file:
      - ../.env

```

You would create the `.env` file in the project root and provide globals within it as such:

```
SOMEENV='someval'
SOMEOTHERENV='someotherval'
```

The globals from the env file would be available on the next ddev start. It is important to note that typically the .env file should *not* be placed under source control, especially if it contains private API keys or passwords.

### Altering the in-container $PATH

Sometimes it's easiest to just put the command you need into the existing `$PATH` using a symbolic link rather than changing the in-container PATH. For example, the project `bin` directory is already in the PATH. So if you have a command you want to run that is not already in the $PATH, you can just add a symlink. A couple of examples:

* On Craft CMS, the `craft` script is often in the project root, which is not in the PATH. But if you `mkdir bin && ln -s craft bin/craft` you should be able to use `ddev exec craft` just fine. (Note however that `ddev craft` takes care of this for you.)
* On projects where the `vendor` directory is not in the project root (Acquia projects, for example, have `composer.json` and `vendor` in the `docroot` directory), you can `mkdir bin && ln -s docroot/vendor/bin/drush bin/drush` to put `drush` in your PATH. (With projects like this make sure to set `composer_root: docroot` so that `ddev composer` works properly.)

Because many things touch the `$PATH` environment variable, it's slightly harder to change it, but it's easy: Add a script to `<project>/.ddev/homeadditions/.bashrc.d/` or (global) `~/.ddev/homeadditions/.bashrc.d/` that adds to the `$PATH` variable. For example, if your project vendor directory is not in the expected place (`/var/www/html/vendor/bin`) you can add a `<project>/.ddev/homeadditions/.bashrc.d/path.sh` with contents:

```bash
export PATH=$PATH:/var/www/html/somewhereelse/vendor/bin
```

## Custom nginx configuration

When you `ddev start` using the `nginx-fpm` webserver_type, ddev creates a configuration customized to your project type in `.ddev/nginx_full/nginx-site.conf`. You can edit and override the configuration by removing the `#ddev-generated` line and doing whatever you need with it. After each change, `ddev start`.

You can also have more than one config file in the `.ddev/nginx_full` directory, they will all get loaded when ddev starts. This can be used for serving multiple docroots (advanced, below), or for any other technique.

### Troubleshooting nginx configuration

* Any errors in your configuration may cause the web container to fail and try to restart, so if you see that behavior, use `ddev logs` to diagnose it.
* You can `ddev exec nginx -t` to test whether your configuration is valid. (Or `ddev ssh` and run `nginx -t`)
* You can reload the nginx configuration either with `ddev start` or `ddev exec nginx -s reload`
* The alias `Alias "/phpstatus" "/var/www/phpstatus.php"` is required for the healthcheck script to work.
* **IMPORTANT**: Changes to configuration take place on a `ddev start`, when the container is rebuilt for another reason, or when the nginx server receives the reload signal.

### Multiple docroots in nginx (advanced)

It's easiest to have different webservers in different ddev projects and different ddev projects can [easily communicate with each other](../basics/faq.md), but some sites require more than one docroot for a single project codebase. Sometimes this is because there's an API built in the same codebase but using different code, or different code for different languages, etc.

The generated `.ddev/nginx_full/seconddocroot.conf.example` demonstrates how to do this. You can create as many of these as you want, change the `servername` and the `root` and customize as you see fit.

### Nginx snippets

To add an nginx snippet to the default config, add an nginx config file as `.ddev/nginx/<something>.conf`. This feature will be disabled in the future.

## Custom apache configuration

If you're using `webserver_type: apache-fpm` in your .ddev/config.yaml, you can override the default site configuration by editing or replacing the ddev-provided `.ddev/apache/apache-site.conf` configuration.

* Edit the `.ddev/apache/apache-site.conf`.
* Add your configuration changes.
* Save your configuration file and run `ddev start` to reload the project. If you encounter issues with your configuration or the project fails to start, use `ddev logs` to inspect the logs for possible apache configuration errors.
* Use `ddev exec apachectl -t` to do a general apache syntax check.
* The alias `Alias "/phpstatus" "/var/www/phpstatus.php"` is required for the healthcheck script to work.
* Any errors in your configuration may cause the web container to fail, so if you see that behavior, use `ddev logs` to diagnose.
* **IMPORTANT**: Changes to .ddev/apache/apache-site.conf take place on a `ddev start`. You can also `ddev exec apachectl -k graceful` to reload the apache configuration.

## Providing custom PHP configuration (php.ini)

You can provide additional PHP configuration for a project by creating a directory called `.ddev/php/` and adding any number of php configuration ini files (they must be \*.ini files). Normally, you should just override the specific option that you need to override. Note that any file that exists in `.ddev/php/` will be copied into `/etc/php/[version]/(cli|fpm)/conf.d`, so it's possible to replace files that already exist in the container. Common usage is to put custom overrides in a file called `my-php.ini`. Make sure you include the section header that goes with each item (like `[PHP]`)

One interesting implication of this behavior is that it's possible to disable extensions by replacing the configuration file that loads them. For instance, if you were to create an empty file at `.ddev/php/20-xdebug.ini`, it would replace the configuration that loads xdebug, which would cause xdebug to not be loaded!

To load the new configuration, just run a `ddev restart`.

An example file in .ddev/php/my-php.ini might look like this:

```ini
[PHP]
max_execution_time = 240;
```

## Custom mysql/MariaDB configuration (`my.cnf`)

You can provide additional MySQL configuration for a project by creating a directory called `.ddev/mysql/` and adding any number of MySQL configuration files (these must have the suffix `.cnf`). These files will be automatically included when MySQL is started. Make sure that the section header is included in the file

An example file in `.ddev/mysql/no_utf8mb4.cnf` might be:

```
[mysqld]
collation-server = utf8_general_ci
character-set-server = utf8
innodb_large_prefix=false
```

To load the new configuration, run `ddev restart`.

## Custom Postgresql configuration

If you are using Postgresql, a default `posgresql.conf` is provided in `.ddev/postgres/postgresql.conf`. If you need to alter it, remove the `#ddev-generated` line and `ddev restart`.

## Extending config.yaml with custom `config.\*.yaml` files

You may add additional config.\*.yaml files to organize additional commands as you see fit for your project and team.

For example, many teams commit their config.yaml and share it throughout the team, but some team members may require overrides to the checked-in version that are custom to their environment and should not be checked in. For example, a team member may want to use a router_http_port other than the team default due to a conflict in their development environment. In this case they could add the file .ddev/config.ports.yaml with the contents:

```yaml
# My machine can't use port 80 so override with port 8080, but don't check this in.
router_http_port: 8080
```

config.\*.yaml is by default omitted from git by the .ddev/.gitignore file. You can commit it by using `git add -f .ddev/config.<something>.yaml`.

Extra config.\*.yaml files are loaded in lexicographic order, so "config.a.yaml" will be overridden by "config.b.yaml".

Teams may choose to use "config.local.yaml" or "config.override.yaml" for all local non-committed config changes, for example.

config.\*.yaml update configuration according to
these rules:

1. Simple fields like `router_http_port` or `webserver_type` are overwritten.
2. Lists of strings like `additional_hostnames` or `additional_fqdns` are merged.
3. The list of environment variables in `web_environment` are "smart merged": if you add the same environment variable with a different value, the value in the override file will replace the value from config.yaml.
4. Hook specifications in the `hooks` variable are also merged.

## Explicit supervisord configuration for additional daemons

Although most extra daemons (like nodejs daemons, etc) can be configured easily using [web_extra_daemons](#running-extra-daemons-in-the-web-container), there may be situations where you want complete control of the `supervisord` configuration.

In these case you can create a `.ddev/web-build/<daemonname>.conf` with configuration like

```
[program:daemonname]
command=/var/www/html/path/to/daemon
directory=/var/www/html/
autorestart=true
startretries=10
stdout_logfile=/proc/self/fd/2
stdout_logfile_maxbytes=0
redirect_stderr=true
```

And create a `.ddev/web-build/Dockerfile.<daemonname>` to install the config file:

```dockerfile
ADD daemonname.conf /etc/supervisor/conf.d
```

Full details for advanced configuration possibilities are in [supervisor docs](http://supervisord.org/configuration.html#program-x-section-settings).
