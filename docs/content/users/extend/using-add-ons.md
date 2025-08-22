---
search:
  boost: 3
---

# Using DDEV Add-ons

DDEV add-ons are pre-packaged extensions that add functionality to your development environment with a single command. They handle installation, configuration, and integration automatically.

## Add-ons vs. Custom Docker Compose Services

**Use add-ons when:**

- A standard, tested service is available as an add-on (Redis, Elasticsearch, Solr)
- You want automatic configuration and setup

**Use custom Docker Compose services when:**

- You need a custom or highly specialized service
- You require deep customization of the service configuration
- You're prototyping or experimenting with service configurations

See [Defining Additional Services with Docker Compose](custom-compose-files.md) for custom Docker Compose service setup.

## Discovering Add-ons

### Web-based Add-on Registry

Use [DDEV Add-on Registry](https://addons.ddev.com/) to discover, explore, and leave comments on available add-ons.

### Command Line Discovery

List official add-ons:

```bash
ddev add-on list
```

See all possible add-ons (including community add-ons):

```bash
ddev add-on list --all
```

## Installing Add-ons

Install any add-on using the repository format:

```bash
ddev add-on get <repo/name>
```

Popular examples:

```bash
ddev add-on get ddev/ddev-redis
ddev add-on get ddev/ddev-elasticsearch
ddev add-on get ddev/ddev-solr
ddev add-on get ddev/ddev-memcached
ddev add-on get ddev/ddev-adminer
```

You can also specify a particular version:

```bash
ddev add-on get ddev/ddev-redis --version v1.0.4
```

Add-ons are installed into your project's `.ddev` directory and automatically integrated with your project configuration.

## Managing Add-ons

### View Installed Add-ons

```bash
ddev add-on list --installed
```

### Update an Add-on

```bash
ddev add-on get <repo/name>
```

This updates to the latest version while preserving your customizations.

### Remove an Add-on

```bash
ddev add-on remove <addon-name>
```

This cleanly removes all add-on files and configurations.

## Customizing Add-on Configuration

Sometimes you need to customize an add-on's default configuration.

### Method 1: Environment Variables (Recommended)

Many add-ons support customization through environment variables. For example, to change the Redis version in [`ddev-redis`](https://github.com/ddev/ddev-redis):

```bash
ddev dotenv set .ddev/.env.redis --redis-tag 7-bookworm
ddev restart
```

This sets `REDIS_TAG="7-bookworm"` which the add-on uses during service startup.

You can also edit the `.ddev/.env.redis` file directly:

```dotenv
REDIS_TAG="7-bookworm"
REDIS_FOO="bar"
```

!!!tip "Check add-on documentation"
    Each add-on documents its available environment variables. Check the add-on's GitHub repository for configuration options.

### Method 2: Docker Compose Override

For more complex customizations, create an override file. For example, `.ddev/docker-compose.redis_extra.yaml`:

```yaml
services:
  redis:
    image: redis:7-bookworm
    command: ["redis-server", "--maxmemory", "256mb"]
```

This approach:

- Maintains your customizations when updating the add-on
- Allows complex service modifications
- Doesn't require modifying the original add-on files

!!!note
    Remove the `#ddev-generated` line from any add-on file you customize directly, but using override files is preferred.

## Official Add-ons

### Database and Caching

- **[`ddev/ddev-redis`](https://github.com/ddev/ddev-redis)** - Redis cache and data store service
- **[`ddev/ddev-memcached`](https://github.com/ddev/ddev-memcached)** - High-performance Memcached caching service
- **[`ddev/ddev-mongo`](https://github.com/ddev/ddev-mongo)** - MongoDB database support

### Search and Analytics

- **[`ddev/ddev-elasticsearch`](https://github.com/ddev/ddev-elasticsearch)** - Elasticsearch full-text search and analytics engine
- **[`ddev/ddev-opensearch`](https://github.com/ddev/ddev-opensearch)** - OpenSearch analytics, logging, and full-text search
- **[`ddev/ddev-solr`](https://github.com/ddev/ddev-solr)** - Apache Solr server setup for search indexing
- **[`ddev/ddev-drupal-solr`](https://github.com/ddev/ddev-drupal-solr)** - Apache Solr search engine integration for Drupal

### Development Tools

- **[`ddev/ddev-adminer`](https://github.com/ddev/ddev-adminer)** - Adminer web-based MySQL, MariaDB, PostgreSQL database browser
- **[`ddev/ddev-phpmyadmin`](https://github.com/ddev/ddev-phpmyadmin)** - Web-based phpMyAdmin interface for MySQL, MariaDB
- **[`ddev/ddev-redis-commander`](https://github.com/ddev/ddev-redis-commander)** - Redis Commander Web UI for use with Redis service
- **[`ddev/ddev-browsersync`](https://github.com/ddev/ddev-browsersync)** - Live-reload and HTTPS auto-refresh on file changes

### Platform and Cloud Integration

- **[`ddev/ddev-platformsh`](https://github.com/ddev/ddev-platformsh)** - Platform.sh integration for project syncing and workflows
- **[`ddev/ddev-ibexa-cloud`](https://github.com/ddev/ddev-ibexa-cloud)** - Pull projects and data from Ibexa Cloud
- **[`ddev/ddev-minio`](https://github.com/ddev/ddev-minio)** - MinIO S3-compatible object storage solution

### Specialized Services

- **[`ddev/ddev-rabbitmq`](https://github.com/ddev/ddev-rabbitmq)** - RabbitMQ message broker, queue manager
- **[`ddev/ddev-cron`](https://github.com/ddev/ddev-cron)** - Run scheduled tasks and cron jobs inside web container
- **[`ddev/ddev-ioncube`](https://github.com/ddev/ddev-ioncube)** - Enable ionCube PHP loaders for encoded files
- **[`ddev/ddev-selenium-standalone-chrome`](https://github.com/ddev/ddev-selenium-standalone-chrome)** - Headless Chrome browser testing with Selenium

### Development Environment

- **[`ddev/ddev-drupal-contrib`](https://github.com/ddev/ddev-drupal-contrib)** - Contrib module development environment for Drupal projects

## Troubleshooting Add-ons

### Check Add-on Status

```bash
ddev describe
```

```bash
ddev logs -s <service>
```

This shows logs from an add-on's service.

### Explore Add-on Files

```bash
ls -la .ddev/
```

Look for files created by the add-on, typically:

- `docker-compose.<addon-name>.yaml`
- Configuration files in `.ddev/<addon-name>/`
- Custom commands in `.ddev/commands/`

### Restart Services

```bash
ddev restart
```

This restarts all services and applies any configuration changes.

### Review Add-on Configuration

```bash
ddev debug compose-config
```

This shows the final Docker Compose configuration including add-on services.

### Common Issues

**Service not starting**: Check `ddev logs -s <service>` for error messages from the add-on service.

**Configuration not applied**: Ensure you've run `ddev restart` after making configuration changes.

## Getting Help

- **Add-on documentation**: Check the add-on's GitHub repository README
- **DDEV Discord**: Join the [DDEV Discord](https://ddev.com/s/discord) for community support
- **GitHub Issues**: Report add-on-specific issues to the add-on's repository
- **Stack Overflow**: Use the [ddev tag](https://stackoverflow.com/tags/ddev)

## Next Steps

- **Create custom add-ons**: See [Creating Add-ons](creating-add-ons.md)
- **Manual services**: See [Defining Additional Services with Docker Compose](custom-compose-files.md)
- **Advanced customization**: See [Extending and Customizing Environments](customization-extendibility.md)
