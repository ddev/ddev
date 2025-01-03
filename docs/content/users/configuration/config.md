---
search:
  boost: 0.5
---

# Config Options

DDEV configuration is stored in YAML files that come in two flavors:

1. :octicons-file-directory-16: **Project** `.ddev/config.yaml` settings, with optional [environmental override](#environmental-overrides) variants.
2. :octicons-globe-16: **Global** `$HOME/.ddev/global_config.yaml` settings that can apply to all projects.

Most of these settings take effect when you run [`ddev start`](../usage/commands.md#start).

## Managing Configuration

### Setting Options

You can hand-edit the YAML files DDEV creates for you after running [`ddev config`](../usage/commands.md#config), and you can also define most settings with equivalent CLI arguments:

=== "config.yaml"

    ```yaml
    php_version: "8.3"
    ```
=== "`ddev config`"

    ```shell
    ddev config --php-version 8.3
    ```

    Run `ddev help config` to see all the available config arguments.

### Environmental Overrides

You can override the per-project `config.yaml` with files named `config.*.yaml`, and files like this are often created by [DDEV add-ons](../extend/additional-services.md). For example, `config.elasticsearch.yaml` in [Elasticsearch add-on](https://github.com/ddev/ddev-elasticsearch) adds additional configuration related to Elasticsearch.

Many teams use `config.local.yaml` for configuration that is specific to one environment, and not checked into the team’s default `config.yaml`. You might [enable Mutagen](../install/performance.md#mutagen) or [enable NFS](../install/performance.md#nfs) for the project, for example, only on your machine. Or maybe use a different database type. The file `config.local.yaml` is gitignored by default.

For examples, see the [Extending and Customizing Environments](../extend/customization-extendibility.md#extending-configyaml-with-custom-configyaml-files) page.

---

## `additional_fqdns`

An array of [extra fully-qualified domain names](../extend/additional-hostnames.md) to be used for a project.

| Type | Default | Usage
| -- | -- | --
| :octicons-file-directory-16: project | `[]` | &zwnj;

Example: `additional_fqdns: ["example.com", "sub1.example.com"]` would provide HTTP and HTTPS URLs for `example.com` and `sub1.example.com`.

See [Hostnames and Wildcards and DDEV, Oh My!](https://ddev.com/blog/ddev-name-resolution-wildcards/) for more information on DDEV hostname resolution.

!!!warning
    Take care with `additional_fqdns`; it adds items to your `/etc/hosts` file which can cause confusion.

## `additional_hostnames`

An array of [extra hostnames](../extend/additional-hostnames.md) to be used for a project.

| Type | Default | Usage
| -- | -- | --
| :octicons-file-directory-16: project | `[]` | &zwnj;

Example: `additional_hostnames: ["somename", "someothername", "*.thirdname"]` would provide HTTP and HTTPS URLs for `somename.ddev.site`, `someothername.ddev.site`, and `one.thirdname.ddev.site` + `two.thirdname.ddev.site`.

The wildcard (`*.<whatever>`) setting only works if you’re [using DNS to resolve hostnames (default)](#use_dns_when_possible) and connected to the internet and using `ddev.site` as your [`project_tld`](#project_tld).

See [Hostnames and Wildcards and DDEV, Oh My!](https://ddev.com/blog/ddev-name-resolution-wildcards/) for more information on DDEV hostname resolution.

## `bind_all_interfaces`

When the network interfaces of a project should be exposed to the local network, you can specify `bind_all_interfaces: true` to do that. This is an unusual application, sometimes used to [share projects on a local network](../topics/sharing.md#exposing-a-host-port-and-providing-a-direct-url).

| Type | Default | Usage
| -- | -- | --
| :octicons-file-directory-16: project | `false` | Can be `true` or `false`.

## `composer_root`

The relative path, from the project root, to the directory containing `composer.json`. (This is where all Composer-related commands are executed.)

| Type | Default | Usage
| -- | -- | --
| :octicons-file-directory-16: project | &zwnj; | &zwnj;

## `composer_version`

Composer version for the web container and the [`ddev composer`](../usage/commands.md#composer) command.

| Type | Default | Usage
| -- | -- | --
| :octicons-file-directory-16: project | `2` | Can be `2`, `1`, or empty (`""`) for latest major version at container build time.<br><br>Can also be a minor version like `2.2` for the latest release of that branch, an explicit version like `1.0.22`, or a keyword like `stable`, `preview` or `snapshot`. See Composer documentation.

!!!tip "How to run Composer from `vendor/bin/composer`?"

    ```shell
    ddev exec vendor/bin/composer --version
    # If you have a custom composer_root:
    ddev exec '$DDEV_COMPOSER_ROOT/vendor/bin/composer --version'
    ```

## `corepack_enable`

Whether to `corepack enable` on Node.js configuration.

| Type | Default | Usage
| -- | -- | --
| :octicons-file-directory-16: project | `false` | Can be `true` or `false`.

When `true`, `corepack_enable` will be executed, making latest `yarn` and `pnpm` package managers available.

## `database`

The type and version of the database engine the project should use.

| Type | Default       | Usage
| -- |---------------| --
| :octicons-file-directory-16: project | MariaDB 10.11 | Can be MariaDB 5.5–10.8, 10.11, or 11.4, MySQL 5.5–8.0, or PostgreSQL 9–15.<br>See [Database Server Types](../extend/database-types.md) for examples and caveats.

## `dbimage_extra_packages`

Extra Debian packages for the project’s database container. (This is rarely used.)

| Type | Default | Usage
| -- | -- | --
| :octicons-file-directory-16: project | `[]` | &zwnj;

Example: `dbimage_extra_packages: ["less"]` will add the `less` package when the database container is built.

## `ddev_version_constraint`

You can configure a [version constraint](https://github.com/Masterminds/semver#checking-version-constraints) for DDEV that will be validated against the running DDEV executable and prevent `ddev start` from running if it doesn't validate. For example:

```yaml
ddev_version_constraint: '>= v1.24.0-alpha1'
```

| Type | Default | Usage
| -- | -- | --
| :octicons-file-directory-16: project | | `>= 1.23.4`

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

## `disable_settings_management`

Whether to disable CMS-specific settings file management.

| Type | Default | Usage
| -- | -- | --
| :octicons-file-directory-16: project | `false` | Can be `true` or `false`.

When `true`, DDEV will not create or update CMS-specific settings files.

## `disable_upload_dirs_warning`

Whether to disable the standard warning issued when a project is using `performance_mode: mutagen` but `upload_dirs` is not configured.

| Type | Default | Usage
| -- | -- | --
| :octicons-file-directory-16: project | `false` | Can be `true` or `false`.

When `true`, DDEV will not issue the normal warning on `ddev start`: "You have Mutagen enabled and your 'php' project type doesn't have `upload_dirs` set". See [Mutagen and User-Generated Uploads](../install/performance.md#mutagen-and-user-generated-uploads) for context on why DDEV avoids doing the Mutagen sync on `upload_dirs`.

## `docroot`

Relative path to the document root containing `index.php` or `index.html`.

| Type | Default | Usage
| -- | -- | --
| :octicons-file-directory-16: project | automatic | DDEV will attempt to detect this and set it for you, otherwise falling back to the current directory.

## `fail_on_hook_fail`

Whether [`ddev start`](../usage/commands.md#start) should be interrupted by a failing [hook](../configuration/hooks.md), on a single project or for all projects if used globally.

| Type | Default | Usage
| -- | -- | --
| :octicons-file-directory-16: project<br>:octicons-globe-16: global | `false` | Can be `true` or `false`.

## `hooks`

DDEV-specific lifecycle [hooks](hooks.md) to be executed.

| Type | Default | Usage
| -- | -- | --
| :octicons-file-directory-16: project | `` | &zwnj;

## `host_db_port`

Port for binding database server to localhost interface.

| Type | Default | Usage
| -- | -- | --
| :octicons-file-directory-16: project | automatic | &zwnj;

Not commonly used. Can be a specific port number for a fixed database port. If unset, the port will be assigned automatically and change each time [`ddev start`](../usage/commands.md#start) is run.

Can be a specific port number for a fixed database port, which can be useful for configuration of host-side database clients. (May still be easier to use [`ddev mysql`](../usage/commands.md#mysql), `ddev psql`, `ddev sequelace`, etc., which handle changing ports automatically, as does the sample command `ddev mysqlworkbench`.)

## `host_https_port`

Specific, persistent HTTPS port for direct binding to localhost interface.

| Type | Default | Usage
| -- | -- | --
| :octicons-file-directory-16: project | automatic | &zwnj;

Not commonly used. Can be a specific port number for a fixed HTTPS URL. If unset, the port will be assigned automatically and change each time [`ddev start`](../usage/commands.md#start) is run.

Example: `59001` will have the project always use `https://127.0.0.1:59001` for the localhost URL—used less commonly than the named URL which is better to rely on.

## `host_mailpit_port`

Specific, persistent Mailpit port for direct binding to localhost interface.

| Type | Default | Usage
| -- | -- | --
| :octicons-file-directory-16: project | automatic | &zwnj;

Not commonly used. Can be a specific port number for a fixed Mailpit URL. If unset, the port will be assigned automatically and change each time [`ddev start`](../usage/commands.md#start) is run.

## `host_webserver_port`

Specific, persistent HTTP port for direct binding to localhost interface.

| Type | Default | Usage
| -- | -- | --
| :octicons-file-directory-16: project | automatic | &zwnj;

Not commonly used. Can be a specific port number for a fixed HTTP URL. If unset, the port will be assigned automatically and change each time [`ddev start`](../usage/commands.md#start) is run.

Example: `59000` will have the project always use `http://127.0.0.1:59000` for the localhost URL—used less commonly than the named URL which is better to rely on.

## `instrumentation_opt_in`

Whether to allow [instrumentation reporting](../usage/diagnostics.md).

| Type | Default | Usage
| -- | -- | --
| :octicons-globe-16: global | `true` | Can be `true` or `false`.

When `true`, anonymous usage information is collected via [Amplitude](https://amplitude.com/).

## `instrumentation_queue_size`

Maximum number of locally collected events for [instrumentation reporting](../usage/diagnostics.md).

| Type | Default | Usage
| -- | -- | --
| :octicons-globe-16: global | `100` | Can be any integer.

## `instrumentation_reporting_interval`

Reporting interval in hours for [instrumentation reporting](../usage/diagnostics.md).

| Type | Default | Usage
| -- | -- | --
| :octicons-globe-16: global | `24` | Can be any integer.

## `instrumentation_user`

Specific name identifier for [instrumentation reporting](../usage/diagnostics.md).

| Type | Default | Usage
| -- | -- | --
| :octicons-globe-16: global | `` | &zwnj;

## `internet_detection_timeout_ms`

Internet detection timeout in milliseconds.

| Type | Default | Usage
| -- | -- | --
| :octicons-globe-16: global | `3000` (3 seconds) | Can be any integer.

DDEV must detect whether the internet is working to determine whether to add hostnames to `/etc/hosts`. In rare cases, you may need to increase this value if you have slow but working internet. See [FAQ](../usage/faq.md) and [GitHub issue](https://github.com/ddev/ddev/issues/2409#issuecomment-662448025).

## `letsencrypt_email`

Email associated with Let’s Encrypt feature. (Works in conjunction with [`use_letsencrypt`](#use_letsencrypt).)

| Type | Default | Usage
| -- | -- | --
| :octicons-globe-16: global | `` | &zwnj;

Set with `ddev config global --letsencrypt-email=me@example.com`. Used with the [hosting](../topics/hosting.md) feature.

## `mailpit_http_port`

Port for project’s Mailpit HTTP URL.

| Type | Default | Usage
| -- | -- | --
| :octicons-file-directory-16: project<br>:octicons-globe-16: global | `8025` | Can be changed to avoid a port conflict.

## `mailpit_https_port`

Port for project’s Mailpit HTTPS URL.

| Type | Default | Usage
| -- | -- | --
| :octicons-file-directory-16: project<br>:octicons-globe-16: global | `8026` | Can be changed to avoid a port conflict.

## `messages`

Configure messages like the Tip of the Day.

| Type | Default | Usage
| -- | -- | --
| :octicons-globe-16: global | `ticker_interval:` | hours between ticker messages.

Example: Disable the "Tip of the Day" ticker in `~/.ddev/global_config.yaml`

```yaml
messages:
  ticker_interval: -1
```

Example: Show the "Tip of the Day" ticket every two hours:

```yaml
messages:
  ticker_interval: 2
```

## `name`

The URL-friendly name DDEV should use to reference the project.

| Type | Default | Usage
| -- | -- | --
| :octicons-file-directory-16: project | enclosing directory name | Must be unique; no two projects can have the same name. It’s best if this matches the directory name. If this option is omitted, the project will take the name of the enclosing directory. This value may also be set via `ddev config --project-name=<name>`. (The `ddev config` flag is `project-name`, not `name`, see [`ddev config` docs](../usage/commands.md#config).)"

## `ngrok_args`

Extra flags for [configuring ngrok](https://ngrok.com/docs/agent/config) when [sharing projects](../topics/sharing.md) with the [`ddev share`](../usage/commands.md#share) command.

| Type | Default | Usage
| -- | -- | --
| :octicons-file-directory-16: project | `` | &zwnj;

Example: `--basic-auth username:pass1234 --domain foo.ngrok-free.app`.

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

Node.js version for the web container’s “system” version. [`n`](https://www.npmjs.com/package/n) tool is under the hood.

There is no need to reconfigure `nodejs_version` unless you want a version other than the version already specified, which will be the default version at the time the project was configured.

Note that specifying any non-default Node.js version will cause DDEV to download and install that version when running `ddev start` the first time on a project. If optimizing first-time startup speed (as in Continuous Integration) is your biggest concern, consider using the default version of Node.js.

| Type | Default | Usage
| -- | -- | --
| :octicons-file-directory-16: project | current LTS version | any [node version](https://www.npmjs.com/package/n#specifying-nodejs-versions), like `16`, `18.2`, `18.19.2`, etc.

!!!tip "How to install the Node.js version from a file"
    Your project team may specify the Node.js version in a more general way than in the `.ddev/config.yaml`. For example, you may use a `.nvmrc` file, the `package.json`, or a similar technique. In that case, DDEV can use the external configuration provided by that file.

    There is an `auto` label (see [full documentation](https://www.npmjs.com/package/n#specifying-nodejs-versions)):

    ```bash
    ddev config --nodejs-version=auto
    ```

    It reads the target version from a file in the [DDEV_APPROOT](../extend/custom-commands.md#environment-variables-provided) directory, or any parent directory.

    `n` looks for in order:

    * `.n-node-version` : version on single line. Custom to `n`.
    * `.node-version` : version on single line. Used by [multiple tools](https://github.com/shadowspawn/node-version-usage).
    * `.nvmrc` : version on single line. Used by `nvm`.
    * if no version file found, look for `engine` as below.

    The `engine` label looks for a `package.json` file and reads the engines field to determine compatible Node.js.

    If your file is not in the `DDEV_APPROOT` directory, you can create a link to the parent folder, so that `n` can find it. For example, if you have `frontend/.nvmrc`, create a `.ddev/web-build/Dockerfile.nvmrc` file:

    ```dockerfile
    RUN ln -sf /var/www/html/frontend/.nvmrc /var/www/.nvmrc
    ```

!!!note "Switching from `nvm` to `nodejs_version`"
    If switching from using `nvm` to using `nodejs_version`, you may find that the container continues to use the previously specified version. If this happens, use `ddev nvm alias default system` or `ddev ssh` into the container (`ddev ssh`) and run `rm -rf /mnt/ddev-global-cache/nvm_dir/${DDEV_PROJECT}-web`, then `ddev restart`.

## `omit_containers`

Containers that should not be loaded automatically for one or more projects.

| Type | Default | Usage
| -- | -- | --
| :octicons-file-directory-16: project<br>:octicons-globe-16: global | `[]` | **For projects**, can include `db`, and `ddev-ssh-agent`.<br>**Globally**, can include `ddev-router`, and `ddev-ssh-agent`.

Example: `omit_containers: [db, ddev-ssh-agent]` starts the project without a `db` container and SSH agent. Some containers can be omitted globally in `~/.ddev/global_config.yaml` and the result is additive; all containers named in both places will be omitted.

!!!warning
    Omitting the `db` container will cause database-dependent DDEV features to be unstable.

## `override_config`

Whether to override config values instead of merging.

| Type | Default | Usage
| -- | -- | --
| :octicons-file-directory-16: project | `false` | Can be `true` or `false`.

When `true`, the `config.*.yaml` file with the option will have its settings *override* rather than *merge with* others. Allows statements like `use_dns_when_possible: false` or `additional_hostnames: []` to work.

See [Extending `config.yaml` with Custom `config.*.yaml` Files](../extend/customization-extendibility.md#extending-configyaml-with-custom-configyaml-files).

## `performance_mode`

Defines the performance optimization mode to be used. Currently [Mutagen asynchronous caching](../install/performance.md#mutagen) and [NFS](../install/performance.md#nfs) are supported. Mutagen is enabled by default on Mac and Windows.

| Type | Default | Usage
| -- | -- | --
| :octicons-file-directory-16: project<br>:octicons-globe-16: global | `` | Can be `global`, `none`, `mutagen`, or `nfs`.

This is typically a global setting. The project-specific value will override global config.

!!!tip "Workstation configuration is required to use NFS!"
    See the [NFS section](../install/performance.md#nfs) on the Performance page.

## `php_version`

The PHP version the project should use.

| Type | Default | Usage
| -- |---------| --
| :octicons-file-directory-16: project | `8.3`   | Can be `5.6` through `8.4`. New versions are added when released upstream.

You can only specify the major version (`7.3`), not a minor version (`7.3.2`), from those explicitly available.

## `project_tld`

Default Top-Level-Domain (`TLD`) to be used for a project’s domains, or globally for all project domains. This defaults to `ddev.site` and it's easiest to work with DDEV using the default setting.

| Type | Default | Usage
| -- | -- | --
| :octicons-file-directory-16: project<br>:octicons-globe-16: global | `ddev.site` | Can be changed to any TLD you’d prefer.

See [Hostnames and Wildcards and DDEV, Oh My!](https://ddev.com/blog/ddev-name-resolution-wildcards/) for more information on DDEV hostname resolution.

## `required_docker_compose_version`

Specific docker-compose version for download.

| Type | Default | Usage
| -- | -- | --
| :octicons-globe-16: global | &zwnj; | &zwnj;

If set to `v2.8.3`, for example, it will download and use that version instead of the expected version for docker-compose.

!!!warning "Troubleshooting Only!"
    This should only be used in specific cases like troubleshooting. Please don't experiment with it unless directed to do so.

## `router_bind_all_interfaces`

Whether to bind `ddev-router`'s ports on all network interfaces.

| Type | Default | Usage
| -- | -- | --
| :octicons-globe-16: global | `false` | Can be `true` or `false`.

When `true`, the router will bind on all network interfaces instead of only `localhost`, exposing DDEV projects to your local network. This is sometimes used to share projects on a local network, see [Sharing Your Project](../topics/sharing.md).

## `router_http_port`

Port for DDEV router’s HTTP traffic.

| Type | Default | Usage
| -- | -- | --
| :octicons-file-directory-16: project<br>:octicons-globe-16: global | `80` | Usually changed only if there’s a conflicting process using that port.

See the [Troubleshooting](../usage/troubleshooting.md#web-server-ports-already-occupied) page for more on addressing port conflicts.

## `router_https_port`

Port for DDEV router’s HTTPS traffic.

| Type | Default | Usage
| -- | -- | --
| :octicons-file-directory-16: project<br>:octicons-globe-16: global | `443` | Usually changed only if there’s a conflicting process using that port.

See the [Troubleshooting](../usage/troubleshooting.md#web-server-ports-already-occupied) page for more on addressing port conflicts.

## `simple_formatting`

Whether to disable most [`ddev list`](../usage/commands.md#list) and [`ddev describe`](../usage/commands.md#describe) table formatting.

| Type | Default | Usage
| -- | -- | --
| :octicons-globe-16: global | `false` | Can be `true` or `false`. If you don't like the table lines in `ddev list` or `ddev describe`, you can completely turn them off with `ddev config --simple-formatting=true`.

When `true`, turns off most table formatting in [`ddev list`](../usage/commands.md#list) and [`ddev describe`](../usage/commands.md#describe) and suppresses colorized text everywhere.

## `table_style`

Style for [`ddev list`](../usage/commands.md#list) and [`ddev describe`](../usage/commands.md#describe).

| Type | Default | Usage
| -- | -- | --
| :octicons-globe-16: global | `default` | Can be `default`, `bold`, and `bright`.

`bright` is a pleasant, colorful output some people may prefer. If you don't like the table lines at all, you can remove them with `ddev config --simple-formatting=true`.

## `timezone`

Timezone for container and PHP configuration.

| Type | Default | Usage
| -- | -- | --
| :octicons-file-directory-16: project | Automatic detection or `UTC` | Can be any [valid timezone](https://en.wikipedia.org/wiki/List_of_tz_database_time_zones), like `Europe/Dublin` or `MST7MDT`.

If `timezone` is unset, DDEV will attempt to derive it from the host system timezone using the `$TZ` environment variable or the `/etc/localtime` symlink.

## `traefik_monitor_port`

Specify an alternate port for the Traefik (ddev-router) monitor port. This defaults to 10999 and rarely needs to be changed, but can be changed in cases of port conflicts.

| Type | Default | Usage
| -- | -- | --
| :octicons-globe-16: global | `10999` | Can be any unused port below 65535.

## `type`

The DDEV-specific project type.

| Type | Default | Usage
| -- | -- | --
| :octicons-file-directory-16: project | `php` | Can be `backdrop`, `cakephp`, `craftcms`, `drupal`, `drupal6`, `drupal7`, `drupal8`, `drupal9`, `drupal10`, `drupal11`,   `laravel`, `magento`, `magento2`, `php`, `shopware6`, `silverstripe`, `symfony`, `typo3`, or `wordpress`.

The `php` type doesn’t attempt [CMS configuration](../../users/quickstart.md) or settings file management and can work with any project.

The many versions of the Drupal project types can be used, for example `drupal11` or `drupal6`. There is also a special `drupal` type that is interpreted as "latest stable Drupal version", so in late 2024, `drupal` means `drupal11`.

## `upload_dirs`

Paths from the project’s docroot to the user-generated files directory targeted by `ddev import-files`. Can be outside the docroot but must be within the project directory e.g. `../private`. Some CMSes and frameworks have default `upload_dirs`, like Drupal's `sites/default/files`; `upload_dirs` will override the defaults, so if you want Drupal to use both `sites/default/files` and `../private` you would list both, `upload_dirs: ["sites/default/files", "../private"]`. `upload_dirs` is used for targeting `ddev import-files` and also, when Mutagen is enabled, to bind-mount those directories so their contents does not need to be synced into Mutagen.

If you do not have directories of static assets of this type, or they are small and you don't care about them, you can disable the warning `You have Mutagen enabled and your 'php' project type doesn't have upload_dirs set.` by setting [`disable_upload_dirs_warning`](#disable_upload_dirs_warning) with `ddev config --disable-upload-dirs-warning`.

| Type | Default | Usage
| -- | -- | --
| :octicons-file-directory-16: project | | A list of directories.

## `use_dns_when_possible`

Whether to use DNS instead of editing `/etc/hosts`.

| Type | Default | Usage
| -- | -- | --
| :octicons-file-directory-16: project | `true` | Can be `true` or `false`.

When `false`, DDEV will always update the `/etc/hosts` file with the project hostname instead of using DNS for name resolution.

See [Using DDEV Offline](../usage/offline.md).

## `use_docker_compose_from_path`

Whether to use the system-installed docker-compose. You can otherwise use [`required_docker_compose_version`](#required_docker_compose_version) to specify a version for download.

| Type | Default | Usage
| -- | -- | --
| :octicons-globe-16: global | `false` | Can `true` or `false`.

When `true`, DDEV will use the docker-compose found in on your system’s path instead of using its private, known-good, docker-compose version.

!!!warning "Troubleshooting Only!"
    This should only be used in specific cases like troubleshooting. (It is used in the Docker Compose automated tests.)

## `use_hardened_images`

Whether to use hardened images for internet deployment.

| Type | Default | Usage
| -- | -- | --
| :octicons-globe-16: global | `false` | Can `true` or `false`.

When `true`, more secure hardened images are used for an internet deployment. These do not include sudo in the web container, and the container is run without elevated privileges. Generally used with the [hosting](../topics/hosting.md) feature.

## `use_letsencrypt`

Whether to enable Let’s Encrypt integration. (Works in conjunction with [`letsencrypt_email`](#letsencrypt_email).)

| Type | Default | Usage
| -- | -- | --
| :octicons-globe-16: global | `false` | Can `true` or `false`.

May also be set via `ddev config global --use-letsencrypt` or `ddev config global --use-letsencrypt=false`. When `true`, `letsencrypt_email` must also be set and the system must be available on the internet. Used with the [hosting](../topics/hosting.md) feature.

## `web_environment`

Additional [custom environment variables](../extend/customization-extendibility.md#environment-variables-for-containers-and-services) for a project’s web container. (Or for all projects if used globally.)

| Type | Default | Usage
| -- | -- | --
| :octicons-file-directory-16: project<br>:octicons-globe-16: global | `[]` | &zwnj;

## `web_extra_daemons`

Additional daemons that should [automatically be started in the web container](../extend/customization-extendibility.md#running-extra-daemons-in-the-web-container).

| Type | Default | Usage
| -- | -- | --
| :octicons-file-directory-16: project | `[]` | &zwnj;

## `web_extra_exposed_ports`

Additional named sets of ports to [expose via `ddev-router`](../extend/customization-extendibility.md#exposing-extra-ports-via-ddev-router).

| Type | Default | Usage
| -- | -- | --
| :octicons-file-directory-16: project | `[]` | &zwnj;

## `webimage`

The Docker image to use for the web server.

| Type | Default | Usage
| -- | -- | --
| :octicons-file-directory-16: project | [`ddev/ddev-webserver`](https://hub.docker.com/r/ddev/ddev-webserver) | Specify your own image based on [ddev/ddev-webserver](https://github.com/ddev/ddev/tree/main/containers/ddev-webserver).

!!!warning "Proceed with caution"
    It’s unusual to change this, and we don’t recommend it without Docker experience and a good reason. Typically, this means additions to the existing web image using a `.ddev/web-build/Dockerfile.*`.

## `webimage_extra_packages`

Extra Debian packages for the project’s web container.

| Type | Default | Usage
| -- | -- | --
| :octicons-file-directory-16: project | `[]` | &zwnj;

Example: `webimage_extra_packages: [php-yaml, php-bcmath]` will add the `php-yaml` and `php-bcmath` packages when the web container is built.

## `webserver_type`

Which available [web server type](../extend/customization-extendibility.md#changing-web-server-type) should be used.

| Type | Default | Usage
| -- | -- | --
| :octicons-file-directory-16: project | `nginx-fpm` | Can be `nginx-fpm` or `apache-fpm`.

To change from the default `nginx-fpm` to `apache-fpm`, for example, you would need to edit your project’s `.ddev/config.yaml` to include the following:

```yaml
webserver_type: apache-fpm
```

Then run the [`ddev restart`](../usage/commands.md#restart) command to have the change take effect.

## `working_dir`

Working directories used by [`ddev exec`](../usage/commands.md#exec) and [`ddev ssh`](../usage/commands.md#ssh).

| Type | Default | Usage
| -- | -- | --
| :octicons-file-directory-16: project | &zwnj; | &zwnj;

Example: `working_dir: { web: "/var/www", db: "/etc" }` sets the working directories for the `web` and `db` containers.

## `wsl2_no_windows_hosts_mgt`

(WSL2 only) Whether to disable the management and checking of the Windows hosts file. By default, when using WSL2, DDEV manages the system-wide hosts file on the Windows side (normally `C:\Windows\system32\drivers\etc\hosts`) by using `ddev.exe` installed on the *Windows* side. This normally works better for all applications, including browsers and IDEs. However, this behavior can be disabled by setting `wsl_no_windows_hosts_mgt: true`.

| Type | Default | Usage
| -- | -- | --
| :octicons-globe-16: global | `false` | Can `true` or `false`.

May also be set via `ddev config global --wsl2-no-windows-hosts-mgt` or `ddev config global --wsl2-no-windows-hosts-mgt=false`.

## `xdebug_enabled`

Whether Xdebug should be enabled for [step debugging](../debugging-profiling/step-debugging.md) or [profiling](../debugging-profiling/xdebug-profiling.md).

| Type | Default | Usage
| -- | -- | --
| :octicons-file-directory-16: project | `false` | Please leave this `false` in most cases. Most people use [`ddev xdebug`](../usage/commands.md#xdebug) and `ddev xdebug off` (or `ddev xdebug toggle`) commands.

## `xdebug_ide_location`

Adjust Xdebug listen location for WSL2 or in-container. This is used very rarely. Ask for help in one of our [support channels](../support.md) before changing it unless you understand its use completely.

| Type | Default | Usage
| -- | -- | --
| :octicons-globe-16: global | `""` | Can be empty (`""`), `"wsl2"`, `"container"`, or an explicit IP address.

For PhpStorm running inside WSL2 (or JetBrains Gateway), use `"wsl2"`. For in-container like VS Code Language Server, set to `"container"`. It can also be set to an explicit IP address.

Examples:

* `xdebug_ide_location: 172.16.0.2` when you need to provide an explicit IP address where the IDE is listening. This is very unusual.
* `xdebug_ide_location: container` when the IDE is actually listening inside the `ddev-webserver` container. This is only done very occasionally with obscure Visual Studio Code setups like VS Code Language Server.
* `xdebug_ide_location: wsl2` when an IDE is running (or listening) in WSL2. This is the situation when running an IDE directly inside WSL2 instead of running it on Windows.
