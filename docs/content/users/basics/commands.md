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
additional services like MailHog and phpMyAdmin. You can run 'ddev describe' from
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

Add [SSH key authentication](../basics/cli-usage.md#ssh-into-containers) to the `ddev-ssh-agent` container.

Example:

```shell
# Add your SSH keys to the SSH agent container
ddev auth ssh
```

Flags:

* `--ssh-key-path`, `-d`: Full path to SSH key directory.

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

## `clean`

Removes items DDEV has created. (See [Uninstalling DDEV](../basics/uninstall.md).)

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

Executes a [Composer command](../basics/developer-tools.md#ddev-and-composer) within the web container.

Example:

```shell
# Install Composer packages
ddev composer install
```

## `config`

Create or modify a DDEV project’s configuration in the current directory.

!!!tip "You can also set these via YAML!"
    These settings, plus a few more, can be set by editing stored [Config Options](../configuration/config.md).

Example:

```shell
# Start interactive project configuration
ddev config

# Configure a Drupal 8 project with a `web` document root
ddev config --docroot=web --project-type=drupal8
```

Flags:

* `--additional-fqdns`: Comma-delimited list of project FQDNs.
* `--additional-hostnames`: Comma-delimited list of project hostnames.
* `--auto`: Automatically run config without prompting. (default `true`)
* `--bind-all-interfaces`: Bind host ports on all interfaces, not just on localhost network interface.
* `--composer-root`: Overrides the default Composer root directory for the web service.
* `--composer-root-default`: Unsets a web service Composer root directory override.
* `--composer-version`: Specify override for Composer version in the web container. This may be `""`, `"1"`, `"2"`, `"2.2"`, `"stable"`, `"preview"`, `"snapshot"`, or a specific version.
* `--create-docroot`: Create the docroot if it doesn’t exist.
* `--database`: Specify the database type:version to use. Defaults to `mariadb:10.4`.
* `--db-image`: Sets the db container image.
* `--db-image-default`: Sets the default db container image for this DDEV version.
* `--db-working-dir`: Overrides the default working directory for the db service.
* `--db-working-dir-default`: Unsets a db service working directory override.
* `--dba-image`: Sets the dba container image.
* `--dba-image-default`: Sets the default dba container image for this DDEV version.
* `--dba-working-dir`: Overrides the default working directory for the dba service.
* `--dba-working-dir-default`: Unsets a dba service working directory override.
* `--dbimage-extra-packages`: A comma-delimited list of Debian packages that should be added to db container when the project is started.
* `--default-container-timeout`: Default time in seconds that DDEV waits for all containers to become ready on start. (default `120`)
* `--disable-settings-management`: Prevent DDEV from creating or updating CMS settings files.
* `--docroot`: Provide the relative docroot of the project, like `docroot` or `htdocs` or `web`. (defaults to empty, the current directory)
* `--fail-on-hook-fail`: Decide whether `ddev start` should be interrupted by a failing hook.
* `--host-db-port`: The db container’s localhost-bound port.
* `--host-dba-port`: The dba (phpMyAdmin) container’s localhost-bound port, if exposed via bind-all-interfaces.
* `--host-https-port`: The web container’s localhost-bound HTTPS port.
* `--host-webserver-port`: The web container’s localhost-bound port.
* `--http-port`: The router HTTP port for this project.
* `--https-port`: The router HTTPS port for this project.
* `--image-defaults`: Sets the default web, db, and dba container images.
* `--mailhog-https-port`: Router port to be used for MailHog HTTPS access.
* `--mailhog-port`: Router port to be used for MailHog HTTP access.
* `--mutagen-enabled`: Enable Mutagen asynchronous update of project in web container.
* `--nfs-mount-enabled`: Enable NFS mounting of project in container.
* `--ngrok-args`: Provide extra args to ngrok in `ddev share`.
* `--no-project-mount`: Whether or not to skip mounting project code into the web container.
* `--nodejs-version`: Specify the Node.js version to use if you don’t want the default Node.js 16.
* `--omit-containers`: Comma-delimited list of container types that should not be started when the project is started.
* `--php-version`: PHP version that will be enabled in the web container.
* `--phpmyadmin-https-port`: Router port to be used for phpMyAdmin (dba) HTTPS container access.
* `--phpmyadmin-port`: Router port to be used forphpMyAdmin (dba) HTTP container access.
* `--project-name`: Provide the project name of project to configure. (normally the same as the last part of directory name)
* `--project-tld`: Set the top-level domain to be used for projects. (default `"ddev.site"`)
* `--project-type`: Provide the project type: `backdrop`, `drupal10`, `drupal6`, `drupal7`, `drupal8`, `drupal9`, `laravel`, `magento`, `magento2`, `php`, `shopware6`, `typo3`, `wordpress`. This is autodetected and this flag is necessary only to override the detection.
* `--show-config-location`: Output the location of the `config.yaml` file if it exists, or error that it doesn’t exist.
* `--timezone`: Specify timezone for containers and PHP, like `Europe/London` or `America/Denver` or `GMT` or `UTC`.
* `--upload-dir`: Sets the project’s upload directory, the destination directory of the import-files command.
* `--use-dns-when-possible`: Use DNS for hostname resolution instead of `/etc/hosts` when possible. (default `true`)
* `--web-environment`: Set the environment variables in the web container: `--web-environment="TYPO3_CONTEXT=Development,SOMEENV=someval"`
* `--web-environment-add`: Append environment variables to the web container: `--web-environment="TYPO3_CONTEXT=Development,SOMEENV=someval"`
* `--web-image`: Sets the web container image.
* `--web-image-default`: Sets the default web container image for this DDEV version.
* `--web-working-dir`: Overrides the default working directory for the web service.
* `--web-working-dir-default`: Unsets a web service working directory override.
* `--webimage-extra-packages`: A comma-delimited list of Debian packages that should be added to web container when the project is started.
* `--webserver-type`: Sets the project’s desired webserver type: `nginx-fpm` or `apache-fpm`.
* `--working-dir-defaults`: Unsets all service working directory overrides.
* `--xdebug-enabled`: Whether or not Xdebug is enabled in the web container.

### `config global`

Change global configuration.

```shell
# Opt out of sharing anonymized usage information
ddev config global --instrumentation-opt-in=false

# Skip phpMyAdmin and the SSH agent for all projects
ddev config global --omit-containers=dba,ddev-ssh-agent
```

* `--auto-restart-containers`: If `true`, automatically restart containers after a reboot or Docker restart.
* `--disable-http2`: Optionally disable http2 in `ddev-router`; `ddev config global --disable-http2` or `ddev config global --disable-http2=false`.
* `--fail-on-hook-fail`: If true, `ddev start` will fail when a hook fails.
* `--instrumentation-opt-in`: `instrumentation-opt-in=true`.
* `--internet-detection-timeout-ms`: Increase timeout when checking internet timeout, in milliseconds. (default `3000`)
* `--letsencrypt-email`: Email associated with Let’s Encrypt; `ddev global --letsencrypt-email=me@example.com`.
* `--mutagen-enabled`: If `true`, web container will use Mutagen caching/asynchronous updates.
* `--nfs-mount-enabled`: Enable NFS mounting on all projects globally.
* `--no-bind-mounts`: If `true`, don’t use bind-mounts. Useful for environments like remote Docker where bind-mounts are impossible. (default `true`)
* `--omit-containers`: For example, `--omit-containers=dba,ddev-ssh-agent`.
* `--required-docker-compose-version`: Override default docker-compose version.
* `--router-bind-all-interfaces`: `router-bind-all-interfaces=true`.
* `--simple-formatting`: If `true`, use simple formatting and no color for tables.
* `--table-style`: Table style for list and describe, see `~/.ddev/global_config.yaml` for values.
* `--use-docker-compose-from-path`: If `true`, use docker-compose from path instead of private `~/.ddev/bin/docker-compose`. (default `true`)
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
ddev debug dockerchck
```

### `debug download-images`

Download all images required by DDEV.

Example:

```shell
# Download DDEV’s Docker images
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

Migrate a MySQL or MariaDB database to a different `dbtype:dbversion`. Works only with MySQL and MariaDB, not with PostgreSQL.

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

Run diagnostics using the embedded [test script](https://github.com/drud/ddev/blob/master/cmd/ddev/cmd/scripts/test_ddev.sh).

Example:

```shell
# Run DDEV’s diagnostic suite
ddev debug test
```

## `delete`

Remove all information, including the database, for an existing project.

Flags:

* `--all`, `-a`: Delete all projects.
* `--clean-containers`: Clean up all DDEV docker containers not required by this version of DDEV. (default true)
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

With with `--all`, it deletes all `drud/ddev-*` Docker images.

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

## `exec`

*Alias: `.`.*

[Execute a shell command in the container](../basics/cli-usage.md#executing-commands-in-containers) for a service. Uses the web service by default.

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

# Say “hi” from the phpMyAdmin container
ddev exec --service dba echo hi
```

## `export-db`

Dump a database to a file or to stdout.

Flags:

* `--bzip2`: Use bzip2 compression.
* `--file`, `-f`: Provide the path to output the dump.
* `--gzip`, `-z`: Use gzip compression. (default `true`)
* `--target-db`, `-d`: If provided, target-db is alternate database to export. (default `"db"`)
* `--xz`: Use xz compression.

Example:

```shell
# Dump and compress the current project’s database to `/tmp/db.sql.gz`
ddev export-db --file=/tmp/db.sql.gz

# Dump the current project’s database, without compressing it, to `/tmp/db.sql`
ddev export-db --gzip=false --file /tmp/db.sql

# Dump and compress the current project’s `foo` database instead of `db`
ddev export-db --target-db=foo --file=/tmp/db.sql.gz

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

Example:

```shell
# List official add-ons
ddev get --list

# List official and third-party add-ons
ddev get --list --all

# Download the official Redis add-on
ddev get drud/ddev-redis

# Download the Drupal 9 Solr add-on from its v0.0.5 release tarball
ddev get https://github.com/drud/ddev-drupal9-solr/archive/refs/tags/v0.0.5.tar.gz

# Copy an add-on available in another directory
ddev get /path/to/package

# Copy an add-on from a tarball in another directory
ddev get /path/to/tarball.tar.gz
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

* `--remove`, `-r`: Remove the provided host name - ip correlation.
* `--remove-inactive`, `-R`: Remove host names of inactive projects.

Example:

```shell
ddev hostname somesite.ddev.local 127.0.0.1
```

## `import-db`

[Import a SQL file](database-management.md) into the project.

Flags:

* `--extract-path`: If provided asset is an archive, provide the path to extract within the archive.
* `--no-drop`: Set if you do NOT want to drop the db before importing.
* `--progress`, `-p`: Display a progress bar during import. (default `true`)
* `--src`, `-f`: Provide the path to a SQL dump in `.sql`, `.tar`, `.tar.gz`, `.tgz`, `.bz2`, `.xx`, or `.zip` format.
* `--target-db`, `-d`: If provided, target-db is alternate database to import into. (default `"db"`)

Example:

```shell
# Start the interactive import utility
ddev import-db

# Import the `.tarballs/db.sql` dump to the project database
ddev import-db --src=.tarballs/db.sql

# Import the compressed `.tarballs/db.sql.gz` dump to the project database
ddev import-db --src=.tarballs/db.sql.gz

# Import the compressed `.tarballs/db.sql.gz` dump to a `newdb` database
ddev import-db --target-db=newdb --src=.tarballs/db.sql.gz

# Import the `db.sql` dump to the project database
ddev import-db <db.sql

# Import the `db.sql` dump to a `newdb` database
ddev import-db newdb <db.sql

# Uncompress `db.sql.gz` and pipe the result to the `import-db` command
gzip -dc db.sql.gz | ddev import-db
```

## `import-files`

Pull the uploaded files directory of an existing project to the default [public upload directory](../basics/cli-usage.md#ddev-import-files) of your project.

Flags:

* `--extract-path`: If provided asset is an archive, optionally provide the path to extract within the archive.
* `--src`: Provide the path to the source directory or archive to import. (Archive can be `.tar`, `.tar.gz`, `.tar.xz`, `.tar.bz2`, `.tgz`, or `.zip`.)

Example:

```shell
# Extract+import `/path/to/files.tar.gz` to the project’s upload directory
ddev import-files --src=/path/to/files.tar.gz

# Import `/path/to/dir` contents to the project’s upload directory
ddev import-files --src=/path/to/dir
```

## `launch`

Launch a browser with the current site (global shell host container command).

Flags:

* `--phpmyadmin`, `-p`: Open phpMyAdmin.
* `--mailhog`, `-m`: Open MailHog.

Example:

```shell
# Open your project’s base URL in the default browser
ddev launch

# Open MailHog in the default browser
ddev launch --mailhog

# Open your project’s base URL appended with `temp/phpinfo.php`
ddev launch temp/phpinfo.php
```

## `list`

*Aliases: `l`, `ls`.*

List projects.

Flags:

* `--active-only`, `-A`: If set, only currently active projects will be displayed.
* `--continuous`: If set, project information will be emitted until the command is stopped.
* `--continuous-sleep-interval`, `-I`: Time in seconds between `ddev list --continuous` output lists. (default `1`)
* `--wrap-table`, `-W`: Display table with wrapped text if required.

Example:

```shell
# List all projects
ddev list

# List all running projects
ddev list --active-only
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

## `mysql`

Run MySQL client in the database container (global shell db container command).

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

Example:

```shell
# Use `nvm` to switch to Node.js v18
ddev nvm install 18
```

## `pause`

*Aliases: `sc`, `stop-containers`.*

Uses `docker stop` to pause/stop the containers belonging to a project.

!!!tip
    This leaves the containers instantiated instead of removing them like [`ddev stop`](#stop) does.

Flags:

* `--all`, `-a`: Pause all running projects.

Example:

```shell
# Pause the current project’s containers
ddev pause

# Pause my-project’s containers
ddev pause my-project

# Pause all projects
ddev pause --all
```

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

## `pull`

Pull files and database using a configured [provider plugin](../providers/).

Flags:

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
```

## `push`

Push files and database using a configured [provider plugin](../providers/).

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
    Some ngrok arguments are supported via CLI, but *any* ngrok flag can be specified in the [`ngrok_args` config setting](../configuration/config.md#ngrok_args).

Flags:

* `--subdomain`: Subdomain to use with paid ngrok account.

Example:

```shell
# Share the current project with ngrok
ddev share

# Share the current project with ngrok, using subdomain `foo.*`
ddev share --subdomain foo

# Share the current project using ngrok’s basic-auth argument
ddev share --basic-auth username:pass1234

# Share my-project with ngrok
ddev share my-project
```

## `snapshot`

Create a database snapshot for one or more projects.

This uses `xtrabackup` or `mariabackup` to create a database snapshot in the `.ddev/db_snapshots` directory. These are compatible with server backups using the same tools and can be restored with the [`snapshot restore`](#snapshot-restore) command.

See [Snapshotting and Restoring a Database](../basics/cli-usage.md#snapshotting-and-restoring-a-database) for more detail, or [Database Management](../basics/database-management.md) for more on working with databases in general.

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
* `--select`, `-s`: Interactively select a project to start.
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
* `--select`, `-s`: Interactively select a project to stop.
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

Open [TablePlus](https://tableplus.com) with the current project’s database (global shell host container command).

Example:

```shell
# Open the current project’s database in TablePlus
ddev tableplus
```

## `version`

Print DDEV and component versions.

Example:

```shell
# Print DDEV and platform version details
ddev version
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

Run [`yarn` commands](https://yarnpkg.com/getting-started/migration#cli-commands) inside the web container in the root of the project (global shell host container command).

!!!tip
    Use `--cwd` for another directory.

Example:

```shell
# Use Yarn to install JavaScript packages
ddev yarn install

# Use Yarn to add the Lerna package
ddev yarn add lerna

# Use Yarn to add the Lerna package from the `web/core` directory
ddev yarn --cwd web/core add lerna
```
