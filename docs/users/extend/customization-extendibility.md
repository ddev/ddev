<h1>Extending and Customizing Environments</h1>
ddev provides several ways in which the environment for a project using ddev can be customized and extended.

## Changing PHP version

The project's `.ddev/config.yaml` file defines the PHP version to use. This can be changed, and the php_version can be set there to (currently) "5.6", "7.0", "7.1", "7.2", or "7.3". The current default is php 7.1.

### Older versions of PHP

[Support for older versions of PHP is available on ddev-contrib](https://github.com/drud/ddev-contrib/blob/master/docker-compose-services/old_php) via [custom Docker compose files](custom-compose-files.md).

## Changing webserver type

DDEV-Local supports nginx with php-fpm by default ("nginx-fpm"), apache2 with php-fpm ("apache-fpm"), and apache2 with embedded php via cgi (apache-cgi). These can be changed using the "webserver_type" value in .ddev/config.yaml, for example `webserver_type: apache-fpm`. 

## Adding services to a project

For most standard web applications, ddev provides everything you need to successfully provision and develop a web application on your local machine out of the box. More complex and sophisticated web applications, however, often require integration with services beyond the standard requirements of a web and database server. Examples of these additional services are Apache Solr, Redis, Varnish, etc. While ddev likely won't ever provide all of these additional services out of the box, it is designed to provide simple ways for the environment to be customized and extended to meet the needs of your project.

A collection of vetted service configurations is available in the [Additional Services Documentation](additional-services.md).

If you need to create a service configuration for your project, see [Defining an additional service with Docker Compose](custom-compose-files.md)

## Providing custom environment variables to a container

Each project can have an unlimited number of .ddev/docker-compose.*.yaml files as described in [Custom Compose Files](./custom-compose-files.md), so it's easy to maintain custom environment variables in a .ddev/docker-compose.environment.yaml file (the exact name doesn't matter, if it just matches docker-compose.*.yaml).

For example, a `.ddev/docker-compose.environment.yaml` with these contents would add a $TYPO3_CONTEXT environment variable to the web container, and a $SOMETHING environment variable to the db container: 

```
version: '3.6'

services:
  web:
    environment:
      - TYPO3_CONTEXT=Development
  db:
    environment:
      - SOMETHING=something special
```

## Providing custom nginx configuration
The default configuration for ddev uses nginx as the web server (`webserver_type: nginx-fpm` in .ddev/config.yaml). Default configurations are provided for most project types. Some projects may require custom configuration, for example to support a module or plugin requiring special rules. To accommodate these needs, ddev provides a way to replace the default configuration with a custom version.

