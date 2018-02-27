<h1>Defining an additional service with Docker Compose</h1>

## Prerequisite knowledge

The majority of ddev's customization ability and extensibility comes from leveraging features and functionality provided by [Docker](https://docs.docker.com/) and [Docker Compose](https://docs.docker.com/compose/overview/). Some working knowledge of these tools is required in order to customize or extend the environment ddev provides.

## Background

Under the hood, ddev uses docker-compose to define and run the multiple containers that make up the local environment for a web application. Docker Compose supports defining multiple compose files to facilitate [sharing Compose configurations between files and projects](https://docs.docker.com/compose/extends/), and ddev is designed to leverage this ability.

To add services to your project, create docker-compose files in the `.ddev` directory for your project. ddev will process any files using the `docker-compose.[servicename].yml` naming convention and include them in executing docker-compose functionality. In addition, create a `docker-compose.override.yml` to override any configurations from the main docker-compose file or any service compose files added to your project.

## Conventions for defining additional services

When defining additional services for your project, we recommended you follow these conventions to ensure ddev handles your service the same way ddev handles default services.

- To name containers follow this naming convention `ddev-[projectname]-[servicename]`

- Provide containers with the following labels

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
