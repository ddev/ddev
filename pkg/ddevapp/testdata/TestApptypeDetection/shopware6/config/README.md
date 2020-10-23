# Configuration

This README describes how to change the configuration.

## Overview

```text
config/
├── bundles.php       # defines static symfony bundles - use plugins for dynamic bundles
├── etc               # contains the configuration of the docker image
├── jwt               # secrets for generating jwt tokens - DO NOT COMMIT these secrets
├── packages/         # package configuration
├── services/         # additional service configuration files
├── README.md         # this file
├── services.xml      # just imports the defaults
└── services_test.xml # just imports the test defaults
```

## `config/bundles.php`

The `bundles.php` defines all static bundles the kernel should load. If
you dont need our storefront or the administration you can remove the
bundle from this file and it will stop being loaded. To completely remove
it you can also stop requiring the package in the `composer.json`.

## `config/packages/*.yml`

`.yml` files for packages contained in this directory are loaded automatically.

### Shopware config `config/packages/shopware.yml`

Define shopware specific configuration.

This file can be added to override the defaults defined in `vendor/shopware/core/Framework/Resources/config/packages/shopware.yaml`.

Example:

```yaml
shopware:
    api:
        max_limit: 1000 # change limit from 500 to 1000

    admin_worker:
        enable_admin_worker: false # disable admin worker - use a different one!

    auto_update:
        enabled: false # disable auto update

```

## `config/services.xml`

Imports defaults from `config/services/defaults.xml` and can be used to override and add service definitions and parameters.