- Run `ddev config` for the project if it has not been used with ddev before.
- To add an nginx snippet to the default config (preferred), add an nginx config file as `.ddev/nginx/<something>.conf`
- To replace the entire nginx-site.conf (legacy), create a file named "nginx-site.conf" in `.ddev/nginx-site.conf` for your project. In the ddev web container, these are separate per project type, and you can see and copy the default configurations in the [web container code](https://github.com/drud/ddev/tree/master/containers/ddev-webserver/files/etc/nginx). You can also use `ddev ssh` to review existing configurations in the container at /etc/nginx. [Additional configuration examples](https://www.nginx.com/resources/wiki/start/#other-examples) and documentation are available at the [nginx wiki](https://www.nginx.com/resources/wiki/) Note that the "root" statement in the server block must be `root $WEBSERVER_DOCROOT;` in order to ensure the path for NGINX to serve the project from is correct.
- Save your configuration file and run `ddev restart` to restart the project with the new configuration. If you encounter issues with your configuration or the project fails to start, use `ddev logs` to inspect the logs for possible NGINX configuration errors.
- The `location ~ ^/(phpstatus|ping)$ {` block is required for the webserver container healthcheck to work.
`
- Any errors in your configuration may cause the web container to fail and try to restart, so if you see that behavior, use `ddev logs` to diagnose.
- **IMPORTANT**: Changes to configuration take place on a `ddev restart` or when the container is rebuilt for another reason.

## Providing custom apache configuration

If you're using `webserver_type: apache-fpm` or `webserver_type: apache-cgi` in your .ddev/config.yaml you can override the default site configuration by adding a `.ddev/apache/apache-site.conf` configuration. A default configuration is provided in the web container that should work for most projects. Some projects may require custom configuration, for example to support a module or plugin requiring special rules. To accommodate these needs, ddev provides a way to replace the default configuration with a custom version.

- Run `ddev config` for the project if it has not been used with ddev before.
- Create a file named "apache-site.conf" in the ".ddev/apache" directory for your project. In the ddev web container, there may be various configuratons per project type, and you can see and copy the default configurations in the [web container code](https://github.com/drud/ddev/tree/master/containers/ddev-webserver/files/etc/apache2). You can also use `ddev ssh` to review existing configurations in the container at /etc/apache2.
- Add your configuration changes to the "apache-site.conf" file. You can optionally use the [ddev apache configs](https://github.com/drud/ddev/tree/master/containers/ddev-webserver/files/etc/apache2) as a starting point. 
- **NOTE:** The `DocumentRoot $WEBSERVER_DOCROOT` and related `        <Directory "$WEBSERVER_DOCROOT/">` statements must be provided as variables in order to ensure the docroot is updated correctly.
- Save your configuration file and run `ddev restart` to start the project. If you encounter issues with your configuration or the project fails to start, use `ddev logs` to inspect the logs for possible apache configuration errors.
- The alias `Alias "/phpstatus" "/var/www/phpstatus.php"` is required for the healthcheck script to work.
`
- Any errors in your configuration may cause the web container to fail and try to restart, so if you see that behavior, use `ddev logs` to diagnose.
- **IMPORTANT**: Changes to .ddev/apache/apache-site.conf take place on a `ddev restart` (or when the container is rebuilt for another reason).

## Providing custom PHP configuration (php.ini)

You can provide additional PHP configuration for a project by creating a directory called `.ddev/php/` and adding any number of php configuration ini files (they must be *.ini files). Normally, you should just override the specific option that you need to override. Note that any file that exists in `.ddev/php/` will be copied into `/etc/php/[version]/(cli|fpm)/conf.d`, so it's possible to replace files that already exist in the container. Common usage is to put custom overrides in a file called `my-php.ini`. Make sure you include the section header that goes with each item (like `[PHP]`)

One interesting implication of this behavior is that it's possible to disable extensions by replacing the configuration file that loads them. For instance, if you were to create an empty file at `.ddev/php/20-xdebug.ini`, it would replace the configuration that loads xdebug, which would cause xdebug to not be loaded!

To load the new configuration, just run a `ddev restart`.

An example file in .ddev/php/my-php.ini might look like this:
```
[PHP]
max_execution_time = 240;
```

## Providing custom mysql/MariaDB configuration (my.cnf)

You can provide additional MySQL configuration for a project by creating a directory called `.ddev/mysql/` and adding any number of MySQL configuration files (these must have the suffix ".cnf"). These files will be automatically included when MySQL is started. Make sure that the section header is included in the file 

An example file in .ddev/mysql/no_utf8mb4.cnf might be:

```
[mysqld]
collation-server = utf8_general_ci
character-set-server = utf8
innodb_large_prefix=false
```

To load the new configuration, run `ddev restart`.

## Extending config.yaml with custom config.*.yaml files

You may add additional config.*.yaml files to organize additional commands as you see fit for your project and team. 

For example, many teams commit their config.yaml and share it throughout the team, but some team members may require overrides to the checked-in version that are custom to their environment and should not be checked in. For example, a team member may want to use a router_http_port other than the team default due to a conflict in their development environment. In this case they could add the file .ddev/config.ports.yaml with the contents:

```
# My machine can't use port 80 so override with port 8080, but don't check this in.
router_http_port: 8080
```

config.*.yaml is by default omitted from git by the .ddev/.gitignore file.

Extra config.*.yaml files are loaded in lexicographic order, so "config.a.yaml" will be overridden by "config.b.yaml". 

Teams may choose to use "config.local.yaml" or "config.override.yaml" for all local non-committed config changes, for example.
