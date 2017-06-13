<h1>Extending and Customizing Environments</h1>
For most standard web applications, ddev provides everything you need to successfully provision and develop a web application on your local machine out of the box. More complex and sophisticated web applications, however, often require integration with services beyond the standard requirements of a web and database server. Examples of these additional services are Apache Solr, Redis, Varnish, etc. While ddev likely won't ever provide all of these additional services out of the box, it is designed to provide simple ways for the environment to be customized and extended to meet the needs of your application.

## Prerequisite knowledge
The majority of ddev's customization ability and extensibility comes from leveraging features and functionality provided by [Docker](https://docs.docker.com/) and [Docker Compose](https://docs.docker.com/compose/overview/). Some working knowledge of these tools is required in order to customize or extend the environment ddev provides.

## Extending with additional docker-compose files
Under the hood, ddev uses docker-compose to define and run the multiple containers that make up the local environment for a web application. Docker compose supports defining multiple compose files to facilitate [sharing Compose configurations between files and projects](https://docs.docker.com/compose/extends/), and ddev is designed to leverage this ability. Additional services can be added to your project by creating additional docker-compose files in the `.ddev` directory for your project. Any files using the `docker-compose.[servicename].yml` naming convention will be automatically processed by ddev to include them in executing docker-compose functionality. A `docker-compose.override.yml` can additionally be created to override any configurations from the main docker-compose file or any service compose files added to your project.

## Conventions for defining additional services
When defining additional services for your project, it is recommended to follow these conventions to ensure your service is handled by ddev the same way the default services are.

- Containers should be follow the naming convention `local-[sitename]-[servicename]`

- Containers should be provided the following labels:

  - `com.ddev.site-name: ${DDEV_SITENAME}`

  - `com.ddev.approot: $DDEV_APPROOT`

  - `com.ddev.app-url: $DDEV_URL`

  - `com.ddev.container-type: [servicename]`

- Exposing ports for service: you can expose the port for a service to be accessible as `sitename.ddev.local:portNum` while your project is running. This is achieved by the following configurations for the container(s) being added:

  - Define only the internal port in the `ports` section for docker-compose. The `hostPort:containerPort` convention normally used to expose ports in docker should not be used here, since we are leveraging the ddev router to expose the ports.

  - To expose a web interface to be accessible over HTTP, define the following environment variables in the `environment` section for docker-compose:

    - `VIRTUAL_HOST=$DDEV_HOSTNAME`

    - `HTTP_EXPOSE=portNum` The `hostPort:containerPort` convention may be used here to expose a container's port to a different external port. To expose multiple ports for a single container, define the ports as comma-separated values.

## Interacting with additional services
Certain ddev commands, namely `ddev exec`, `ddev ssh`, and `ddev logs` interact with containers on an individual basis. By default, these commands interact with the web container for a project. All of these commands, however, provide a `--service` or `-s` flag allowing you to specify the service name of the container you want to interact with. For example, if you added a service to provide Apache Solr, and the service was named `solr`, you would be able to run `ddev logs --service solr` to retrieve the logs of the Solr container.

### Defining an additional service - Apache Solr example
The following is an example of an additional service that could be added to a project. This is a working example at time of writing. Annotations are added to highlight configurations important for running the service as part of a project in ddev. For full documentation on usage, see the [Docker Compose documentation](https://docs.docker.com/compose/overview/).

This file would be saved as `docker-compose.solr.yml` in your project's `.ddev` folder.

```
# ddev apache solr recipe file
#
# To use this in your own project: Copy this file to your project's .ddev folder,
# and create the folder path .ddev/solr/conf. Then, copy the solr configuration
# files for your project to .ddev/solr/conf. E.g., using Drupal Search API Solr, 
# you would copy the solr-conf/5.x/ contents into .ddev/solr/conf. The configuration
# files must be present before running `ddev start`.

version: '3'

services:
  solr: # This is the service name used when running ddev commands accepting the --service flag
    container_name: ddev-${DDEV_SITENAME}-solr # This is the name of the container. It is recommended to follow the same name convention used in the main docker-compose.yml file.
    image: solr:5.4
    restart: always
    ports:
      - 8983 # Solr is served from this port inside the container
    labels:
    # These labels ensure this service is discoverable by ddev
      com.ddev.site-name: ${DDEV_SITENAME}
      com.ddev.approot: $DDEV_APPROOT
      com.ddev.app-url: $DDEV_URL
    environment:
      - VIRTUAL_HOST=$DDEV_HOSTNAME # This defines the host name the service should be accessible from. This will be sitename.ddev.local
      - HTTP_EXPOSE=8983 # This defines the port the service should be accessible from at sitename.ddev.local
    volumes:
      - "./solr:/solr-conf" # This exposes a mount to the host system `.ddev/solr-conf` directory.
    entrypoint:
      - docker-entrypoint.sh
      - solr-precreate
      - dev
      - /solr-conf
# This links the solr service to the web service defined in the main docker-compose.yml, allowing applications running in the web service to access the solr service at sitename.ddev.local:8983
  web:
    links:
      - solr:$DDEV_HOSTNAME
```

## Providing custom nginx configuration
The default web container for ddev uses NGINX as the web server. A default configuration is provided in the web container that should work for most Drupal 7+ and WordPress sites. Some sites may require custom configuration, for example to support a module or plugin requiring special rules. To accommodate these needs, ddev provides a way to replace the default configuration with a custom version.

- Run `ddev config` for the site if it has not been used with ddev before.
- Create a file named "nginx-site.conf" in the ".ddev" directory for your site.
- Add your configurations to the "nginx-site.conf" file. You can optionally use the [default NGINX configuration provided by ddev](https://github.com/drud/docker.nginx-php-fpm-local/blob/master/files/etc/nginx/nginx-site.conf) as a reference or starting point. [Additional configuration examples](https://www.nginx.com/resources/wiki/start/#other-examples) and documentation are available at the [nginx wiki](https://www.nginx.com/resources/wiki/)
- **NOTE:** The "root" statement in the server block must be `root $NGINX_DOCROOT;` in order to ensure the path for NGINX to serve the site from is correct.
- Save your configuration file and run `dev start` to start the site environment. If you encounter issues with your configuration or the site fails to start, use `ddev logs` to inspect the logs for possible NGINX configuration errors.

## Overriding default container images
The default container images provided by ddev are defined in the `config.yaml` file in the `.ddev` folder of your project. This means that _defining_ an alternative image for default services is as simple as changing the image definition in `config.yaml`. In practice, however, ddev currently has certain expectations and assumptions for what the web and database containers provide. At this time, it is recommended that the default container projects be referenced or used as a starting point for developing an alternative image. If you encounter difficulties integrating alternative images, please [file an issue and let us know](https://github.com/drud/ddev/issues/new).
