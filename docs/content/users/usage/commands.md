# Commands

You can tell DDEV what to do by running its commands. This page details each of the available commands and their options, or flags.

Run DDEV without any commands or flags to see this list in your terminal:

```
→  ddev
Create and maintain a local web development environment.
Docs: https://ddev.readthedocs.io
Support: https://ddev.readthedocs.io/en/stable/users/support

Usage:
  ddev [command]

Available Commands:
  auth             A collection of authentication commands
  blackfire        Enable or disable blackfire.io profiling (global shell web container command)
  clean            Removes items ddev has created
  composer         Executes a composer command within the web container
...
```

Use [`ddev help`](#help) to learn more about a specific command, like this example for [`ddev describe`](#describe):

```
→  ddev help describe
Get a detailed description of a running ddev project. Describe provides basic
information about a ddev project, including its name, location, url, and status.
It also provides details for MySQL connections, and connection information for
additional services like Mailpit. You can run 'ddev describe' from
a project directory to describe that project, or you can specify a project to describe by
running 'ddev describe <projectname>'.

Usage:
  ddev describe [projectname] [flags]

Aliases:
  describe, status, st, desc

Examples:
ddev describe
ddev describe <projectname>
ddev status
ddev st

Flags:
  -h, --help   help for describe

Global Flags:
  -j, --json-output   If true, user-oriented output will be in JSON format.
```

## Global Flags

Two flags are available for every command:

* `--help` or `-h`: Outputs more information about a command rather than executing it.
* `--json-output` or `-j`: Format user-oriented output in JSON.

---

## `auth`

Authentication commands.

### `auth ssh`

Add [SSH key authentication](../usage/cli.md#ssh-into-containers) to the `ddev-ssh-agent` container.

Example:

```shell
# Add your SSH keys to the SSH agent container
ddev auth ssh
```

Flags:

* `--ssh-key-path`, `-d`: Full path to SSH key directory.

## `artisan`

Run the `artisan` command; available only in projects of type `laravel`, and only available if `artisan` is in the project root.

```shell
# Show all artisan subcommands
ddev artisan list
```

## `blackfire`

Enable or disable [Blackfire profiling](../debugging-profiling/blackfire-profiling.md) (global shell web container command).

```shell
# Display Blackfire profiling status
ddev blackfire status

# Start Blackfire profiling
ddev blackfire on

# Stop Blackfire profiling
ddev blackfire off
```

!!!tip
    There are synonyms for the `on` and `off` arguments that have the exact same effect:

    * `on`: `start`, `enable`, `true`
    * `off`: `stop`, `disable`, `false`

## `cake`

Run the `cake` command; available only in projects of type `cakephp`, and only available if `cake.php` is in bin folder.

```shell
# Show all cake subcommands
ddev cake
```

## `clean`

Removes items DDEV has created. (See [Uninstalling DDEV](../usage/uninstall.md).)

Flags:

* `--all`, `-a`: Clean all DDEV projects.
* `--dry-run`: Run the clean command without deleting.

Example:

```shell
# Preview cleaning all projects without actually removing anything
ddev clean --dry-run --all

# Clean all projects
ddev clean --all

# Clean my-project and my-other-project
ddev clean my-project my-other-project
```

## `composer`

Executes a [Composer command](../usage/developer-tools.md#ddev-and-composer) within the web container.

`ddev composer create` is a special command that is an adaptation of `composer create-project`. See [DDEV and Composer](../usage/developer-tools.md#ddev-and-composer) for more information.

Example:

```shell
# Install Composer packages
ddev composer install
```

Example of `ddev composer create`:

```shell
# Create a new Drupal project in the current directory
ddev composer create drupal/recommended-project
```

## `config`

Create or modify a DDEV project’s configuration in the current directory. By default, `ddev config` will not change configuration that already exists in your `.ddev/config.yaml`, it will only make changes you specify with flags. However, if you want to autodetect everything, `ddev config --update` will usually do everything you need.

!!!tip "You can also set these via YAML!"
    These settings, plus a few more, can be set by editing stored [Config Options](../configuration/config.md).

Example:

```shell
# Start interactive project configuration
ddev config

# Accept defaults on a new project. This is the same as hitting <RETURN>
# on every question in `ddev config`
ddev config --auto

## Detect docroot, project type, and expected defaults for an existing project
ddev config --update

# Configure a Drupal project with a `web` document root
ddev config --docroot=web --project-type=drupal

# Switch the project’s default `nginx-fpm` to `apache-fpm`
ddev config --webserver-type=apache-fpm
```

Flags:

* `--additional-fqdns`: Comma-delimited list of project FQDNs.
* `--additional-hostnames`: Comma-delimited list of project hostnames.
* `--auto`: Automatically run config without prompting.
* `--bind-all-interfaces`: Bind host ports on all interfaces, not only on localhost network interface.
* `--composer-root`: Overrides the default Composer root directory for the web service.
* `--composer-root-default`: Unsets a web service Composer root directory override.
* `--composer-version`: Specify override for Composer version in the web container. This may be `""`, `"1"`, `"2"`, `"2.2"`, `"stable"`, `"preview"`, `"snapshot"`, or a specific version.
* `--database`: Specify the database type:version to use. Defaults to `mariadb:10.11`.
* `--db-image`: Sets the db container image.
* `--db-image-default`: Sets the default db container image for this DDEV version.
* `--db-working-dir`: Overrides the default working directory for the db service.
* `--db-working-dir-default`: Unsets a db service working directory override.
* `--dbimage-extra-packages`: A comma-delimited list of Debian packages that should be added to db container when the project is started.
* `--default-container-timeout`: Default time in seconds that DDEV waits for all containers to become ready on start. (default `120`)
* `--disable-settings-management`: Prevent DDEV from creating or updating CMS settings files.
* `--disable-upload-dirs-warning`: Suppresses warning when a project is using `performance_mode: mutagen` but does not have `upload_dirs` set.
* `--docroot`: Provide the relative docroot of the project, like `docroot` or `htdocs` or `web`. (defaults to empty, the current directory)
* `--fail-on-hook-fail`: Decide whether `ddev start` should be interrupted by a failing hook.
* `--host-db-port`: The db container’s localhost-bound port.
* `--host-https-port`: The web container’s localhost-bound HTTPS port.
* `--host-webserver-port`: The web container’s localhost-bound port.
* `--http-port`: The router HTTP port for this project.
* `--https-port`: The router HTTPS port for this project.
* `--image-defaults`: Sets the default web and db container images.
* `--mailpit-http-port`: Router port to be used for Mailpit HTTP access.
* `--mailpit-https-port`: Router port to be used for Mailpit HTTPS access.
* `--ngrok-args`: Provide extra args to ngrok in `ddev share`.
* `--no-project-mount`: Whether or not to skip mounting project code into the web container.
* `--nodejs-version`: Specify the Node.js version to use if you don’t want the default version.
* `--omit-containers`: Comma-delimited list of container types that should not be started when the project is started.
* `--performance-mode`: Performance optimization mode, possible values are `global`, `none`, `mutagen`, `nfs`.
* `--performance-mode-reset`: Reset performance mode to global configuration.
* `--php-version`: PHP version that will be enabled in the web container.
* `--project-name`: Provide the project name of project to configure. (normally the same as the last part of directory name)
* `--project-tld`: Set the top-level domain to be used for projects. (default `"ddev.site"`)
* `--project-type`: Provide the project type: `backdrop`, `drupal`, `drupal6`, `drupal7`, `laravel`, `magento`, `magento2`, `php`, `shopware6`, `silverstripe`, `typo3`, `wordpress`. This is autodetected and this flag is necessary only to override the detection.
* `--show-config-location`: Output the location of the `config.yaml` file if it exists, or error that it doesn’t exist.
* `--timezone`: Specify timezone for containers and PHP, like `Europe/London` or `America/Denver` or `GMT` or `UTC`.
* `--update`: Automatically detect and update settings by inspecting the code.
* `--upload-dirs`: Sets the project’s upload directories, the destination directories of the import-files command.
* `--use-dns-when-possible`: Use DNS for hostname resolution instead of `/etc/hosts` when possible. (default `true`)
* `--web-environment`: Set the environment variables in the web container: `--web-environment="TYPO3_CONTEXT=Development,SOMEENV=someval"`
* `--web-environment-add`: Append environment variables to the web container: `--web-environment="TYPO3_CONTEXT=Development,SOMEENV=someval"`
* `--web-image`: Sets the web container image.
* `--web-image-default`: Sets the default web container image for this DDEV version.
* `--web-working-dir`: Overrides the default working directory for the web service.
* `--web-working-dir-default`: Unsets a web service working directory override.
* `--webimage-extra-packages`: A comma-delimited list of Debian packages that should be added to web container when the project is started.
* `--webserver-type`: Sets the project’s desired web server type: `nginx-fpm`, `nginx-gunicorn`, or `apache-fpm`.
* `--working-dir-defaults`: Unsets all service working directory overrides.
* `--xdebug-enabled`: Whether or not Xdebug is enabled in the web container.

### `config global`

Change global configuration.

```shell
# Opt out of sharing anonymized usage information
ddev config global --instrumentation-opt-in=false

# Skip the SSH agent for all projects
ddev config global --omit-containers=ddev-ssh-agent
```

* `--disable-http2`: Optionally disable http2 in `ddev-router`; `ddev config global --disable-http2` or `ddev config global --disable-http2=false`. This option is not available in the current Traefik-based `ddev-router`, but only in the deprecated `nginx-proxy` router.
* `--fail-on-hook-fail`: If true, `ddev start` will fail when a hook fails.
* `--instrumentation-opt-in`: `instrumentation-opt-in=true`.
* `--internet-detection-timeout-ms`: Increase timeout when checking internet timeout, in milliseconds. (default `3000`)
* `--letsencrypt-email`: Email associated with Let’s Encrypt; `ddev global --letsencrypt-email=me@example.com`.
* `--mailpit-http-port`: The Mailpit HTTP port *default* for all projects; can be overridden by project configuration.
* `--mailpit-https-port`: The Mailpit HTTPS port *default* for all projects; can be overridden by project configuration.
* `--no-bind-mounts`: If `true`, don’t use bind-mounts. Useful for environments like remote Docker where bind-mounts are impossible. (default is equal to `--no-bind-mounts=true`)
* `--omit-containers`: For example, `--omit-containers=ddev-ssh-agent`.
* `--performance-mode`: Performance optimization mode, possible values are `none`, `mutagen`, `nfs`.
* `--performance-mode-reset`: Reset performance optimization mode to operating system default (`none` for Linux and WSL2, `mutagen` for macOS and traditional Windows).
* `--project-tld`: Set the default top-level domain to be used for all projects. (default `"ddev.site"`). Note that this will be overridden in a project that defines `project_tld`.
* `--router-http-port`: The router HTTP port *default* for all projects; can be overridden by project configuration.
* `--router-https-port`: The router HTTPS port *default* for all projects; can be overridden by project configuration.
* `--simple-formatting`: If `true`, use simple formatting and no color for tables.
* `--table-style`: Table style for list and describe, see `~/.ddev/global_config.yaml` for values.
* `--traefik-monitor-port`: Can be used to change the Traefik monitor port in case of port conflicts, for example `ddev config global --traefik-monitor-port=11999`.
* `--use-hardened-images`: If `true`, use more secure 'hardened' images for an actual internet deployment.
* `--use-letsencrypt`: Enables experimental Let’s Encrypt integration; `ddev global --use-letsencrypt` or `ddev global --use-letsencrypt=false`.
* `--web-environment`: Set the environment variables in the web container: `--web-environment="TYPO3_CONTEXT=Development,SOMEENV=someval"`
* `--web-environment-add`: Append environment variables to the web container: `--web-environment="TYPO3_CONTEXT=Development,SOMEENV=someval"`

## `craft`

Run a [Craft CMS command](https://craftcms.com/docs/4.x/console-commands.html) inside the web container (global shell web container command).

Example:

```shell
# Run pending Craft migrations and apply pending project config changes
ddev craft up
```

## `dbeaver`

Open [DBeaver](https://dbeaver.io/) with the current project’s database (global shell host container command). This command is only available if `DBeaver.app` is installed as `/Applications/DBeaver.app` for macOS, if `dbeaver.exe` is installed to all users as `C:/Program Files/dbeaver/dbeaver.exe` for WSL2, and if `dbeaver` (or another binary like `dbeaver-ce`) available inside `/usr/bin` for Linux (Flatpak and snap support included).

Example:

```shell
# Open the current project’s database in DBeaver
ddev dbeaver
```

## `debug`

*Aliases: `d`, `dbg`.*

A collection of debugging commands, often useful for [troubleshooting](troubleshooting.md).

### `debug capabilities`

Show capabilities of this version of DDEV.

Example:

```shell
# List capabilities of the current project
ddev debug capabilities

# List capabilities of `my-project`
ddev debug capabilities my-project
```

### `debug check-db-match`

Verify that the database in the db server matches the configured [type and version](../extend/database-types.md).

Example:

```shell
# Check whether project’s running database matches configuration
ddev debug check-db-match
```

### `debug compose-config`

Prints the current project’s docker-compose configuration.

Example:

```shell
# Print docker-compose config for the current project
ddev debug compose-config

# Print docker-compose config for `my-project`
ddev debug compose-config my-project
```

### `debug configyaml`

Prints the project [`config.*.yaml`](../configuration/config.md) usage.

Example:

```shell
# Print config for the current project
ddev debug configyaml

# Print config specifically for `my-project`
ddev debug configyaml my-project
```

### `debug dockercheck`

Diagnose DDEV Docker/Colima setup.

Example:

```shell
# Output contextual details for the Docker provider
ddev debug dockercheck
```

### `debug download-images`

Download the basic Docker images required by DDEV. This can be useful on a new machine to prevent `ddev start` or other commands having to download the various images.

Example:

```shell
# Download DDEV’s basic Docker images
ddev debug download-images
```

### `debug fix-commands`

Refreshes [custom command](../extend/custom-commands.md) definitions without running [`ddev start`](#start).

Example:

```shell
# Refresh the current project’s custom commands
ddev debug fix-commands
```

### `debug get-volume-db-version`

Get the database type and version found in the `ddev-dbserver` database volume, which may not be the same as the configured database [type and version](../extend/database-types.md).

Example:

```shell
# Print the database volume’s engine and version
ddev debug get-volume-db-version
```

### `debug migrate-database`

Migrate a MySQL or MariaDB database to a different `dbtype:dbversion`. Works only with MySQL and MariaDB, not with PostgreSQL. It will export your database, create a snapshot, destroy your current database, and import into the new database type. It only migrates the 'db' database. It will update the database version in your project's `config.yaml` file.

Example:

```shell
# Migrate the current project’s database to MariaDB 10.7
ddev debug migrate-database mariadb:10.7
```

### `debug mutagen`

Allows access to any [Mutagen command](https://mutagen.io/documentation/introduction).

Example:

```shell
# Run Mutagen’s `sync list` command
ddev debug mutagen sync list
```

### `debug nfsmount`

Checks to see if [NFS mounting](../install/performance.md#nfs) works for current project.

Example:

```shell
# See if NFS is working as expected for the current project
ddev debug nfsmount
```

### `debug refresh`

Refreshes the project’s Docker cache.

Example:

```shell
# Refresh the current project’s Docker cache
ddev debug refresh
```

### `debug router-nginx-config`

Prints the router’s [nginx config](../extend/customization-extendibility.md#custom-nginx-configuration).

Example:

```shell
# Output router nginx configuration
ddev debug router-nginx-config
```

### `debug test`

Run diagnostics using the embedded [test script](https://github.com/ddev/ddev/blob/master/cmd/ddev/cmd/scripts/test_ddev.sh).

Example:

```shell
# Run DDEV’s diagnostic suite
ddev debug test
```

### `debug testcleanup`

Removes all diagnostic projects created with `ddev debug test`.

Example:

```shell
# Remove all DDEV’s diagnostic projects
ddev debug testcleanup
```

## `delete`

Remove all information, including the database, for an existing project.

Flags:

* `--all`, `-a`: Delete all projects.
* `--clean-containers`: Clean up all DDEV Docker containers not required by this version of DDEV. (default `true`)
* `--omit-snapshot`, `-O`: Omit/skip database snapshot.
* `--yes`, `-y`: Skip confirmation prompt.

Example:

```shell
# Delete my-project and my-other-project
ddev delete my-project my-other-project

# Delete the current project without taking a snapshot or confirming
ddev delete --omit-snapshot --yes
```

### `delete images`

With `--all`, it deletes all `ddev/ddev-*` Docker images.

Flags:

* `--all`, `-a`: If set, deletes all Docker images created by DDEV.
* `--yes`, `-y`: Skip confirmation prompt.

Example:

```shell
# Delete images
ddev delete images

# Delete images and skip confirmation
ddev delete images -y

# Delete all DDEV-created images
ddev delete images --all
```

## `describe`

*Aliases: `status`, `st`, `desc`.*

Get a detailed description of a running DDEV project.

Example:

```shell
# Display details for the current project
ddev describe

# Display details for my-project
ddev describe my-project
```

## `drush`

Run the `drush` command; available only in projects of type `drupal*`, and only available if `drush` is in the project. On projects of type `drupal`, `drush` should be installed in the project itself, (`ddev composer require drush/drush`). On projects of type `drupal7` `drush` 8 is provided by DDEV.

```shell
# Show drush status/configuration
ddev drush st
```

## `exec`

*Alias: `.`.*

[Execute a shell command in the container](../usage/cli.md#executing-commands-in-containers) for a service. Uses the web service by default.

To run your command in a different service container, run `ddev exec --service <service> <cmd>`. Use the `--raw` flag if you’d like to run a raw, uninterpreted command in a container.

Flags:

* `--dir`, `-d`: Defines the execution directory within the container.
* `--raw`: Use raw exec (do not interpret with Bash inside container). (default `true`)
* `--service`, `-s`: Defines the service to connect to. (e.g. `web`, `db`) (default `"web"`)

Example:

```shell
# List the web container’s docroot contents
ddev exec ls /var/www/html

# List the web container’s vendor directory contents
ddev exec --dir /var/www/html/vendor ls

# Output a long, recursive list of the files in the web container
ddev exec --raw -- ls -lR
```

## `export-db`

Dump a database to a file or to stdout.

Flags:

* `--bzip2`: Use bzip2 compression.
* `--database`, `-d`: Target database to export from (default `"db"`)
* `--file`, `-f`: Path to a SQL dump file to export to
* `--gzip`: Use gzip compression (default `true`)
* `--xz`: Use xz compression.

Example:

```shell
# Dump and compress the current project’s database to `/tmp/db.sql.gz`
ddev export-db --file=/tmp/db.sql.gz

# Dump the current project’s database, without compressing it, to `/tmp/db.sql`
ddev export-db --gzip=false --file /tmp/db.sql

# Dump and compress the current project’s `foo` database instead of `db`
ddev export-db --database=foo --file=/tmp/db.sql.gz

# Output the current project’s database and use `>` to write to `/tmp/db.sql.gz`
ddev export-db > /tmp/db.sql.gz

# Dump my-project’s database, without compressing it, to `/tmp/my-project.sql`
ddev export-db my-project --gzip=false --file=/tmp/my-project.sql
```

## `get`

Download an [add-on](../extend/additional-services.md) (service, provider, etc.).

Flags:

* `--all`: List unofficial *and* official add-ons. (default `true`)
* `--list`: List official add-ons. (default `true`)
* `--installed`: List installed add-ons
* `--remove <add-on>`: Remove an installed add-on
* `--version <version>`: Specify a version to download
* `--verbose`, `-v`: Output verbose error information with Bash `set -x` (default `false`)

Environment variables:

* `DDEV_GITHUB_TOKEN`: A [GitHub token](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/creating-a-personal-access-token) may be used for `ddev get` requests (which result in GitHub API queries). It's unusual for casual users to need this, but if you're doing lots of `ddev get` requests you may run into rate limiting. The token you use requires no privileges at all. Example:

```bash
export DDEV_GITHUB_TOKEN=<your github token>
ddev get --list --all
```

Example:

```shell
# List official add-ons
ddev get --list

# List official and third-party add-ons
ddev get --list --all

# Download the official Redis add-on
ddev get ddev/ddev-redis

# Get debug info about `ddev get` failure
ddev get ddev/ddev-redis --verbose

# Download the official Redis add-on, version v1.0.4
ddev get ddev/ddev-redis --version v1.0.4

# Download the Drupal Solr add-on from its v1.2.3 release tarball
ddev get https://github.com/ddev/ddev-drupal-solr/archive/refs/tags/v1.2.3.tar.gz

# Copy an add-on available in another directory
ddev get /path/to/package

# Copy an add-on from a tarball in another directory
ddev get /path/to/tarball.tar.gz

# View installed add-ons
ddev get --installed

# Remove an add-on can be done with the full name, the short name of repo
# or with owner/repo format
ddev get --remove redis
ddev get --remove ddev-redis
ddev get --remove ddev/ddev-redis
```

In general, you can run `ddev get` multiple times without doing any damage. Updating an add-on can be done by running `ddev get <add-on-name>`. If you have changed an add-on file and removed the `#ddev-generated` marker in the file, that file will not be touched and DDEV will let you know about it.

## `heidisql`

Open [HeidiSQL](https://www.heidisql.com/) with the current project’s database (global shell host container command). This command is only available if `Heidisql.exe` is installed as `C:\Program Files\HeidiSQL\Heidisql.exe`.

Example:

```shell
# Open the current project’s database in HeidiSQL
ddev heidisql
```

## `help`

Help about any command.

Example:

```shell
# Illuminate the virtues of the `describe` command
ddev help describe
```

## `hostname`

Manage your hostfile entries.

Flags:

* `--remove`, `-r`: Remove the provided hostname - ip correlation.
* `--remove-inactive`, `-R`: Remove hostnames of inactive projects.

Example:

```shell
ddev hostname somesite.ddev.local 127.0.0.1
```

## `import-db`

[Import a SQL file](database-management.md) into the project.

Flags:

* `--database`, `-d`: Target database to import into (default `"db"`)
* `--extract-path`: Path to extract within the archive
* `--file`, `-f`: Path to a SQL dump in `.sql`, `.tar`, `.tar.gz`, `.tar.bz2`, `.tar.xz`, `.tgz`, or `.zip` format
* `--no-drop`: Do not drop the database before importing
* `--no-progress`: Do not output progress

Example:

```shell
# Start the interactive import utility
ddev import-db

# Import the `.tarballs/db.sql` dump to the project database
ddev import-db --file=.tarballs/db.sql

# Import the compressed `.tarballs/db.sql.gz` dump to the project database
ddev import-db --file=.tarballs/db.sql.gz

# Import the compressed `.tarballs/db.sql.gz` dump to a `other_db` database
ddev import-db --database=additional_db --file=.tarballs/db.sql.gz

# Import the `db.sql` dump to the project database
ddev import-db < db.sql

# Import the `db.sql` dump to the `my-project` default database
ddev import-db my-project < db.sql

# Uncompress `db.sql.gz` and pipe the result to the `import-db` command
gzip -dc db.sql.gz | ddev import-db
```

## `import-files`

Pull the uploaded files directory of an existing project to the default public upload directory of your project. More usage information and a description of the Tar or ZIP archive is in the [usage section](../usage/cli.md#ddev-import-files).

Flags:

* `--extract-path`: Path to extract within the archive.
* `--source`, `-s`: Path to the source directory or source archive in `.tar`, `.tar.gz`, `.tar.bz2`, `.tar.xz`, `.tgz`, or `.zip` format.
* `--target`, `-t`: Target upload dir, defaults to the first upload dir.

Example:

```shell
# Extract+import `/path/to/files.tar.gz` to the project’s first upload directory
ddev import-files --source=/path/to/files.tar.gz

# Import `/path/to/dir` contents to the project’s first upload directory
ddev import-files --source=/path/to/dir

# Import `.tarballs/files.tar.xz` contents to the project’s `../private` upload directory
ddev import-files --source=.tarballs/files.tar.xz --target=../private

# Import `/path/to/dir` contents to the project’s `sites/default/files` upload directory
ddev import-files -s=.tarballs/files.tar.gz -t=sites/default/files
```

## `launch`

Launch a browser with the current site (global shell host container command).

Flags:

* `--mailpit`, `-m`: Open Mailpit.

!!!tip "How to disable HTTP redirect to HTTPS?"
    Recommendations for:

    * [Google Chrome](https://stackoverflow.com/q/73875589)
    * [Mozilla Firefox](https://stackoverflow.com/q/30532471)
    * [Safari](https://stackoverflow.com/q/46394682)

Example:

```shell
# Open your project’s base URL in the default browser
ddev launch

# Open Mailpit in the default browser
ddev launch --mailpit

# Open your project’s base URL appended with `temp/phpinfo.php`
ddev launch temp/phpinfo.php

# Open the full URL (any website) in the default browser
ddev launch https://your.ddev.site

# Open your project’s base URL using a specific port
ddev launch :3000
```

## `list`

*Aliases: `l`, `ls`.*

List projects.

Flags:

* `--active-only`, `-A`: If set, only currently active projects will be displayed.
* `--continuous`: If set, project information will be emitted until the command is stopped.
* `--continuous-sleep-interval`, `-I`: Time in seconds between `ddev list --continuous` output lists. (default `1`)
* `--type`, `-t`: Show only projects of this type (e.g. `drupal`, `wordpress`, `php`).
* `--wrap-table`, `-W`: Display table with wrapped text if required.

Example:

```shell
# List all projects
ddev list

# List all running projects
ddev list --active-only

# List all WordPress projects
ddev list --type wordpress
```

## `logs`

Get the logs from your running services.

Flags:

* `--follow`, `-f`: Follow the logs in real time.
* `--service`, `-s`: Defines the service to retrieve logs from (e.g. `web`, `db`). (default `"web"`)
* `--tail`: How many lines to show.
* `--time`, `-t`: Add timestamps to logs.

Example:

```shell
# Display recent logs from the current project’s web server
ddev logs

# Stream logs from the current project’s web server in real time
ddev logs -f

# Display recent logs from the current project’s database server
ddev logs -s db

# Display recent logs from my-project’s database server
ddev logs -s db my-project
```

## `mailpit`

Launch a browser with mailpit for the current project (global shell host container command).

Example:

```shell
# Open Mailpit in the default browser
ddev mailpit
```

## `magento`

Run the `magento` command; available only in projects of type `magento2`, and only works if `bin/magento` is in the project.

```shell
# Show all magento subcommands
ddev magento list
```

## `mutagen`

Commands for [Mutagen](../install/performance.md#mutagen) status and sync, etc.

### `mutagen logs`

Show Mutagen logs for debugging.

Flags:

* `--verbose`: Show full Mutagen logs.

Example:

```shell
# Stream Mutagen’s logs in real time
ddev mutagen logs

# Stream Mutagen’s more detailed logs in real time
ddev mutagen logs --verbose
```

### `mutagen monitor`

Monitor Mutagen status.

Example:

```shell
# Start Mutagen’s sync process and monitor its status in real time
ddev mutagen sync && ddev mutagen monitor
```

### `mutagen reset`

Stops a project and removes the Mutagen Docker volume.

```shell
# Reset Mutagen data for the current project
ddev mutagen reset

# Reset Mutagen data for my-project
ddev mutagen reset my-project
```

### `mutagen status`

Shows Mutagen sync status.

Flags:

* `--verbose`, `-l`: Extended/verbose output for Mutagen status.

Example:

```shell
# Display Mutagen sync status for the current project
ddev mutagen status

# Display Mutagen sync status for my-project
ddev mutagen status my-project
```

### `mutagen sync`

Explicit sync for Mutagen.

Flags:

* `--verbose`: Extended/verbose output for Mutagen status.

Example:

```shell
# Initiate Mutagen sync for the current project
ddev mutagen sync

# Initiate Mutagen sync for my-project
ddev mutagen sync my-project
```

### `mutagen version`

Display the version of the Mutagen binary and the location of its components.

Example:

```shell
# Print Mutagen details
ddev mutagen version
```

## `mysql`

Run MySQL client in the database container (global shell db container command). This is only available on projects that use the `mysql` or `mariadb` database types.

Example:

```shell
# Run the database container’s MySQL client
ddev mysql

# Run the database container’s MySQL client as root user
ddev mysql -uroot -proot

# Pipe the `SHOW TABLES;` command to the MySQL client to see a list of tables
echo 'SHOW TABLES;' | ddev mysql
```

## `npm`

Run [`npm`](https://docs.npmjs.com/cli/v9/commands/npm) inside the web container (global shell web container command).

Example:

```shell
# Install JavaScript packages using `npm`
ddev npm install

# Update JavaScript packages using `npm`
ddev npm update
```

## `nvm`

Run [`nvm`](https://github.com/nvm-sh/nvm#usage) inside the web container (global shell web container command).

!!!tip
    Use of `ddev nvm` is discouraged because `nodejs_version` is much easier to use, can specify any version, and is more robust than using `nvm`.

Example:

```shell
# Use `nvm` to switch to Node.js v20
ddev nvm install 20

# Check the installed Node.js version
ddev nvm current

# Reset Node.js to `nodejs_version`
ddev nvm alias default system

# Switch between two installed Node.js versions
ddev nvm install 20
ddev nvm install 18
ddev nvm alias default 20
ddev nvm alias default 18
```

!!!warning "`nvm use` works only inside the web container after `ddev ssh`"
    Use `ddev nvm alias default <version>` instead.

## `php`

Run `php` inside the web container (global shell web container command).

Example:

```shell
# Output the web container’s PHP version
ddev php --version
```

## `poweroff`

*Alias: `powerdown`.*

Completely stop all projects and containers.

!!!tip
    This is the equivalent of running `ddev stop -a --stop-ssh-agent`.

Example:

```shell
# Stop all projects and containers
ddev poweroff
```

## `psql`

Run PostgreSQL client in the database container (global shell db container command). This is only available on projects that use the `postgres` database type.

Example:

```shell
# List available databases
ddev psql -l

# List tables in the default 'db' database
echo "\dt;" | ddev psql

```

## `pull`

Pull files and database using a configured [provider plugin](./../providers/index.md).

Flags:

* `--environment=ENV1=val1,ENV2=val2`
* `--skip-confirmation`, `-y`: Skip confirmation step.
* `--skip-db`: Skip pulling database archive.
* `--skip-files`: Skip pulling file archive.
* `--skip-import`: Download archive(s) without importing than.

Example:

```shell
# Pull a backup from the configured Pantheon project to use locally
ddev pull pantheon

# Pull a backup from the configured Platform.sh project to use locally
ddev pull platform

# Pull a backup from the configured Pantheon project without confirming
ddev pull pantheon -y

# Pull the Platform.sh database archive *only* without confirming
ddev pull platform --skip-files -y

# Pull the localfile integration’s files *only* without confirming
ddev pull localfile --skip-db -y

# Pull from Platform.sh specifying the environment variables PLATFORM_ENVIRONMENT and PLATFORM_CLI_TOKEN on the command line
ddev pull platform --environment=PLATFORM_ENVIRONMENT=main,PLATFORMSH_CLI_TOKEN=abcdef
```

## `push`

Push files and database using a configured [provider plugin](./../providers/index.md).

Example:

```shell
# Push local files and database to the configured Pantheon project
ddev push pantheon

# Push local files and database to the configured Platform.sh project
ddev push platform

# Push files and database to Pantheon without confirming
ddev push pantheon -y

# Push database only to Platform.sh without confirming
ddev push platform --skip-files -y

# Push files only to Acquia without confirming
ddev push acquia --skip-db -y
```

## `python`

Runs `python` inside the web container in the same relative directory you're in on the host.

`ddev python` is only available on Python-based project types like Django and Python.

Example:

```shell
# Run manage.py
ddev python manage.py migrate
```

## `querious`

Open [Querious](https://www.araelium.com/querious) with the current project’s MariaDB or MySQL database (global shell host container command). This is only available if `Querious.app` is installed as `/Applications/Querious.app`, and only for projects with `mysql` or `mariadb` databases.

Example:

```shell
# Open the current project’s database in Querious
ddev querious
```

## `restart`

Restart one or several projects.

Flags:

* `--all`, `-a`: Restart all projects.

Example:

```shell
# Restart the current project
ddev restart

# Restart my-project and my-other-project
ddev restart my-project my-other-project

# Restart all projects
ddev restart --all
```

## `sake`

Run the `sake` command, only available for Silverstripe projects and if the Silverstripe `sake` command is
available in the `vendor/bin` folder.

Common commands:

* Build database: `ddev sake dev/build`
* List of available tasks: `ddev sake dev/tasks`

## `self-upgrade`

Output instructions for updating or upgrading DDEV. The command doesn’t perform the upgrade, but tries to provide instructions relevant to your installation.

Example:

```
→  ddev self-upgrade

DDEV appears to have been installed with install_ddev.sh, you can run that script again to update.
curl -fsSL https://ddev.com/install.sh | bash
```

## `sequelace`

Open [SequelAce](https://sequel-ace.com/) with the current project’s database (global shell host container command). This command is only available if `Sequel Ace.app` is installed as `/Applications/Sequel ace.app`, and only for projects with `mysql` or `mariadb` databases.

Example:

```shell
# Open the current project’s database in SequelAce
ddev sequelace
```

## `sequelpro`

!!!warning "Sequel Pro is abandoned!"
    The project is abandoned and doesn’t work with MySQL 8. We recommend Sequel Ace, Querious, TablePlus, and DBeaver.

Open Sequel Pro with the current project’s database (global shell host container command). This command is only available if `Sequel Pro.app` is installed as `/Applications/Sequel pro.app`, and only for projects with `mysql` or `mariadb` databases.

Example:

```shell
# Open the current project’s database in Sequel Pro
ddev sequelpro
```

## `service`

Add or remove, enable or disable [extra services](../extend/additional-services.md).

### `service disable`

Disable a service.

Example:

```shell
# Disable the Solr service
ddev service disable solr
```

### `service enable`

Enable a service.

Example:

```shell
# Enable the Solr service
ddev service enable solr
```

## `share`

[Share the current project](../topics/sharing.md) on the internet via [ngrok](https://ngrok.com).

!!!tip
    Any ngrok flag can also be specified in the [`ngrok_args` config setting](../configuration/config.md#ngrok_args).

Flags:

* `--ngrok-args`: Accepts any flag from `ngrok http --help`.

Example:

```shell
# Share the current project with ngrok
ddev share

# Share the current project with ngrok, using domain `foo.ngrok-free.app`
ddev share --ngrok-args "--domain foo.ngrok-free.app"

# Share the current project using ngrok’s basic-auth argument
ddev share --ngrok-args "--basic-auth username:pass1234"

# Share my-project with ngrok
ddev share my-project
```

## `snapshot`

Create a database snapshot for one or more projects.

This uses `xtrabackup` or `mariabackup` to create a database snapshot in the `.ddev/db_snapshots` directory. These are compatible with server backups using the same tools and can be restored with the [`snapshot restore`](#snapshot-restore) command.

See [Snapshotting and Restoring a Database](../usage/cli.md#snapshotting-and-restoring-a-database) for more detail, or [Database Management](../usage/database-management.md) for more on working with databases in general.

Flags:

* `--all`, `-a`: Snapshot all projects. (Will start stopped or paused projects.)
* `--cleanup`, `-C`: Cleanup snapshots.
* `--list`, `-l`: List snapshots.
* `--name`, `-n`: Provide a name for the snapshot.
* `--yes`, `-y`: Skip confirmation prompt.

Example:

```shell
# Take a database snapshot for the current project
ddev snapshot

# Take a database snapshot for the current project, named `my_snapshot_name`
ddev snapshot --name my_snapshot_name

# Take a snapshot for the current project, cleaning up existing snapshots
ddev snapshot --cleanup

# Take a snapshot for the current project, cleaning existing snapshots and skipping prompt
ddev snapshot --cleanup -y

# List the current project’s snapshots
ddev snapshot --list

# Take a snapshot for each project
ddev snapshot --all
```

### `snapshot restore`

Restores a database snapshot from the `.ddev/db_snapshots` directory.

Flags:

* `--latest`: Use the latest snapshot.

Example:

```shell
# Restore the most recent snapshot
ddev snapshot restore --latest

# Restore the previously-taken `my_snapshot_name` snapshot
ddev snapshot restore my_snapshot_name
```

## `ssh`

Starts a shell session in a service container. Uses the web service by default.

Flags:

* `--dir`, `-d`: Defines the destination directory within the container.
* `--service`, `-s`: Defines the service to connect to. (default `"web"`)

Example:

```shell
# SSH into the current project’s web container
ddev ssh

# SSH into the current project’s database container
ddev ssh -s db

# SSH into the web container for my-project
ddev ssh my-project

# SSH into the docroot of the current project’s web container
ddev ssh -d /var/www/html
```

## `start`

Start a DDEV project.

Flags:

* `--all`, `-a`: Start all projects.
* `--skip-confirmation`, `-y`: Skip any confirmation steps.

Example:

```shell
# Start the current project
ddev start

# Start my-project and my-other-project
ddev start my-project my-other-project

# Start all projects
ddev start --all
```

## `stop`

*Aliases: `rm`, `remove`.*

Stop and remove the containers of a project. Does not lose or harm anything unless you add `--remove-data`.

Flags:

* `--all`, `-a`: Stop and remove all running or container-stopped projects and remove from global projects list.
* `--omit-snapshot`, `-O`: Omit/skip database snapshot.
* `--remove-data`, `-R`: Remove stored project data (MySQL, logs, etc.).
* `--snapshot`, `-S`: Create database snapshot.
* `--stop-ssh-agent`: Stop the `ddev-ssh-agent` container.
* `--unlist`, `-U`: Remove the project from global project list, so it won’t appear in [`ddev list`](#list) until started again.

Example:

```shell
# Stop the current project
ddev stop

# Stop my-project, my-other-project, and my-third-project
ddev stop my-project my-other-project my-third-project

# Stop all projects
ddev stop --all

# Stop all projects and the `ddev-ssh-agent` container
ddev stop --all --stop-ssh-agent

# Stop all projects and remove their data
ddev stop --remove-data
```

## `tableplus`

Open [TablePlus](https://tableplus.com) with the current project’s database (global shell host container command). This command is only available if `TablePlus.app` is installed as `/Applications/TablePlus.app`.

Example:

```shell
# Open the current project’s database in TablePlus
ddev tableplus
```

## `typo3`

Run the `typo3` command; available only in projects of type `typo3`, and only works if `typo3` is in the `$PATH` inside the container; normally it's in `vendor/bin/typo3` so will be found.

```shell
# Show typo3 site configuration
ddev typo3 site:show
```

## `version`

Print DDEV and component versions.

Example:

```shell
# Print DDEV and platform version details
ddev version
```

!!!tip
    `ddev --version` is a more concise command that only outputs the DDEV version without component versions.

## `wp`

Run the [WP-CLI `wp` command](https://wp-cli.org/); available only in projects of type `wordpress`.

```shell
# Install WordPress site using `wp core install`
ddev wp core install --url='$DDEV_PRIMARY_URL' --title='New-WordPress' --admin_user=admin --admin_email=admin@example.com --prompt=admin_password

```

## `xdebug`

Enable or disable [Xdebug](../debugging-profiling/step-debugging.md) (global shell web container command).

* The `on` argument is equivalent to `enable` and `true`.
* The `off` argument is equivalent to `disable` and `false`.

```shell
# Display whether Xdebug is running
ddev xdebug status

# Turn Xdebug on
ddev xdebug

# Turn Xdebug on
ddev xdebug on

# Turn Xdebug off
ddev xdebug off

# Toggle Xdebug on and off
ddev xdebug toggle
```

## `xhprof`

Enable or disable [Xhprof](../debugging-profiling/xhprof-profiling.md) (global shell web container command).

* The `on` argument is equivalent to `enable` and `true`.
* The `off` argument is equivalent to `disable` and `false`.

```shell
# Display whether Xhprof is running
ddev xhprof status

# Turn Xhprof on
ddev xhprof

# Turn Xhprof on
ddev xhprof on

# Turn Xhprof off
ddev xhprof off
```

## `yarn`

Run [`yarn` commands](https://yarnpkg.com/cli) inside the web container in the root of the project (global shell host container command).

!!!tip
    Use `--cwd` for another directory, or you can change directories to the desired directory and `ddev yarn` will act on the same relative directory inside the container.

!!!tip
    If you want to define your Yarn version on a per project basis, set `corepack_enable: true` in `.ddev/config.yaml` or `ddev config --corepack-enable`

Example:

```shell
# Use Yarn to install JavaScript packages
ddev yarn install

# Use Yarn to add the Lerna package
ddev yarn add lerna

# Use yarn in a relative directory
cd web/core && ddev yarn add lerna

# Use Yarn to add the Lerna package from the `web/core` directory
ddev yarn --cwd web/core add lerna

# Use latest yarn or specified yarn
ddev config --corepack-enable && ddev restart
ddev yarn set version stable
ddev yarn --version
```
