---
search:
  boost: 2
---

# Additional Service Configurations & Add-ons

DDEV projects can be extended using add-ons or custom Docker Compose services.

## DDEV Add-ons (Recommended)

Add-ons are pre-packaged extensions that provide services with automatic installation and configuration.

**See [Using Add-ons](using-add-ons.md) for:**

- Installing add-ons with `ddev add-on get`
- Managing and customizing add-ons
- Official add-on catalog

**See [Creating Add-ons](creating-add-ons.md) for:**

- Building your own add-ons
- Using PHP or bash actions
- Publishing and sharing add-ons

## Custom Docker Compose Services

For specialized needs or deep customization, you can create custom services using `docker-compose.*.yaml` files.

**See [Custom Docker Compose Services](custom-docker-services.md) for:**

- Manual service configuration
- Advanced service patterns
- Converting services to add-ons

## Quick Reference

### Using Add-ons

```bash
# Discover add-ons
ddev add-on list
ddev add-on list --all

# Install add-ons  
ddev add-on get ddev/ddev-redis
ddev add-on get ddev/ddev-elasticsearch

# Manage add-ons
ddev add-on list --installed
ddev add-on remove redis
```

### Custom Services

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
```

## Resources

- **[DDEV Add-on Registry](https://addons.ddev.com/)** - Browse and discover add-ons
- **[ddev-addon-template](https://github.com/ddev/ddev-addon-template)** - Template for creating add-ons
- **[ddev-contrib](https://github.com/ddev/ddev-contrib)** - Community examples and specialized services
