# Config Options

DDEV configuration is stored in YAML files that come in two flavors:

1. :octicons-file-directory-16: **Project** `.ddev/config.yaml` settings, with optional [environmental override](#environmental-overrides) variants.
2. :octicons-globe-16: **Global** `$HOME/.ddev/global_config.yaml` settings that can apply to all projects.

Most of these settings take effect when you run [`ddev start`](../basics/commands.md#start).

## Managing Configuration

### Setting Options

You can hand-edit the YAML files DDEV creates for you after running [`ddev config`](../basics/commands.md#config), and you can also define most settings with equivalent CLI arguments:

=== "config.yaml"

    ```yaml
    php_version: "8.1"
    ```
=== "`ddev config`"

    ```shell
    ddev config --php-version 8.1
    ```

    Run `ddev help config` to see all the available config arguments.

### Environmental Overrides

You can override the per-project `config.yaml` with files named `config.*.yaml`, which are gitignored by default and not checked in.

Many teams use `config.local.yaml` for configuration that’s specific to one environment, and not checked into the team’s default `config.yaml`. You might [enable Mutagen](../install/performance.md#mutagen) or [enable NFS](../install/performance.md#nfs) for the project, for example, just on your machine. Or maybe use a different database type.

For examples, see the [Extending and Customizing Environments](../extend/customization-extendibility.md#extending-configyaml-with-custom-configyaml-files) page.

---

## `additional_fqdns`

An array of [extra fully-qualified domain names](../extend/additional-hostnames.md) to be used for a project.

| Type | Default | Usage
| -- | -- | --
| :octicons-file-directory-16: project | `[]` |

Example: `additional_fqdns: ["example.com", "sub1.example.com"]` would provide HTTP and HTTPS URLs for `example.com` and `sub1.example.com`.

!!!warning
    Take care with `additional_fqdns`; it adds items to your `/etc/hosts` file which can cause confusion.

## `additional_hostnames`

An array of [extra hostnames](../extend/additional-hostnames.md) to be used for a project.

| Type | Default | Usage
| -- | -- | --
| :octicons-file-directory-16: project | `[]` |

Example: `additional_hostnames: ["somename", "someothername", "*.thirdname"]` would provide HTTP and HTTPS URLs for `somename.ddev.site`, `someothername.ddev.site`, and `one.thirdname.ddev.site` + `two.thirdname.ddev.site`.

The wildcard (`*`) setting only works if you’re using DNS to resolve hostnames (default) and connected to the internet.

## `composer_root`

The relative path, from the project root, to the directory containing `composer.json`. (This is where all Composer-related commands are executed.)

| Type | Default | Usage
| -- | -- | --
| :octicons-file-directory-16: project | |

## `composer_version`

Composer version for the web container and the [`ddev composer`](../basics/commands.md#composer) command.

| Type | Default | Usage
| -- | -- | --
| :octicons-file-directory-16: project | `2` | Can be `2`, `1`, or empty (`""`) for latest major version at container build time.<br><br>Can also be a minor version like `2.2` for the latest release of that branch, an explicit version like `1.0.22`, or a keyword like `stable`, `preview` or `snapshot`. See Composer documentation.

## `database`

The type and version of the database engine the project should use.

| Type | Default | Usage
| -- | -- | --
| :octicons-file-directory-16: project | MariaDB 10.4 | Can be MariaDB 5.5–10.7, MySQL 5.5–8.0, or PostgreSQL 9–15.<br>See [Database Server Types](../extend/database-types.md) for examples and caveats.

!!!note
    DDEV v1.18 and earlier used `mariadb_version` and `mysql_version`, which are now automatically converted to the `database` format.

## `dbimage_extra_packages`

Extra Debian packages for the project’s database container.

| Type | Default | Usage
| -- | -- | --
| :octicons-file-directory-16: project | `[]` |

Example: `dbimage_extra_packages: ["less"]` will add the `less` package when the database container is built.

## `default_container_timeout`

Seconds DDEV will wait for all containers to become ready.

| Type | Default | Usage
| -- | -- | --
| :octicons-file-directory-16: project | `120` | Can be adjusted to avoid timeouts on slower systems or for huge snapshot restores.

## `developer_mode`

Not currently used.

| Type | Default | Usage
| -- | -- | --
| :octicons-globe-16: global | `false` | Can `true` or `false`.

## `disable_http2`

Whether to disable http/2 listen in `ddev-router`.

| Type | Default | Usage
| -- | -- | --
| :octicons-globe-16: global | `false` | Can be `true` or `false`.

When `true`, nginx will not listen for http/2, but just use http/1.1 SSL. (Some browsers don’t work well with http/2.)

## `disable_settings_management`

Whether to disable CMS-specific settings file management.

| Type | Default | Usage
| -- | -- | --
| :octicons-file-directory-16: project | `false` | Can be `true` or `false`.

When `true`, DDEV will not create or update CMS-specific settings files.

## `docroot`

Relative path to the document root containing `index.php` or `index.html`.

| Type | Default | Usage
| -- | -- | --
| :octicons-file-directory-16: project | automatic | DDEV will attempt to detect this and set it for you, otherwise falling back to the current directory.

## `fail_on_hook_fail`

Whether [`ddev start`](../basics/commands.md#start) should be interrupted by a failing [hook](../configuration/hooks.md), on a single project or for all projects if used globally.

| Type | Default | Usage
| -- | -- | --
| :octicons-file-directory-16: project<br>:octicons-globe-16: global | `false` | Can be `true` or `false`.

## `hooks`

DDEV-specific lifecycle [hooks](hooks.md) to be executed.

| Type | Default | Usage
| -- | -- | --
| :octicons-file-directory-16: project | `` |

## `host_db_port`

Port for binding database server to localhost interface.

| Type | Default | Usage
| -- | -- | --
| :octicons-file-directory-16: project | automatic |

Not commonly used. Can be a specific port number for a fixed database port. If unset, the port will be assigned automatically and change each time [`ddev start`](../basics/commands.md#start) is run.

Can be a specific port number for a fixed database port, which can be useful for configuration of host-side database clients. (May still be easier to use [`ddev mysql`](../basics/commands.md#mysql), `ddev psql`, `ddev sequelpro`, etc., which handle changing ports automatically, as does the sample command `ddev mysqlworkbench`.)

## `host_https_port`

Specific, persistent HTTPS port for direct binding to localhost interface.

| Type | Default | Usage
| -- | -- | --
| :octicons-file-directory-16: project | automatic |

Not commonly used. Can be a specific port number for a fixed HTTPS URL. If unset, the port will be assigned automatically and change each time [`ddev start`](../basics/commands.md#start) is run.

Example: `59001` will have the project always use `https://127.0.0.1:59001` for the localhost URL—used less commonly than the named URL which is better to rely on.

## `host_webserver_port`

Specific, persistent HTTP port for direct binding to localhost interface.

| Type | Default | Usage
| -- | -- | --
| :octicons-file-directory-16: project | automatic |

Not commonly used. Can be a specific port number for a fixed HTTP URL. If unset, the port will be assigned automatically and change each time [`ddev start`](../basics/commands.md#start) is run.

Example: `59000` will have the project always use `http://127.0.0.1:59000` for the localhost URL—used less commonly than the named URL which is better to rely on.

## `instrumentation_opt_in`

Whether to allow [instrumentation reporting](../details/opting-in.md).

| Type | Default | Usage
| -- | -- | --
| :octicons-globe-16: global | `true` | Can be `true` or `false`.

When `true`, anonymous usage information is collected via [Segment](https://segment.com).

## `internet_detection_timeout_ms`

Internet detection timeout in milliseconds.

| Type | Default | Usage
| -- | -- | --
| :octicons-globe-16: global | `1000` (1 second) | Can be any integer.

DDEV must detect whether the internet is working to determine whether to add hostnames to `/etc/hosts`. In rare cases, you may need to increase this value if you have slow but working internet. See [FAQ](../basics/faq.md) and [GitHub issue](https://github.com/drud/ddev/issues/2409#issuecomment-662448025).

## `letsencrypt_email`

Email associated with Let’s Encrypt feature. (Works in conjunction with [`use_letsencrypt`](#use_letsencrypt).)

| Type | Default | Usage
| -- | -- | --
| :octicons-globe-16: global | `` |

Set with `ddev global --letsencrypt-email=me@example.com`. Used with the [casual hosting](../details/alternate-uses.md#casual-hosting) feature.

## `mailhog_https_port`

Port for project’s MailHog HTTPS URL.

| Type | Default | Usage
| -- | -- | --
| :octicons-file-directory-16: project | `8026` | Can be changed to avoid a port conflict.

## `mailhog_port`

Port for project’s MailHog HTTP URL.

| Type | Default | Usage
| -- | -- | --
| :octicons-file-directory-16: project | `8025` | Can be changed to avoid a port conflict.

## `mutagen_enabled`

Whether to enable [Mutagen asynchronous caching](../install/performance.md#mutagen) for all projects.

| Type | Default | Usage
| -- | -- | --
| :octicons-file-directory-16: project<br>:octicons-globe-16: global | `false` | Can be `true` or `false`; only `true` has any effect.

This overrides NFS mounting as it’s incompatible with NFS.

## `name`

The URL-friendly name DDEV should use to reference the project.

| Type | Default | Usage
| -- | -- | --
| :octicons-file-directory-16: project | enclosing directory name | Must be unique; no two projects can have the same name. It’s best if this matches the directory name. If this option is omitted, the project will take the name of the enclosing directory.

## `nfs_mount_enabled`

Whether to use [NFS](../install/performance.md#nfs) to mount the project into the container for performance. (Mostly superseded by [`mutagen_enabled`](#mutagen_enabled).)

| Type | Default | Usage
| -- | -- | --
| :octicons-file-directory-16: project<br>:octicons-globe-16: global | `false` | Can be `true` or `false`; only `true` has any effect.

!!!tip "Workstation configuration required!"
    See the [NFS section](../install/performance.md#nfs) on the Performance page.

This is typically a global setting. If it’s ever set in both places, the global config will override the project-specific value.

## `ngrok_args`

Extra flags for [configuring ngrok](https://ngrok.com/docs/ngrok-agent/config) when [sharing projects](../topics/sharing.md) with the [`ddev share`](../basics/commands.md#share) command.

| Type | Default | Usage
| -- | -- | --
| :octicons-file-directory-16: project | `` |

Example: `--basic-auth username:pass1234`.

## `no_bind_mounts`

Whether to not use Docker bind mounts.

| Type | Default | Usage
| -- | -- | --
| :octicons-globe-16: global | `false` | Can `true` or `false`.

Some Docker environments (like remote Docker) do not allow bind mounts, so when `true` this turns those off, turns on Mutagen, and uses volume copies to do what bind mounts would otherwise do.

## `no_project_mount`

Whether to skip mounting project into web container.

| Type | Default | Usage
| -- | -- | --
| :octicons-file-directory-16: project | `false` | Can be `true` or `false`.

!!!warning "Advanced users only!"
    When `true`, project will not be mounted by DDEV into the web container. Enables experimentation with alternate file mounting strategies.

## `nodejs_version`

Node.js version for the web container’s “system” version.

| Type | Default | Usage
| -- | -- | --
| :octicons-file-directory-16: project | current LTS version | Can be `14`, `16`, or `18`.

`nvm` is also available inside the container and via [`ddev nvm`](../basics/commands.md#nvm), and can be set to any valid version including much older ones.

## `omit_containers`

Containers that should not be loaded automatically for one or more projects.

| Type | Default | Usage
| -- | -- | --
| :octicons-file-directory-16: project<br>:octicons-globe-16: global | `[]` | **For projects**, can include `db`, `dba`, and `ddev-ssh-agent`.<br>**Globally**, can include `dba` and `ddev-ssh-agent` only.

Example: `omit_containers: [db, dba, ddev-ssh-agent]` starts the project without a `db` container, phpMyAdmin, and SSH agent. Some containers can be omitted globally in `~/.ddev/global_config.yaml` and the result is additive; all containers named in both places will be omitted.

!!!warning
    Omitting the `db` container will cause database-dependent DDEV features to be unstable.

## `override_config`

Whether to override config values instead of merging.

| Type | Default | Usage
| -- | -- | --
| :octicons-file-directory-16: project | `false` | Can be `true` or `false`.

When `true`, the `config.*.yaml` file with the option will have its settings *override* rather than *merge with* others. Allows statements like `nfs_mount_enabled: false` or `additional_hostnames: []` to work.

See [Extending `config.yaml` with Custom `config.*.yaml` Files](../extend/customization-extendibility.md#extending-configyaml-with-custom-configyaml-files).

## `php_version`

The PHP version the project should use.

| Type | Default | Usage
| -- | -- | --
| :octicons-file-directory-16: project | `8.0` | Can be `5.6`, `7.0`, `7.1`, `7.2`, `7.3`, `7.4`, `8.0`, `8.1`, or `8.2`.

You can only specify the major version (`7.3`), not a minor version (`7.3.2`), from those explicitly available.

## `phpmyadmin_https_port`

Port for project’s phpMyAdmin HTTPS URL.

| Type | Default | Usage
| -- | -- | --
| :octicons-file-directory-16: project | `8037` | Can be changed to avoid a port conflict.

## `phpmyadmin_port`

Port for project’s phpMyAdmin HTTP URL.

| Type | Default | Usage
| -- | -- | --
| :octicons-file-directory-16: project | `8036` | Can be changed to avoid a port conflict.

## `project_tld`

Default TLD to be used for a project’s domains, or globally for all project domains.

| Type | Default | Usage
| -- | -- | --
| :octicons-file-directory-16: project<br>:octicons-globe-16: global | `ddev.site` | Can be changed to any TLD you’d prefer.

## `required_docker_compose_version`

Specific docker-compose version for download.

| Type | Default | Usage
| -- | -- | --
| :octicons-globe-16: global | |

If set to `v2.8.0`, for example, it will download and use that version instead of the expected version for docker-compose.

!!!warning "Troubleshooting Only!"
    This should only be used in specific cases like troubleshooting. Best avoided otherwise.

## `router_bind_all_interfaces`

Whether to bind `ddev-router`'s ports on all network interfaces.

| Type | Default | Usage
| -- | -- | --
| :octicons-globe-16: global | `false` | Can be `true` or `false`.

When `true`, `ddev-router` will bind on all network interfaces instead of only `localhost`, exposing DDEV projects to your local network. If you set this to `true`, consider setting `omit_containers: ["dba"]` so phpMyAdmin is not available.

## `router_http_port`

Port for DDEV router’s HTTP traffic.

| Type | Default | Usage
| -- | -- | --
| :octicons-file-directory-16: project | `80` | Usually changed only if there’s a conflicting process using that port.

See the [Troubleshooting](../basics/troubleshooting.md#web-server-ports-already-occupied) page for more on addressing port conflicts.

## `router_https_port`

Port for DDEV router’s HTTPS traffic.

| Type | Default | Usage
| -- | -- | --
| :octicons-file-directory-16: project | `443` | Usually changed only if there’s a conflicting process using that port.

See the [Troubleshooting](../basics/troubleshooting.md#web-server-ports-already-occupied) page for more on addressing port conflicts.

## `simple-formatting`

Whether to disable most [`ddev list`](../basics/commands.md#list) and [`ddev describe`](../basics/commands.md#describe) table formatting.

| Type | Default | Usage
| -- | -- | --
| :octicons-globe-16: global | `false` | Can be `true` or `false`.

When `true`, turns off most table formatting in [`ddev list`](../basics/commands.md#list) and [`ddev describe`](../basics/commands.md#describe) and suppresses colorized text everywhere.

## `table-style`

Style for [`ddev list`](../basics/commands.md#list) and [`ddev describe`](../basics/commands.md#describe).

| Type | Default | Usage
| -- | -- | --
| :octicons-globe-16: global | `default` | Can be `default`, `bold`, and `bright`.

`bright` is a pleasant, colorful output some people may prefer.

## `timezone`

Timezone for container and PHP configuration.

| Type | Default | Usage
| -- | -- | --
| :octicons-file-directory-16: project | `UTC` | Can be any [valid timezone](https://en.wikipedia.org/wiki/List_of_tz_database_time_zones), like `Europe/Dublin` or `MST7MDT`.

## `type`

The DDEV-specific project type.

| Type | Default | Usage
| -- | -- | --
| :octicons-file-directory-16: project | `php` | Can be `backdrop`, `craftcms`, `drupal6`, `drupal7`, `drupal8`, `drupal9`, `drupal10`,  `laravel`, `magento`, `magento2`, `php`, `shopware6`, `typo3`, or `wordpress`.

The `php` type doesn’t attempt [CMS configuration](../../users/quickstart.md) or settings file management and can work with any project.

## `upload_dir`

Path from the project’s docroot to the user-generated files directory targeted by `ddev import-files`.

| Type | Default | Usage
| -- | -- | --
| :octicons-file-directory-16: project | |

## `use_dns_when_possible`

Whether to use DNS instead of editing `/etc/hosts`.

| Type | Default | Usage
| -- | -- | --
| :octicons-file-directory-16: project | `true` | Can be `true` or `false`.

When `false`, DDEV will always update the `/etc/hosts` file with the project hostname instead of using DNS for name resolution.

See [Using DDEV Offline](../details/offline-usage.md).

## `use_docker_compose_from_path`

Whether to use the system-installed docker-compose. You can otherwise use [`required_docker_compose_version`](#required_docker_compose_version) to specify a version for download.

| Type | Default | Usage
| -- | -- | --
| :octicons-globe-16: global | `false` | Can `true` or `false`.

When `true`, DDEV will use the docker-compose found in on your system’s path instead of using its private, known-good, docker-compose version.

!!!warning "Troubleshooting Only!"
    This should only be used in specific cases like troubleshooting. Best avoided otherwise.

## `use_hardened_images`

Whether to use hardened images for internet deployment.

| Type | Default | Usage
| -- | -- | --
| :octicons-globe-16: global | `false` | Can `true` or `false`.

When `true`, more secure hardened images are used for an internet deployment. These do not include sudo in the web container, and the container is run without elevated privileges. Generally used with the [casual hosting](../details/alternate-uses.md#casual-hosting) feature.

## `use_letsencrypt`

Whether to enable Let’s Encrypt integration. (Works in conjunction with [`letsencrypt_email`](#letsencrypt_email).)

| Type | Default | Usage
| -- | -- | --
| :octicons-globe-16: global | `false` | Can `true` or `false`.

May also be set via `ddev global --use-letsencrypt` or `ddev global --use-letsencrypt=false`. When `true`, `letsencrypt_email` must also be set and the system must be available on the internet. Used with the [casual hosting](../details/alternate-uses.md#casual-hosting) feature.

## `web_environment`

Additional [custom environment variables](../extend/customization-extendibility.md#providing-custom-environment-variables-to-a-container) for a project’s web container. (Or for all projects if used globally.)

| Type | Default | Usage
| -- | -- | --
| :octicons-file-directory-16: project<br>:octicons-globe-16: global | `[]` |

## `web_extra_daemons`

Additional daemons that should [automatically be started in the web container](../extend/customization-extendibility.md#running-extra-daemons-in-the-web-container).

| Type | Default | Usage
| -- | -- | --
| :octicons-file-directory-16: project | `[]` |

## `web_extra_exposed_ports`

Additional named sets of ports to [expose via `ddev-router`](../extend/customization-extendibility.md#exposing-extra-ports-via-ddev-router).

| Type | Default | Usage
| -- | -- | --
| :octicons-file-directory-16: project | `[]` |

## `webimage`

The Docker image to use for the web server.

| Type | Default | Usage
| -- | -- | --
| :octicons-file-directory-16: project | [`drud/ddev-webserver`](https://hub.docker.com/r/drud/ddev-webserver) | Specify your own image based on [drud/ddev-webserver](https://github.com/drud/ddev/tree/master/containers/ddev-webserver).

!!!warning "Proceed with caution"
    It’s unusual to change this, and we don’t recommend it without Docker experience and a good reason. Typically, this means additions to the existing web image using a `.ddev/web-build/Dockerfile.*`.

## `webimage_extra_packages`

Extra Debian packages for the project’s web container.

| Type | Default | Usage
| -- | -- | --
| :octicons-file-directory-16: project | `[]` |

Example: `webimage_extra_packages: [php-yaml, php-bcmath]` will add the `php-yaml` and `php-bcmath` packages when the web container is built.

## `webserver_type`

Whether Xdebug should be enabled for [step debugging](../debugging-profiling/step-debugging.md) or [profiling](../debugging-profiling/xdebug-profiling.md).

| Type | Default | Usage
| -- | -- | --
| :octicons-file-directory-16: project | `nginx-fpm` | Can be `nginx-fpm` or `apache-fpm`.

## `working_dir`

Working directories used by [`ddev exec`](../basics/commands.md#exec) and [`ddev ssh`](../basics/commands.md#ssh).

| Type | Default | Usage
| -- | -- | --
| :octicons-file-directory-16: project | |

Example: `working_dir: { web: "/var/www", db: "/etc" }` sets the working directories for the `web` and `db` containers.

## `xdebug_enabled`

Whether Xdebug should be enabled for [step debugging](../debugging-profiling/step-debugging.md) or [profiling](../debugging-profiling/xdebug-profiling.md).

| Type | Default | Usage
| -- | -- | --
| :octicons-file-directory-16: project | `false` | Please leave this `false` in most cases. Most people use [`ddev xdebug`](../basics/commands.md#xdebug) and `ddev xdebug off` commands.

## `xdebug_ide_location`

Adjust Xdebug listen location for WSL2 or in-container.

| Type | Default | Usage
| -- | -- | --
| :octicons-globe-16: global | `""` | Can be empty (`""`), `"wsl2"`, `"container"`, or an explicit IP address.

For PhpStorm running inside WSL2 (or JetBrains Gateway), use `"wsl2"`. For in-container like VS Code Language Server, set to `"container"`. It can also be set to an explicit IP address.
