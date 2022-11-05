# Defining Additional Services with Docker Compose

## Prerequisite

Much of DDEV’s customization ability and extensibility comes from leveraging features and functionality provided by [Docker](https://docs.docker.com/) and [Docker Compose](https://docs.docker.com/compose/overview/). Some working knowledge of these tools is required in order to customize or extend the environment DDEV provides.

There are [many examples of custom docker-compose files](https://github.com/drud/ddev-contrib#additional-services-added-via-docker-composeserviceyaml) available on [ddev-contrib](https://github.com/drud/ddev-contrib).

## Background

Under the hood, DDEV uses a private copy of docker-compose to define and run the multiple containers that make up the local environment for a project. docker-compose supports defining multiple compose files to facilitate [sharing Compose configurations between files and projects](https://docs.docker.com/compose/extends/), and DDEV is designed to leverage this ability.

To add custom configuration or additional services to your project, create docker-compose files in the `.ddev` directory. DDEV will process any files with the `docker-compose.[servicename].yaml` naming convention and include them in executing docker-compose functionality. You can optionally create a `docker-compose.override.yaml` to override any configurations from the main `.ddev/.ddev-docker-compose-base.yaml` or any additional docker-compose files added to your project.

!!!warning "Don’t modify `.ddev-docker-compose-base.yaml` or `.ddev-docker-compose-full.yaml`!"

    The main docker-compose file is `.ddev/.ddev-docker-compose-base.yaml`, reserved exclusively for DDEV’s use. It’s overwritten every time a project is started, so any edits will be lost. If you need to override configuration provided by `.ddev/.ddev-docker-compose-base.yaml`, use an additional `docker-compose.<whatever>.yaml` file instead.

## `docker-compose.*.yaml` Examples

* Expose an additional port 9999 to host port 9999, in a file perhaps called `docker-compose.ports.yaml`:

```yaml
services:
  someservice:
    ports:
    - "9999:9999"
```

That approach usually isn’t sustainable because two projects might want to use the same port, so we *expose* the additional port to the Docker network and then use `ddev-router` to bind it to the host. This works only for services with an HTTP API, but results in having both HTTP and HTTPS ports (9998 and 9999).

```yaml
services:
    someservice:
        container_name: "ddev-${DDEV_SITENAME}-someservice"
        labels:
        com.ddev.site-name: ${DDEV_SITENAME}
        com.ddev.approot: ${DDEV_APPROOT}
        expose: 
        - "9999"
        environment:
        - VIRTUAL_HOST=$DDEV_HOSTNAME
        - HTTP_EXPOSE=9998:9999
        - HTTPS_EXPOSE=9999:9999
```

## Confirming docker-compose Configurations

To better understand how DDEV parses your custom docker-compose files, run `ddev debug compose-config`. This prints the final, DDEV-generated docker-compose configuration when starting your project.

## Conventions for Defining Additional Services

When defining additional services for your project, we recommended following these conventions to ensure DDEV handles your service the same way DDEV handles default services.

* The container name should be `ddev-${DDEV_SITENAME}-<servicename>`.
* Provide containers with required labels:

    ```yaml
        labels:
          com.ddev.site-name: ${DDEV_SITENAME}
          com.ddev.approot: ${DDEV_APPROOT}
    ```

* Exposing ports for service: you can expose the port for a service to be accessible as `projectname.ddev.site:portNum` while your project is running. This is achieved by the following configurations for the container(s) being added:

    * Define only the internal port in the `expose` section for docker-compose; use `ports:` only if the port will be bound directly to `localhost`, as may be required for non-HTTP services.

    * To expose a web interface to be accessible over HTTP, define the following environment variables in the `environment` section for docker-compose:

        * `VIRTUAL_HOST=$DDEV_HOSTNAME`
        * `HTTP_EXPOSE=portNum` The `hostPort:containerPort` convention may be used here to expose a container’s port to a different external port. To expose multiple ports for a single container, define the ports as comma-separated values.
        * `HTTPS_EXPOSE=<exposedPortNumber>:portNum` This will expose an HTTPS interface on `<exposedPortNumber>` to the host (and to the `web` container) as `https://<project>.ddev.site:exposedPortNumber`. To expose multiple ports for a single container, use comma-separated definitions, as in `HTTPS_EXPOSE=9998:80,9999:81`, which would expose HTTP port 80 from the container as `https://<project>.ddev.site:9998` and HTTP port 81 from the container as `https://<project>.ddev.site:9999`.

## Interacting with Additional Services

[`ddev exec`](../basics/commands.md#exec), [`ddev ssh`](../basics/commands.md#ssh), and [`ddev logs`](../basics/commands.md#logs) interact with containers on an individual basis.

By default, these commands interact with the `web` container for a project. All of these commands, however, provide a `--service` or `-s` flag allowing you to specify the service name of the container to interact with. For example, if you added a service to provide Apache Solr, and the service was named `solr`, you would be able to run `ddev logs --service solr` to retrieve the Solr container’s logs.
