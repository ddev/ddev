---
search:
  boost: 2
---

# Custom Docker Compose Services

When you need services that aren't available as DDEV add-ons, or require deep customization beyond what add-ons provide, you can create custom Docker Compose services using `docker-compose.*.yaml` files. For how DDEV merges these files and the conventions your service should follow (labels, container naming, the `build:`/`-built` tag pattern, port exposure), see [Defining Additional Services with Docker Compose](custom-compose-files.md).

!!!tip "From Custom Services to Add-ons"
    Many successful custom services eventually become DDEV add-ons so they can be shared with teams, between projects, or with the broader community. If you find your custom service useful and stable, convert it to an add-on using the [DDEV Add-on Template](https://github.com/ddev/ddev-addon-template) and see [Creating Add-ons](creating-add-ons.md).

## When to Use Custom Services

**Use custom Docker Compose services when:**

- You need a custom or highly specialized service
- You require deep customization of service configuration
- You're prototyping or experimenting with service configurations
- The service doesn't justify creating a full add-on yet
- You need tight integration with your specific project setup

**Use add-ons when:**

- An add-on is already available that provides a standard, tested service (Redis, Elasticsearch, Solr)
- You want automatic configuration and setup

See [Using Add-ons](using-add-ons.md) for pre-built add-ons.

Whichever you choose, the same conventions apply once you have a compose file — see [Defining Additional Services with Docker Compose](custom-compose-files.md#conventions-for-defining-additional-services).

## Creating Custom Services

A custom service typically runs a container based on a Docker image and provides a specific "service". It's defined in a `.ddev/docker-compose.*.yaml`. DDEV automatically processes any files matching this pattern and merges them into the full Compose configuration.

### Basic Service Example

Create `.ddev/docker-compose.myservice.yaml`:

```yaml
services:
  myservice:
    container_name: "ddev-${DDEV_SITENAME}-myservice"
    image: nginx:alpine
    # These two labels are added automatically since DDEV v1.25.2+
    labels:
      com.ddev.site-name: ${DDEV_SITENAME}
      com.ddev.approot: ${DDEV_APPROOT}
    restart: "no"
    expose:
      - "8080"
    environment:
      - VIRTUAL_HOST=${DDEV_HOSTNAME}
      - HTTP_EXPOSE=8080:8080
      - HTTPS_EXPOSE=8081:8080
    volumes:
      - ".:/mnt/ddev_config"
```

### Service Configuration Best Practices

Required labels, container naming, restart policy, and HTTP vs. direct port binding are covered in [Conventions for Defining Additional Services](custom-compose-files.md#conventions-for-defining-additional-services) — follow those for every custom service.

#### Volume Mounts

Provide a bind-mount of your `.ddev` directory for configuration access:

```yaml
volumes:
  - ".:/mnt/ddev_config"
```

Mount project files if needed:

```yaml
volumes:
  - "../:/var/www/html:cached"
```

## `x-ddev` Extension

The `x-ddev` extension field lets you customize DDEV behavior per service in your `.ddev/docker-compose.*.yaml` files.

| Key | Description |
|-----|-------------|
| [`describe-url-port`](#customizing-ddev-describe-output) | Text shown in the `URL/PORT` column of `ddev describe` |
| [`describe-info`](#customizing-ddev-describe-output) | Text shown in the `INFO` column of `ddev describe` |
| [`ssh-shell`](../extend/in-container-configuration.md#changing-ddev-ssh-shell) | Shell used by `ddev ssh -s <service>` for this service |
| [`omit-ddev-labels`](#omitting-comddev-labels-from-a-service) | Skip injecting `com.ddev.*` labels onto this service |
<!-- TODO: support the ssh-user key when https://github.com/ddev/ddev/pull/8829 lands -->

### Customizing `ddev describe` Output

You can use the `x-ddev` extension field in your `.ddev/docker-compose.*.yaml` configuration to customize the output of [`ddev describe`](../usage/commands.md#describe).

This feature is useful for showing credentials, URLs, or usage notes for custom services.

```yaml
services:
  rabbitmq:
    container_name: "ddev-${DDEV_SITENAME}-rabbitmq"
    image: rabbitmq:3-management-alpine
    # These two labels are added automatically since DDEV v1.25.2+
    labels:
      com.ddev.site-name: ${DDEV_SITENAME}
      com.ddev.approot: ${DDEV_APPROOT}
    restart: "no"
    expose:
      - "15672"
    environment:
      - VIRTUAL_HOST=${DDEV_HOSTNAME}
      - HTTP_EXPOSE=15672:15672
      - HTTPS_EXPOSE=15673:15672
      - RABBITMQ_DEFAULT_USER=rabbitmq
      - RABBITMQ_DEFAULT_PASS=rabbitmq
    x-ddev:
      # Can be multi-line block
      describe-info: |
        User: rabbitmq
        Pass: rabbitmq
      # Or single line string
      describe-url-port: "extra help here"
```

- `x-ddev.describe-url-port`: Appears in the `URL/PORT` column when running [`ddev describe`](../usage/commands.md#describe).
- `x-ddev.describe-info`: Appears in the `INFO` column, making it easy for team members to view relevant service details without checking config files.

!!!tip
    See related `x-ddev.ssh-shell` configuration for [Changing `ddev ssh` Shell](../extend/in-container-configuration.md#changing-ddev-ssh-shell).

### Omitting `com.ddev.*` Labels from a Service

Setting `x-ddev.omit-ddev-labels: true` stops DDEV from injecting `com.ddev.*` labels onto a service.

DDEV normally stamps these labels (like `com.ddev.site-name`) on every service and has `ddev start` wait for the matching containers. That makes `ddev start` fail when a one-shot container exits before the wait completes. Omitting the labels drops the service from that wait, while leaving it in the compose file so `ddev stop`, `ddev poweroff`, and `ddev delete` still tear it down.

```yaml
services:
  # One-shot container: prepares the shared volume, then exits
  init:
    container_name: "ddev-${DDEV_SITENAME}-init"
    image: busybox:stable
    command:
      - sh
      - -c
      - |
        mkdir -p /mnt/assets/cache &&
        chown -R ${DDEV_UID}:${DDEV_GID} /mnt/assets
    volumes:
      - "assets:/mnt/assets"
    x-ddev:
      omit-ddev-labels: true

  # web mounts the same volume
  web:
    volumes:
      - "assets:/mnt/assets"
    depends_on:
      init:
        condition: service_completed_successfully

volumes:
  assets:
```

## Advanced Service Examples

### Service with a Custom Build

If the stock image needs an extra tool or package, add a `build:` section instead of a plain `image:`. Follow the `-${DDEV_SITENAME}-built` tag convention from [Conventions for Defining Additional Services](custom-compose-files.md#conventions-for-defining-additional-services), and give the tag a unique segment per service, as shown here, so two build services sharing the same base image don't overwrite each other's built image.

Create `.ddev/docker-compose.myservice.yaml`:

```yaml
services:
  myservice:
    container_name: "ddev-${DDEV_SITENAME}-myservice"
    image: ${BASE_IMAGE:-nginx:alpine}-${DDEV_SITENAME}-myservice-built
    build:
      dockerfile_inline: |
        ARG BASE_IMAGE="debian"
        FROM $${BASE_IMAGE}
        RUN apt-get update && apt-get install -y curl
      args:
        BASE_IMAGE: ${BASE_IMAGE:-debian}
    # These two labels are added automatically since DDEV v1.25.2+
    labels:
      com.ddev.site-name: ${DDEV_SITENAME}
      com.ddev.approot: ${DDEV_APPROOT}
    restart: "no"
```

### SQL Server Database Service

This example shows a custom SQL Server database service, useful when you need a database not natively supported by DDEV.

!!!tip "A maintained add-on already exists"
    This example grew into the [`ddev-sqlsrv`](https://github.com/ddev/ddev-sqlsrv) add-on. Its `sqlsrv` service skips host `ports:` entirely and is reached only from the `web` container over the Docker network, so it avoids the port conflict noted below — use that pattern instead of the one here unless you specifically need to connect from the host.

Create `.ddev/docker-compose.sqlsrv.yaml`:

```yaml
services:
  sqlsrv:
    container_name: "ddev-${DDEV_SITENAME}-sqlsrv"
    image: mcr.microsoft.com/mssql/server:2022-latest
    # These two labels are added automatically since DDEV v1.25.2+
    labels:
      com.ddev.site-name: ${DDEV_SITENAME}
      com.ddev.approot: ${DDEV_APPROOT}
    restart: "no"
    ports:
      - "1433:1433"  # Direct port binding for SQL Server protocol
    environment:
      - SA_PASSWORD=Password123!
      - ACCEPT_EULA=Y
      - MSSQL_PID=Express
    volumes:
      - "sqlsrv-data:/var/opt/mssql"
      - ".:/mnt/ddev_config"
    # Platform specification for ARM64 compatibility
    platform: linux/amd64

volumes:
  sqlsrv-data:
    external: true
    name: "${DDEV_SITENAME}-sqlsrv-data"
```

!!!note "Non-HTTP Services Require Direct Port Binding"
    SQL Server uses a proprietary protocol that cannot be routed through the DDEV router, so it requires direct `ports` binding. This means only one project can use SQL Server at a time unless you change the port.

### Service with Custom Configuration

!!!tip "A maintained add-on already exists"
    This example grew into the [ddev-elasticsearch](https://github.com/ddev/ddev-elasticsearch) add-on, which also adds a `healthcheck:` and supports switching Elasticsearch major versions.

Create `.ddev/docker-compose.elasticsearch.yaml`:

```yaml
services:
  elasticsearch:
    container_name: "ddev-${DDEV_SITENAME}-elasticsearch"
    image: elasticsearch:8.11.0
    # These two labels are added automatically since DDEV v1.25.2+
    labels:
      com.ddev.site-name: ${DDEV_SITENAME}
      com.ddev.approot: ${DDEV_APPROOT}
    restart: "no"
    expose:
      - "9200"
    environment:
      - VIRTUAL_HOST=${DDEV_HOSTNAME}
      - HTTP_EXPOSE=9200:9200
      - discovery.type=single-node
      - xpack.security.enabled=false
      - "ES_JAVA_OPTS=-Xms512m -Xmx512m"
    volumes:
      - "elasticsearch-data:/usr/share/elasticsearch/data"
      - "./elasticsearch/config/elasticsearch.yml:/usr/share/elasticsearch/config/elasticsearch.yml:ro"

volumes:
  elasticsearch-data:
    external: true
    name: "${DDEV_SITENAME}-elasticsearch-data"
```

### Multi-Service Setup

It's usually easier and clearer to have a separate `docker-compose.*.yaml` file for each service, but it's possible to define more than one in a single file.

!!!tip "Maintained add-ons already exist"
    These two grew into the [ddev-redis](https://github.com/ddev/ddev-redis) and [ddev-memcached](https://github.com/ddev/ddev-memcached) add-ons. Neither binds a host port at all, since both are reached only from the `web` container — that's why `expose:`, not `ports:`, is used below.

Create `.ddev/docker-compose.cache.yaml`:

```yaml
services:
  redis:
    container_name: "ddev-${DDEV_SITENAME}-redis"
    image: redis:7-alpine
    # These two labels are added automatically since DDEV v1.25.2+
    labels:
      com.ddev.site-name: ${DDEV_SITENAME}
      com.ddev.approot: ${DDEV_APPROOT}
    restart: "no"
    expose:
      - "6379"

  memcached:
    container_name: "ddev-${DDEV_SITENAME}-memcached"
    image: memcached:alpine
    # These two labels are added automatically since DDEV v1.25.2+
    labels:
      com.ddev.site-name: ${DDEV_SITENAME}
      com.ddev.approot: ${DDEV_APPROOT}
    restart: "no"
    expose:
      - "11211"
```

## Environment Variables and Configuration

DDEV variables such as `${DDEV_SITENAME}` and `${DDEV_APPROOT}` are available for interpolation in your service definitions; see [Environment Variables Provided](custom-commands.md#environment-variables-provided) for the full list.

### Custom Environment Variables

Define project-specific variables in `.ddev/.env`:

```dotenv
MYSERVICE_VERSION=latest
MYSERVICE_MEMORY=512m
```

Then use in your service:

```yaml
services:
  myservice:
    image: myservice:${MYSERVICE_VERSION:-latest}
    environment:
      - MEMORY_LIMIT=${MYSERVICE_MEMORY:-256m}
```

### Service-Specific Environment Files

Use `.ddev/.env.servicename` for service-specific variables:

```bash
ddev dotenv set .ddev/.env.myservice --memory-limit 1024m --debug-mode true
```

Variables from every `.ddev/.env*` file are available for interpolation here, but only a file named after an existing service sets variables inside that container. See [Environment Variables](../configuration/environment-variables.md) for the file naming rules, the global `$HOME/.ddev/.env*` equivalents, and where to keep a secret.

## Debugging a Custom Service

`ddev logs --service myservice` and `ddev exec --service myservice bash` cover most cases; see [Interacting with Additional Services](custom-compose-files.md#interacting-with-additional-services). To see how DDEV merged your compose files, run `ddev utility compose-config`.

Custom services let you integrate exactly what your project needs, with the potential to evolve into a shareable add-on — see the tip at the top of this page.
