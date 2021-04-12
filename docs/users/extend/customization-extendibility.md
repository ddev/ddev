## Extending and Customizing Environments

ddev provides several ways in which the environment for a project using ddev can be customized and extended.

### Changing PHP version

The project's `.ddev/config.yaml` file defines the PHP version to use. This can be changed, and the php_version can be set there to `5.6`, `7.0`, `7.1`, `7.2`,  `7.3`, `7.4` or `8.0`. The current default is php 7.4.

#### Older versions of PHP

[Support for older versions of PHP is available on ddev-contrib](https://github.com/drud/ddev-contrib/blob/master/docker-compose-services/old_php) via [custom Docker compose files](custom-compose-files.md).

### Changing webserver type

DDEV-Local supports nginx with php-fpm by default ("nginx-fpm") and also apache2 with php-fpm ("apache-fpm"). These can be changed using the "webserver_type" value in .ddev/config.yaml, for example `webserver_type: apache-fpm`.

### Adding services to a project

For most standard web applications, ddev provides everything you need to successfully provision and develop a web application on your local machine out of the box. More complex and sophisticated web applications, however, often require integration with services beyond the standard requirements of a web and database server. Examples of these additional services are Apache Solr, Redis, Varnish, etc. While ddev likely won't ever provide all of these additional services out of the box, it is designed to provide simple ways for the environment to be customized and extended to meet the needs of your project.

A collection of vetted service configurations is available in the [Additional Services Documentation](additional-services.md).

If you need to create a service configuration for your project, see [Defining an additional service with Docker Compose](custom-compose-files.md)

## Providing custom environment variables to a container

Custom environment variables may be set in the project config.yaml or the ~/.ddev/global_config.yaml with the `web_environment` key, for example

```yaml
web_environment:
- SOMEENV=someval
- SOMEOTHERENV=someotherval
```

You can also use `ddev config global --web-environment="SOMEENV=someval"` or `ddev config --web-environment="SOMEENV=someval"` for the same purpose. The command just sets the values in the configuration files.

### Providing custom nginx configuration

When you `ddev start` using the `nginx-fpm` webserver_type, ddev creates a configuration customized to your project type in `.ddev/nginx_full/nginx-site.conf`. You can edit and override the configuration by removing the `#ddev-generated` line and doing whatever you need with it. After each change, `ddev start`.

You can also have more than one config file in the `.ddev/nginx_full` directory, they will all get loaded when ddev starts. This can be used for serving multiple docroots (advanced, below), or for any other technique.

#### Troubleshooting nginx configuration

* Any errors in your configuration may cause the web container to fail and try to restart, so if you see that behavior, use `ddev logs` to diagnose it.
* You can `ddev exec nginx -t` to test whether your configuration is valid. (Or `ddev ssh` and run `nginx -t`)
* You can reload the nginx configuration either with `ddev start` or `ddev exec nginx -s reload`
* The alias `Alias "/phpstatus" "/var/www/phpstatus.php"` is required for the healthcheck script to work.
* **IMPORTANT**: Changes to configuration take place on a `ddev start`, when the container is rebuilt for another reason, or when the nginx server receives the reload signal.

#### Multiple docroots in nginx (advanced)

It's easiest to have different webservers in different ddev projects and different ddev projects can [easily communicate with each other](../faq.md), but some sites require more than one docroot for a single project codebase. Sometimes this is because there's an API built in the same codebase but using different code, or different code for different languages, etc.

The generated `.ddev/nginx_full/seconddocroot.conf.example` demonstrates how to do this. You can create as many of these as you want, change the `servername` and the `root` and customize as you see fit.

#### Nginx snippets (deprecated)

To add an nginx snippet to the default config, add an nginx config file as `.ddev/nginx/<something>.conf`. This feature will be disabled in the future.

### Providing custom apache configuration

If you're using `webserver_type: apache-fpm` in your .ddev/config.yaml, you can override the default site configuration by editing or replacing the ddev-provided `.ddev/apache/apache-site.conf` configuration.

* Edit the `.ddev/apache/apache-site.conf`.
* Add your configuration changes.
* Save your configuration file and run `ddev start` to reload the project. If you encounter issues with your configuration or the project fails to start, use `ddev logs` to inspect the logs for possible apache configuration errors.
* Use `ddev exec apachectl -t` to do a general apache syntax check.
* The alias `Alias "/phpstatus" "/var/www/phpstatus.php"` is required for the healthcheck script to work.
* Any errors in your configuration may cause the web container to fail, so if you see that behavior, use `ddev logs` to diagnose.
* **IMPORTANT**: Changes to .ddev/apache/apache-site.conf take place on a `ddev start`. You can also `ddev exec apachectl -k graceful` to reload the apache configuration.

### Providing custom PHP configuration (php.ini)

You can provide additional PHP configuration for a project by creating a directory called `.ddev/php/` and adding any number of php configuration ini files (they must be \*.ini files). Normally, you should just override the specific option that you need to override. Note that any file that exists in `.ddev/php/` will be copied into `/etc/php/[version]/(cli|fpm)/conf.d`, so it's possible to replace files that already exist in the container. Common usage is to put custom overrides in a file called `my-php.ini`. Make sure you include the section header that goes with each item (like `[PHP]`)

One interesting implication of this behavior is that it's possible to disable extensions by replacing the configuration file that loads them. For instance, if you were to create an empty file at `.ddev/php/20-xdebug.ini`, it would replace the configuration that loads xdebug, which would cause xdebug to not be loaded!

To load the new configuration, just run a `ddev restart`.

An example file in .ddev/php/my-php.ini might look like this:

```ini
[PHP]
max_execution_time = 240;
```

### Providing custom mysql/MariaDB configuration (my.cnf)

You can provide additional MySQL configuration for a project by creating a directory called `.ddev/mysql/` and adding any number of MySQL configuration files (these must have the suffix ".cnf"). These files will be automatically included when MySQL is started. Make sure that the section header is included in the file

An example file in .ddev/mysql/no_utf8mb4.cnf might be:

```
[mysqld]
collation-server = utf8_general_ci
character-set-server = utf8
innodb_large_prefix=false
```

To load the new configuration, run `ddev restart`.

### Extending config.yaml with custom config.\*.yaml files

You may add additional config.\*.yaml files to organize additional commands as you see fit for your project and team.

For example, many teams commit their config.yaml and share it throughout the team, but some team members may require overrides to the checked-in version that are custom to their environment and should not be checked in. For example, a team member may want to use a router_http_port other than the team default due to a conflict in their development environment. In this case they could add the file .ddev/config.ports.yaml with the contents:

```yaml
# My machine can't use port 80 so override with port 8080, but don't check this in.
router_http_port: 8080
```

config.\*.yaml is by default omitted from git by the .ddev/.gitignore file.

Extra config.\*.yaml files are loaded in lexicographic order, so "config.a.yaml" will be overridden by "config.b.yaml".

Teams may choose to use "config.local.yaml" or "config.override.yaml" for all local non-committed config changes, for example.
