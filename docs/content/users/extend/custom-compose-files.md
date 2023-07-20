# Defining Additional Services with Docker Compose

## Prerequisite

Much of DDEV’s customization ability and extensibility comes from leveraging features and functionality provided by [Docker](https://docs.docker.com/) and [Docker Compose](https://docs.docker.com/compose/overview/). Some working knowledge of these tools is required in order to customize or extend the environment DDEV provides.

There are [many examples of custom docker-compose files](https://github.com/ddev/ddev-contrib#additional-services-added-via-docker-composeserviceyaml) available on [ddev-contrib](https://github.com/ddev/ddev-contrib).

## Background

Under the hood, DDEV uses a private copy of docker-compose to define and run the multiple containers that make up the local environment for a project. docker-compose supports defining multiple compose files to facilitate sharing Compose configurations between files and projects, and DDEV is designed to leverage this ability.

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

[`ddev exec`](../usage/commands.md#exec), [`ddev ssh`](../usage/commands.md#ssh), and [`ddev logs`](../usage/commands.md#logs) interact with containers on an individual basis.

By default, these commands interact with the `web` container for a project. All of these commands, however, provide a `--service` or `-s` flag allowing you to specify the service name of the container to interact with. For example, if you added a service to provide Apache Solr, and the service was named `solr`, you would be able to run `ddev logs --service solr` to retrieve the Solr container’s logs.

## Third Party Services May Need To Trust `ddev-webserver`

Sometimes a third-party service (`docker-compose.*.yaml`) may need to consume content from the `ddev-webserver` container. A PDF generator like [Gotenberg](https://github.com/gotenberg/gotenberg), for example, might need to read in-container images or text in order to create a PDF. Or a testing service may need to read data in order to support tests.

A third-party service is not configured to trust DDEV’s `mkcert` certificate authority by default, so in cases like this you have to either use HTTP between the two containers, or make the third-party service ignore or trust the certificate authority.

Using plain HTTP between the containers is the simplest technique. For example, the [`ddev-selenium-standalone-chrome`](https://github.com/ddev/ddev-selenium-standalone-chrome) service must consume content, so it conducts interactions with the `ddev-webserver` [by accessing `http://web`](https://github.com/ddev/ddev-selenium-standalone-chrome/blob/main/config.selenium-standalone-chrome.yaml#L17). In this case, the `selenium-chrome` container accesses the `web` container via HTTP instead of HTTPS.

A second technique is to tell the third-party service to ignore HTTPS/TLS errors. For example, if the third-party service uses cURL, it could use `curl --insecure https://web` or `curl --insecure https://<project>.ddev.site`.

A third and more complex approach is to make the third-party container actually trust the self-signed certificate that the `ddev-webserver` container is using. This can be done in many cases using a custom Dockerfile and some extra configuration in the `ddev-config.*.yaml`. An example would be:

```yaml
services:
  example:
    container_name: ddev-${DDEV_SITENAME}-example
    command: "bash -c 'mkcert -install && original-start-command-from-image'"
    # Add a build stage so we can add `mkcert`, etc.
    # The Dockerfile for the build stage goes in the `.ddev/example directory` here
    build:
      context: example
    environment:
      - HTTP_EXPOSE=3001:3000
      - HTTPS_EXPOSE=3000:3000
      - VIRTUAL_HOST=$DDEV_HOSTNAME
    # Adding external_links allows connections to `https://example.ddev.site`,
    # which then can go through `ddev-router`
    external_links:
      - ddev-router:${DDEV_SITENAME}.${DDEV_TLD}
    labels:
      com.ddev.approot: $DDEV_APPROOT
      com.ddev.site-name: ${DDEV_SITENAME}
    restart: 'no'
    volumes:
      - .:/mnt/ddev_config
      # `ddev-global-cache` gets mounted so we have the CAROOT
      # This is required so that the CA is available for `mkcert` to install
      - ddev-global-cache:/mnt/ddev-global-cache
```

```Dockerfile
FROM example/example

# CAROOT for `mkcert` to use, has the CA config
ENV CAROOT=/mnt/ddev-global-cache/mkcert

# If the image build does not run as the default `root` user,
# temporarily change to root. If the image already has the default setup
# where it builds as `root`, then
# there is no need to change user here.
USER root
# Give the `example` user full `sudo` privileges
RUN echo "example ALL=(ALL) NOPASSWD: ALL" >> /etc/sudoers.d/example && chmod 0440 /etc/sudoers.d/example
# Install the correct architecture binary of `mkcert`
RUN export TARGETPLATFORM=linux/$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/') && mkdir -p /usr/local/bin && curl --fail -JL -s -o /usr/local/bin/mkcert "https://dl.filippo.io/mkcert/latest?for=${TARGETPLATFORM}"
RUN chmod +x /usr/local/bin/mkcert
USER original_user
```
