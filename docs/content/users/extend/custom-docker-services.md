---
search:
  boost: 2
---

# Custom Docker Compose Services

When you need services that aren't available as DDEV add-ons, or require deep customization beyond what add-ons provide, you can create custom Docker Compose services using `docker-compose.*.yaml` files.

!!!tip "From Custom Services to Add-ons"
    Many successful custom services eventually become DDEV add-ons so they can be shared with teams, between projects, or with the broader community. If you find your custom service useful and stable, consider converting it to an add-on using the [DDEV Add-on Template](https://github.com/ddev/ddev-addon-template).

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

## Creating Custom Services

Create `docker-compose.*.yaml` files in your project's `.ddev` directory. DDEV automatically processes any files matching this pattern and merges them into the full docker-compose configuration.

### Basic Service Example

Create `.ddev/docker-compose.myservice.yaml`:

```yaml
services:
  myservice:
    container_name: "ddev-${DDEV_SITENAME}-myservice"
    image: nginx:alpine
    labels:
      com.ddev.site-name: ${DDEV_SITENAME}
      com.ddev.approot: ${DDEV_APPROOT}
    restart: "no"
    ports:
      - "8080"
    environment:
      - VIRTUAL_HOST=${DDEV_HOSTNAME}
      - HTTP_EXPOSE=8080:8080
      - HTTPS_EXPOSE=8081:8080
    volumes:
      - ".:/mnt/ddev_config"
```

### Service Configuration Best Practices

#### Required Labels

Always include these labels for proper DDEV integration:

```yaml
labels:
  com.ddev.site-name: ${DDEV_SITENAME}
  com.ddev.approot: ${DDEV_APPROOT}
```

#### Container Naming

Use consistent naming with the DDEV project:

```yaml
container_name: "ddev-${DDEV_SITENAME}-servicename"
```

#### Restart Policy

Set restart policy to prevent issues:

```yaml
restart: "no"
```

#### Port Exposure

For HTTP services that should be accessible via ddev-router:

```yaml
ports:
  - "8080"  # Expose port to Docker network
environment:
  - VIRTUAL_HOST=${DDEV_HOSTNAME}
  - HTTP_EXPOSE=8080:8080    # HTTP access
  - HTTPS_EXPOSE=8081:8080   # HTTPS access
```

For direct port binding (can cause conflicts between projects):

```yaml
ports:
  - "9999:9999"  # Bind to host port 9999
```

#### Volume Mounts

Mount your `.ddev` directory for configuration access:

```yaml
volumes:
  - ".:/mnt/ddev_config"
```

Mount project files if needed:

```yaml
volumes:
  - "../:/var/www/html:cached"
```

### Customizing `ddev describe` Output and Container Shell

You can use the `x-ddev` extension field in your `.ddev/docker-compose.*.yaml` configuration to customize the output of [`ddev describe`](../usage/commands.md#describe) and set the default shell for `ddev exec` or `ddev ssh`.

This feature is useful for:

- Displaying credentials, URLs, or usage instructions for custom services.
- Changing the default shell used inside containers.

```yaml
services:
  rabbitmq:
    container_name: "ddev-${DDEV_SITENAME}-rabbitmq"
    image: rabbitmq:3-management-alpine
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
      # Custom shell (must be installed in the image)
      shell: "bash"
```

The `x-ddev.describe-url-port` value appears in the `URL/PORT` column when running [`ddev describe`](../usage/commands.md#describe) and the `x-ddev-describe-info` value appears in the `INFO` column, making it easy for team members to see important service information without digging through documentation and configuration files.

The `x-ddev.shell` value defines the default shell for [`ddev exec`](../usage/commands.md#exec) and [`ddev ssh`](../usage/commands.md#ssh). Ensure the shell (e.g., Zsh or Bash) is installed in the image, otherwise these commands will fail:

Example: changing the default shell to Zsh inside the `web` container:

```yaml
# .ddev/config.yaml
webimage_extra_packages: [zsh]
```

```yaml
# .ddev/docker-compose.web-shell.yaml
services:
  web:
    x-ddev:
      shell: "zsh"
```

To change the shell for a custom service, add the `x-ddev.shell` field to that service's configuration and ensure the desired shell is [installed in the image](./customizing-images.md) if needed.

## Advanced Service Examples

### SQL Server Database Service

This example shows a custom SQL Server database service, useful when you need a database not natively supported by DDEV.

Create `.ddev/docker-compose.sqlsrv.yaml`:

```yaml
services:
  sqlsrv:
    container_name: "ddev-${DDEV_SITENAME}-sqlsrv"
    image: mcr.microsoft.com/mssql/server:2022-latest
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

Create `.ddev/docker-compose.elasticsearch.yaml`:

```yaml
services:
  elasticsearch:
    container_name: "ddev-${DDEV_SITENAME}-elasticsearch"
    image: elasticsearch:8.11.0
    labels:
      com.ddev.site-name: ${DDEV_SITENAME}
      com.ddev.approot: ${DDEV_APPROOT}
    restart: "no"
    ports:
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

Create `.ddev/docker-compose.cache.yaml`:

```yaml
services:
  redis:
    container_name: "ddev-${DDEV_SITENAME}-redis"
    image: redis:7-alpine
    labels:
      com.ddev.site-name: ${DDEV_SITENAME}
      com.ddev.approot: ${DDEV_APPROOT}
    restart: "no"
    ports:
      - "6379"

  memcached:
    container_name: "ddev-${DDEV_SITENAME}-memcached"
    image: memcached:alpine
    labels:
      com.ddev.site-name: ${DDEV_SITENAME}
      com.ddev.approot: ${DDEV_APPROOT}
    restart: "no"
    ports:
      - "11211"
```

## Environment Variables and Configuration

### Available DDEV Variables

Here is a compressed list of commonly used variables in your service definitions:

- `${DDEV_SITENAME}` - Project name
- `${DDEV_HOSTNAME}` - Comma-separated list of FQDN hostnames
- `${DDEV_TLD}` - Default top-level domain (`ddev.site`)
- `${DDEV_APPROOT}` - Full path to project root
- `${DDEV_DOCROOT}` - Document root (relative to project root)
- `${DDEV_PHP_VERSION}` - PHP version
- `${DDEV_WEBSERVER_TYPE}` - Web server type
- `${DDEV_DATABASE_FAMILY}` - Database family (`mysql`, `postgres`)

For a full list, please see [Environment Variables Provided](custom-commands.md#environment-variables-provided).

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

## Testing and Debugging Services

### Check Service Status

```bash
ddev logs myservice
```

### Verify Configuration

```bash
ddev utility compose-config
```

This shows the complete merged docker-compose configuration.

### Connect to Service

```bash
ddev exec --service=myservice bash
```

### Network Connectivity

Test connectivity from the web container:

```bash
ddev exec "curl -s http://myservice:8080/health"
```

## Service Integration Patterns

### Database Integration

Add database connection info to web container in `.ddev/docker-compose.web-env.yaml`:

```yaml
services:
  web:
    environment:
      - MYDB_HOST=mydb
      - MYDB_PORT=5432
      - MYDB_DATABASE=myproject
      - MYDB_USER=db
      - MYDB_PASSWORD=db
```

### Configuration File Mounting

Mount configuration from your project in `.ddev/docker-compose.config.yaml`:

```yaml
services:
  myservice:
    volumes:
      - "./config/myservice.conf:/etc/myservice/myservice.conf:ro"
```

### Initialization Scripts

Run initialization scripts in `.ddev/docker-compose.init.yaml`:

```yaml
services:
  myservice:
    volumes:
      - "./scripts/init.sh:/docker-entrypoint-initdb.d/init.sh:ro"
```

## Troubleshooting

### Common Issues

**Port conflicts**: Multiple projects using the same service may conflict. Use project-specific ports or let DDEV handle routing.

**Service won't start**: Check `ddev logs servicename` for error messages.

**Network connectivity**: Ensure services are on the same Docker network (automatic with DDEV).

**File permissions**: Use appropriate volume mount options (`:cached`, `:ro`).

### Debugging Steps

1. Verify syntax: `ddev utility compose-config`
2. Check logs: `ddev logs servicename`
3. Test connectivity: `ddev exec "ping servicename"`
4. Inspect container: `ddev exec --service=servicename bash`

## Migration from ddev-contrib

Many services previously documented in [ddev-contrib](https://github.com/ddev/ddev-contrib) have been converted to official add-ons. Check [DDEV Add-on Registry](https://addons.ddev.com/) first before creating custom services.

### Still Available in ddev-contrib

- **Old PHP Versions**: [Old PHP Versions](https://github.com/ddev/ddev-contrib/blob/master/docker-compose-services/old_php)
- **Specialized configurations**: Various experimental and niche services

## Best Practices

### Performance

- Use specific image tags instead of `latest`
- Set appropriate resource limits
- Use volume caching options (`:cached`)
- Minimize container layers and size

### Security

- Don't expose unnecessary ports to the host
- Use non-root users when possible

### Maintainability

- Document your service configuration
- Use meaningful container names
- Group related services in single files
- Comment complex configurations

### Team Sharing

- Include service documentation in your project readme
- Use environment variables for customizable values
- Provide setup and testing instructions
- Consider creating an add-on for reusable services

## Converting to Add-ons

If your custom service becomes stable and useful for multiple projects, consider converting it to a DDEV add-on. This allows you to:

- Share the service with your team across projects
- Contribute to the DDEV community
- Benefit from automatic installation and configuration
- Add version management and updates

**Steps to convert:**

1. Create an add-on repository from [DDEV Add-on Template](https://github.com/ddev/ddev-addon-template)
2. Move your service configuration to the add-on
3. Add installation actions and configuration options
4. Create tests and documentation
5. Publish and share with the community

See [Creating Add-ons](creating-add-ons.md) for detailed instructions.

Custom Docker Compose services provide the ultimate flexibility for customizing your DDEV environment. While add-ons are recommended for common services, custom services let you integrate exactly what your project needs, with the potential to evolve into shareable add-ons.
