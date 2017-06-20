<h1>Extending and Customizing Environments</h1>
ddev provides several ways in which the environment for a project using ddev can be customized and extended.

## Adding services to a project
For most standard web applications, ddev provides everything you need to successfully provision and develop a web application on your local machine out of the box. More complex and sophisticated web applications, however, often require integration with services beyond the standard requirements of a web and database server. Examples of these additional services are Apache Solr, Redis, Varnish, etc. While ddev likely won't ever provide all of these additional services out of the box, it is designed to provide simple ways for the environment to be customized and extended to meet the needs of your application.

A collection of vetted service configurations is available in the [Additional Services Documentation](extend/additional-services.md).

If you need to create a service configuration for your project, see [Defining an additional service with Docker Compose](extend/custom-compose-files.md)

## Providing custom nginx configuration
The default web container for ddev uses NGINX as the web server. A default configuration is provided in the web container that should work for most Drupal 7+ and WordPress sites. Some sites may require custom configuration, for example to support a module or plugin requiring special rules. To accommodate these needs, ddev provides a way to replace the default configuration with a custom version.

- Run `ddev config` for the site if it has not been used with ddev before.
- Create a file named "nginx-site.conf" in the ".ddev" directory for your site.
- Add your configurations to the "nginx-site.conf" file. You can optionally use the [default NGINX configuration provided by ddev](https://github.com/drud/docker.nginx-php-fpm-local/blob/master/files/etc/nginx/nginx-site.conf) as a reference or starting point. [Additional configuration examples](https://www.nginx.com/resources/wiki/start/#other-examples) and documentation are available at the [nginx wiki](https://www.nginx.com/resources/wiki/)
- **NOTE:** The "root" statement in the server block must be `root $NGINX_DOCROOT;` in order to ensure the path for NGINX to serve the site from is correct.
- Save your configuration file and run `dev start` to start the site environment. If you encounter issues with your configuration or the site fails to start, use `ddev logs` to inspect the logs for possible NGINX configuration errors.

## Overriding default container images
The default container images provided by ddev are defined in the `config.yaml` file in the `.ddev` folder of your project. This means that _defining_ an alternative image for default services is as simple as changing the image definition in `config.yaml`. In practice, however, ddev currently has certain expectations and assumptions for what the web and database containers provide. At this time, it is recommended that the default container projects be referenced or used as a starting point for developing an alternative image. If you encounter difficulties integrating alternative images, please [file an issue and let us know](https://github.com/drud/ddev/issues/new).
