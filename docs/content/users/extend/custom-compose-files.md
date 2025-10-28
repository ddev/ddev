# Defining Additional Services with Docker Compose

## Prerequisite

Much of DDEV’s customization ability and extensibility comes from leveraging features and functionality provided by [Docker](https://docs.docker.com/) and [Docker Compose](https://docs.docker.com/compose/overview/). Some working knowledge of these tools is required in order to customize or extend the environment DDEV provides.

There are [many examples of custom docker-compose files](https://github.com/ddev/ddev-contrib#additional-services-added-via-docker-composeserviceyaml). The best examples are in the many available maintained DDEV add-ons.

## Background

Under the hood, DDEV uses a private copy of docker-compose to define and run the multiple containers that make up the local environment for a project. `docker-compose` (also called `docker compose`) supports defining multiple compose files to facilitate sharing Compose configurations between files and projects, and DDEV is designed to leverage this ability.

To add custom configuration or additional services to your project, create `docker-compose` files in the `.ddev` directory. DDEV will process any files with the `docker-compose.*.yaml` naming convention and merge them into a full docker-compose file.

!!!warning "Don’t modify `.ddev/.ddev-docker-compose-base.yaml` or `.ddev/.ddev-docker-compose-full.yaml`!"

    The main docker-compose file is `.ddev/.ddev-docker-compose-base.yaml`, reserved exclusively for DDEV’s use. It’s overwritten every time a project is started, so any edits will be lost. If you need to add configuration, use an additional `.ddev/docker-compose.<whatever>.yaml` file instead.

## `docker-compose.*.yaml` Examples

For most HTTP-based services, use `expose` with `HTTP_EXPOSE` and `HTTPS_EXPOSE` environment variables. This approach allows multiple projects to run simultaneously without port conflicts:

```yaml
services:
  dummy-service:
    container_name: "ddev-${DDEV_SITENAME}-dummy-service"
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

!!!warning "Avoid using `ports` - it prevents multiple projects from running"

    Direct port binding with `ports` should be avoided for most services because it prevents multiple projects with the same service from running simultaneously. Only use `ports` for non-HTTP services that cannot work through the DDEV router.

**Only use `ports` for non-HTTP services** that must bind directly to localhost, such as database connections or other protocols that cannot be routed through HTTP/HTTPS:

```yaml
services:
  special-service:
    ports:
    - "9999:9999"  # Use only when HTTP routing is not possible
```

### Customizing Existing Services

You can also modify existing DDEV services like the `web` container without adding new services. This is useful for adding environment variables, volumes, or build customizations:

```yaml
services:
  web:
    environment:
      - CUSTOM_ENV_VAR=value
    volumes:
      - ./custom-config:/etc/custom-config:ro
```

For more complex customizations, you can add a custom build stage to an existing service:

```yaml
services:
  web:
    build:
      context: .
      dockerfile_inline: |
        FROM ddev-webserver
        RUN apt-get update && apt-get install -y custom-package
        COPY custom-script.sh /usr/local/bin/
```

## Confirming docker-compose Configurations

To better understand how DDEV parses your custom docker-compose files, run `ddev utility compose-config` or review the `.ddev/.ddev-docker-compose-full.yaml` file. This prints the final, DDEV-generated docker-compose configuration when starting your project.

## Conventions for Defining Additional Services

When defining additional services for your project, we recommend following these conventions to ensure DDEV handles your service the same way DDEV handles default services.

* The container name should be `ddev-${DDEV_SITENAME}-<servicename>`. This ensures the auto-generated [Traefik routing configuration](./traefik-router.md#project-traefik-configuration) matches your custom service.
* Provide containers with required labels:

    ```yaml
    services:
      dummy-service:
        image: ${YOUR_DOCKER_IMAGE:-example/example:latest}
        labels:
          com.ddev.site-name: ${DDEV_SITENAME}
          com.ddev.approot: ${DDEV_APPROOT}
    ```

* When using a custom `build` configuration with `dockerfile_inline` or `Dockerfile`, define the `image` with the `-${DDEV_SITENAME}-built` suffix:

    ```yaml
    services:
      dummy-service:
        image: ${YOUR_DOCKER_IMAGE:-example/example:latest}-${DDEV_SITENAME}-built
        build:
          dockerfile_inline: |
            ARG YOUR_DOCKER_IMAGE="scratch"
            FROM $${YOUR_DOCKER_IMAGE}
            # ...
          args:
            YOUR_DOCKER_IMAGE: ${YOUR_DOCKER_IMAGE:-example/example:latest}
    ```

    This enables DDEV to operate in [offline mode](../usage/offline.md) once the base image has been pulled.

* Exposing ports for service: you can expose the port for a service to be accessible as `projectname.ddev.site:portNum` while your project is running. This is achieved by the following configurations for the container(s) being added:

    * Define only the internal port in the `expose` section for docker-compose; use `ports:` only if the port will be bound directly to `localhost`, as may be required for non-HTTP services.

    * To expose a web interface to be accessible over HTTP, define the following environment variables in the `environment` section for docker-compose:

        * `VIRTUAL_HOST=$DDEV_HOSTNAME` You can set a subdomain with `VIRTUAL_HOST=mysubdomain.$DDEV_HOSTNAME`. You can also specify an arbitrary hostname like `VIRTUAL_HOST=extra.ddev.site`.
        * `HTTP_EXPOSE=portNum` The `hostPort:containerPort` convention may be used here to expose a container’s port to a different external port. To expose multiple ports for a single container, define the ports as comma-separated values.
        * `HTTPS_EXPOSE=<exposedPortNumber>:portNum` This will expose an HTTPS interface on `<exposedPortNumber>` to the host (and to the `web` container) as `https://<project>.ddev.site:exposedPortNumber`. To expose multiple ports for a single container, use comma-separated definitions, as in `HTTPS_EXPOSE=9998:80,9999:81`, which would expose HTTP port 80 from the container as `https://<project>.ddev.site:9998` and HTTP port 81 from the container as `https://<project>.ddev.site:9999`.

## Interacting with Additional Services

[`ddev exec`](../usage/commands.md#exec), [`ddev ssh`](../usage/commands.md#ssh), and [`ddev logs`](../usage/commands.md#logs) interact with containers on an individual basis.

By default, these commands interact with the `web` container for a project. All of these commands, however, provide a `--service` or `-s` flag allowing you to specify the service name of the container to interact with. For example, if you added a service to provide Apache Solr, and the service was named `solr`, you would be able to run `ddev logs --service solr` to retrieve the Solr container’s logs.

## Third Party Services May Need To Trust `ddev-webserver`

Sometimes a third-party service (defined in a `.ddev/docker-compose.*.yaml` file) needs to consume content from the `ddev-webserver` container. For example, a PDF generator such as [Gotenberg](https://github.com/gotenberg/gotenberg) might need to read in-container images or text to generate a PDF, or a testing service might need to read data to perform tests.

By default, a third-party service does not trust DDEV's `mkcert` certificate authority (CA). In such cases, you have three main options:

* Use plain HTTP between the containers.
* Configure the third-party service to ignore HTTPS/TLS errors.
* Make the third-party service trust DDEV's CA.

### Option 1: Use HTTP Between Containers

Using HTTP is the simplest approach. For instance, the [`ddev-selenium-standalone-chrome`](https://github.com/ddev/ddev-selenium-standalone-chrome) service consumes web content by accessing the `ddev-webserver` over plain HTTP, see [its configuration here](https://github.com/ddev/ddev-selenium-standalone-chrome/blob/main/config.selenium-standalone-chrome.yaml#L17). In this case, the `selenium-chrome` container interacts with the `web` container via `http://web` instead of HTTPS.

### Option 2: Ignore TLS Errors

Another approach is to configure the third-party service to ignore certificate errors.  
For example, if it uses cURL, you can disable verification with:

```bash
curl --insecure https://web
# or
curl --insecure https://<project>.ddev.site
```

### Option 3: Make the Container Trust DDEV's Certificate Authority

A more advanced solution is to configure the third-party container to trust the same self-signed certificate used by the `ddev-webserver` container:

```yaml
# .ddev/docker-compose.example.yaml
services:
  example:
    container_name: ddev-${DDEV_SITENAME}-example
    # Run `mkcert -install` on container start
    # (choose either this or the `post_start` approach, not both):
    command: "bash -c 'mkcert -install && original-start-command-from-image'"
    # Or run `mkcert -install` on container post_start
    # (choose either this or the `command` approach, not both):
    post_start:
      - command: mkcert -install
    # Add an image and a build stage so we can add `mkcert`, etc.
    # The Dockerfile for the build stage goes in the `.ddev/example/` directory
    image: ${YOUR_DOCKER_IMAGE:-example/example:latest}-${DDEV_SITENAME}-built
    build:
      context: example
      args:
        YOUR_DOCKER_IMAGE: ${YOUR_DOCKER_IMAGE:-example/example:latest}
    environment:
      - HTTP_EXPOSE=3001:3000
      - HTTPS_EXPOSE=3000:3000
      - VIRTUAL_HOST=$DDEV_HOSTNAME
    # Adding external_links allows connections to `https://example.ddev.site`,
    # which then can go through `ddev-router`
    # Tip: external_links are not needed anymore in DDEV v1.24.9+
    external_links:
      - ddev-router:${DDEV_SITENAME}.${DDEV_TLD}
    labels:
      com.ddev.approot: ${DDEV_APPROOT}
      com.ddev.site-name: ${DDEV_SITENAME}
    restart: 'no'
    volumes:
      - .:/mnt/ddev_config
      # `ddev-global-cache` gets mounted so we have the CAROOT
      # This is required so that the CA is available for `mkcert` to install
      # and for custom commands to work
      - ddev-global-cache:/mnt/ddev-global-cache
```

And the corresponding `Dockerfile`:

```Dockerfile
# .ddev/example/Dockerfile
ARG YOUR_DOCKER_IMAGE="scratch"
FROM $YOUR_DOCKER_IMAGE
# Define CAROOT for mkcert
ENV CAROOT=/mnt/ddev-global-cache/mkcert
# Switch to root if needed (skip if already root)
USER root
# Optionally install sudo if missing
RUN (apt-get update || true) && apt-get install -y --no-install-recommends sudo \
    && rm -rf /var/lib/apt/lists/*
# Allow the `example` user passwordless sudo for `mkcert -install`
RUN mkdir -p /etc/sudoers.d && \
    echo "example ALL=(ALL) NOPASSWD: ALL" > /etc/sudoers.d/example && \
    chmod 0440 /etc/sudoers.d/example
# Install mkcert for the correct architecture
ARG TARGETARCH
RUN mkdir -p /usr/local/bin && \
    curl --fail -JL -s -o /usr/local/bin/mkcert "https://dl.filippo.io/mkcert/latest?for=linux/${TARGETARCH}" && \
    chmod +x /usr/local/bin/mkcert
# Switch back to non-root user
USER example
```

## Matching Container User to Host User

When mounting host directories for file editing, it's often necessary to align container user permissions with those of the host user. This prevents ownership or permission mismatches when files are created or modified inside the container.

### Option 1: Run as the Host User

The simplest way is to run the container using the same UID and GID as the host user. DDEV automatically provides these values through its [environment variables](./custom-commands.md#environment-variables-provided) `DDEV_UID` and `DDEV_GID`.

```yaml
# .ddev/docker-compose.example.yaml
services:
  example:
    container_name: ddev-${DDEV_SITENAME}-example
    image: ${YOUR_DOCKER_IMAGE:-example/example:latest}
    labels:
      com.ddev.approot: ${DDEV_APPROOT}
      com.ddev.site-name: ${DDEV_SITENAME}
    restart: 'no'
    # Run the container as the same user/group as the host
    user: "${DDEV_UID}:${DDEV_GID}"
    volumes:
      - .:/mnt/ddev_config
      - ddev-global-cache:/mnt/ddev-global-cache
      # Mount the project root to /var/www/html inside the container
      - ../:/var/www/html
```

### Option 2: Create Matching User Inside Container

If you need a more sophisticated user setup, similar to what `ddev-webserver` uses, you can create a user inside the container during the build process that matches the host user's UID and GID.

```yaml
# .ddev/docker-compose.example.yaml
services:
  example:
    container_name: ddev-${DDEV_SITENAME}-example
    image: ${YOUR_DOCKER_IMAGE:-example/example:latest}-${DDEV_SITENAME}-built
    build:
      context: example
      args:
        YOUR_DOCKER_IMAGE: ${YOUR_DOCKER_IMAGE:-example/example:latest}
        username: ${DDEV_USER}
        uid: ${DDEV_UID}
        gid: ${DDEV_GID}
    labels:
      com.ddev.approot: ${DDEV_APPROOT}
      com.ddev.site-name: ${DDEV_SITENAME}
    restart: 'no'
    # Run the container as the same user/group as the host
    user: "${DDEV_UID}:${DDEV_GID}"
    volumes:
      - .:/mnt/ddev_config
      - ddev-global-cache:/mnt/ddev-global-cache
      # Mount the project root to /var/www/html inside the container
      - ../:/var/www/html
```

And the corresponding `Dockerfile`:

```Dockerfile
# .ddev/example/Dockerfile
ARG YOUR_DOCKER_IMAGE="scratch"
FROM $YOUR_DOCKER_IMAGE
# Switch to root if needed (skip if already root)
USER root
# Allow large UIDs and GIDs (in case of very large values on host)
RUN printf "UID_MAX 2147483647\nGID_MAX 2147483647\n" >> /etc/login.defs
# Accept build arguments for user creation
ARG username
ARG uid
ARG gid
# Ensure tty group exists
RUN getent group tty || groupadd tty
# Create group and user, trying multiple methods for compatibility
RUN (groupadd --gid "$gid" "$username" || groupadd "$username" || true) && \
    (useradd -G tty -l -m -s "/bin/bash" --gid "$username" --comment '' --uid "$uid" "$username" || \
    useradd -G tty -l -m -s "/bin/bash" --gid "$username" --comment '' "$username" || \
    useradd -G tty -l -m -s "/bin/bash" --gid "$gid" --comment '' "$username" || \
    useradd -G tty -l -m -s "/bin/bash" --comment '' "$username")
# Switch to the created user
USER "$username"
```

## Optional Services

Services in named Docker Compose profiles will not automatically be started on `ddev start`. This is useful when you want to define a service that is not always needed, but can be started by an additional command when it is time to use it. In this way, it doesn't use system resources unless needed. In this example, the `busybox` container will only be started if the `busybox` profile is requested, for example with `ddev start --profiles=busybox`. More than one service can be labeled for a single Docker Compose profile.

!!!tip "Run `ddev start --profiles='*'` to start all defined profiles."

```yaml
services:
  busybox:
    image: busybox:stable
    command: tail -f /dev/null
    profiles:
      - busybox
    container_name: ddev-${DDEV_SITENAME}-busybox
    labels:
      com.ddev.site-name: ${DDEV_SITENAME}
      com.ddev.approot: ${DDEV_APPROOT}
```
