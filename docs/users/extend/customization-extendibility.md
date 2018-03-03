<h1>Extending and Customizing Environments</h1>
ddev provides several ways in which the environment for a project using ddev can be customized and extended.

## Changing PHP version

The project's `.ddev/config.yaml` file defines the PHP version to use. This can be changed, and the php_version can be set there to (currently) "5.6", "7.0", or "7.1". The current default is php 7.1.

## Adding services to a project

For most standard web applications, ddev provides everything you need to successfully provision and develop a web application on your local machine out of the box. More complex and sophisticated web applications, however, often require integration with services beyond the standard requirements of a web and database server. Examples of these additional services are Apache Solr, Redis, Varnish, etc. While ddev likely won't ever provide all of these additional services out of the box, it is designed to provide simple ways for the environment to be customized and extended to meet the needs of your application.

A collection of vetted service configurations is available in the [Additional Services Documentation](additional-services.md).

If you need to create a service configuration for your project, see [Defining an additional service with Docker Compose](custom-compose-files.md)

## Providing custom nginx configuration
The default web container for ddev uses NGINX as the web server. A default configuration is provided in the web container that should work for most Drupal 7+ and WordPress projects. Some projects may require custom configuration, for example to support a module or plugin requiring special rules. To accommodate these needs, ddev provides a way to replace the default configuration with a custom version.

- Run `ddev config` for the project if it has not been used with ddev before.
- Create a file named "nginx-site.conf" in the ".ddev" directory for your project. In the ddev web container, these are separate per project type, and you can see and copy the default configurations in the [web container code](https://github.com/drud/docker.nginx-php-fpm-local/tree/master/files/etc/nginx). You can also use `ddev ssh` to review existing configurations in the container at /etc/nginx.
- Add your configuration changes to the "nginx-site.conf" file. You can optionally use the [ddev nginx configs](https://github.com/drud/docker.nginx-php-fpm-local/tree/master/files/etc/nginx) as a starting point. [Additional configuration examples](https://www.nginx.com/resources/wiki/start/#other-examples) and documentation are available at the [nginx wiki](https://www.nginx.com/resources/wiki/)
- **NOTE:** The "root" statement in the server block must be `root $NGINX_DOCROOT;` in order to ensure the path for NGINX to serve the project from is correct.
- Save your configuration file and run `dev start` to start the project environment. If you encounter issues with your configuration or the project fails to start, use `ddev logs` to inspect the logs for possible NGINX configuration errors (or use `ddev ssh` and inspect /var/log/nginx/error.log.)
- Any errors in your configuration may cause the web container to fail and try to restart, so if you see that behavior, check your container.
- Changes to .ddev/nginx-site.conf take place only after you do a `ddev rm` followed by `ddev start`.

## Providing custom PHP configuration (php.ini)

You can provide an alternate PHP configuration for a project as .ddev/php.ini. After `ddev rm` and `ddev start` you should see the behavior of your PHP configuration.

For starter PHP configurations, you can use the php.ini files found under the [fpm configuration](https://github.com/drud/docker.nginx-php-fpm-local/tree/master/files/etc/php) in each version of php.

## Providing custom mysql/MariaDB configuration (my.cnf)

You can provide an alternate /etc/my.cnf file for MariaDB by placing it in .ddev/my.cnf. After `ddev rm` and `ddev start` you should see the database server behavior change.

For a starter /etc/my.cnf, see the [my.cnf used by default](https://github.com/drud/mariadb-local/blob/master/files/etc/my.cnf)

## Overriding default container images
The default container images provided by ddev are defined in the `config.yaml` file in the `.ddev` folder of your project. This means that _defining_ an alternative image for default services is as simple as changing the image definition in `config.yaml`. In practice, however, ddev currently has certain expectations and assumptions for what the web and database containers provide. At this time, it is recommended that the default container projects be referenced or used as a starting point for developing an alternative image. If you encounter difficulties integrating alternative images, please [file an issue and let us know](https://github.com/drud/ddev/issues/new).
