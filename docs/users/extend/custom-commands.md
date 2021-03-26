## Custom (Shell) Commands

It's quite easy to add custom commands to ddev; they can execute either on the host or in the various containers. The basic idea is to add a bash script to either the specific project in `.ddev/commands/host` or `.ddev/commands/<containername>` or globally for every project in `~/.ddev/commands`

There are example commands provided in `ddev/commands/*/*.example` that can just be copied or moved (or symlinked) and used as commands. For example, [.ddev/commands/host/mysqlworkbench.example](https://github.com/drud/ddev/blob/master/cmd/ddev/cmd/dotddev_assets/commands/host/mysqlworkbench.example) can be used to add a "ddev mysqlworkbench" command, just change it from "mysqlworkbench.example" to "mysqlworkbench". Also, a new `ddev mysql` command has been added using this technique (as a db container command). See [mysql command](https://github.com/drud/ddev/blob/master/cmd/ddev/cmd/dotddev_assets/commands/db/mysql). If you're on macOS or Linux (or some configurations of Windows) you can just `cd .ddev/commands/host && ln -s mysqlworkbench.example mysqlworkbench`.

### Notes for all command types

* The command filename is not what determines the name of the command.  That comes from the Usage doc line (`## Usage: commandname`).
* To confirm that your custom command is available, run `ddev -h`, and look for it in the list.

### Host commands

To provide host commands, place a bash script in .ddev/commands/host. For example, a PhpStorm launcher to make the `ddev PhpStorm` command might go in .ddev/commands/host/phpstorm` with these contents:

```bash
#!/usr/bin/env bash

## Description: Open PhpStorm with the current project
## Usage: phpstorm
## Example: "ddev phpstorm"

# Example is macOS-specific, but easy to adapt to any OS
open -a PhpStorm.app ${DDEV_APPROOT}
```

### Container commands

To provide a command which will execute in a container, add a bash script to `.ddev/commands/<container_name>`, for example, `.ddev/commands/web/mycommand`. The bash script will be executed inside the named container. For example, the [reload-nginx.example](https://github.com/drud/ddev/blob/master/cmd/ddev/cmd/dotddev_assets/commands/web/reload-nginx.example), which executes a script inside the container with the arguments provided, would go in `.ddev/commands/web/reload-nginx` as:

```bash
#!/bin/bash

## Description: Reload config for nginx and php-fpm inside web container
## Usage: restart-nginx
## Example: "ddev restart-nginx"

killall -HUP nginx php-fpm
```

In addition to commands that run in the standard ddev containers like "web" and "db", you can run commands in custom containers, just using the service name, like `.ddev/commands/solr/<command>`. Note, however, that your service must mount /mnt/ddev_config as the web and db containers do, so the `volumes` section of docker-compose.<servicename>.yaml needs:

```
    volumes:
    - ".:/mnt/ddev_config"
```

For example, to add a "solrtail" command that runs in a solr service, add `.ddev/commands/solr/solrtail` with:

```bash
#!/bin/bash

## Description: Tail the main solr log
## Usage: solrtail
## Example: ddev solrtail

tail -f /opt/solr/server/logs/solr.log

```

### Global commands

Global commands work exactly the same as project-level commands, you just have to put them in your global .ddev directory. Your home directory has a .ddev/commands in it; there you can add host or web or db commands. You might want to copy the drush.example above to ~/.ddev/commands/web to make the "ddev drush" command available in every project.

### Environment variables provided

A number of environment variables are provided to the script. These are generally supported, but please avoid using undocumented environment variables. Useful variables for host scripts are:

* DDEV_APPROOT: file system location of the project on the host)
* DDEV_DOCROOT: Relative path from approot to docroot
* DDEV_HOSTNAME: Comma-separated list of FQDN hostnames
* DDEV_HOST_DB_PORT: Localhost port of the database server
* DDEV_HOST_HTTPS_PORT: Localhost port for https on webserver
* DDEV_HOST_WEBSERVER_PORT: Localhost port of the webserver
* DDEV_PHP_VERSION
* DDEV_PRIMARY_URL: Primary URL for the project
* DDEV_PROJECT: Project name, like "d8composer"
* DDEV_PROJECT_TYPE: drupal8, typo3, backdrop, wordpress, etc.
* DDEV_ROUTER_HTTP_PORT: Router port for http
* DDEV_ROUTER_HTTPS_PORT: Router port for https
* DDEV_SITENAME: Project name, like "d8composer".
* DDEV_TLD: Top-level domain of project, like "ddev.site"
* DDEV_WEBSERVER_TYPE: nginx-fpm, apache-fpm

Useful variables for container scripts are:

* DDEV_DOCROOT: Relative path from approot to docroot
* DDEV_FILES_DIR: Directory of user-uploaded files
* DDEV_HOSTNAME: Comma-separated list of FQDN hostnames
* DDEV_PHP_VERSION
* DDEV_PRIMARY_URL: Primary URL for the project
* DDEV_PROJECT: Project name, like "d8composer"
* DDEV_PROJECT_TYPE: drupal8, typo3, backdrop, wordpress, etc.
* DDEV_ROUTER_HTTP_PORT: Router port for http
* DDEV_ROUTER_HTTPS_PORT: Router port for https
* DDEV_SITENAME: Project name, like "d8composer".
* DDEV_TLD: Top-level domain of project, like "ddev.site"
* DDEV_WEBSERVER_TYPE: nginx-fpm, apache-fpm
* IS_DDEV_PROJECT: if set to "true" it means that php is running under DDEV

### Annotations supported

The custom commands support various annotations in the header which are used to provide additional information about the command to the user.

#### Description

`Description` is used for the listing of available commands and for the help message of the custom command.

Usage: `## Description: <command-description>`
Example: `## Description: my great custom command`

#### Usage

`Usage` is used for the help message to provide an idea to the user how to use this command.

Usage: `## Usage: <command-usage>`
Example: `## Usage: commandname [flags] [args]`

#### Example

`Example` is used for the help message to provide some usage examples to the user. Use `\n` to force a line break.

Usage: `## Example: <command-example>`
Example: `## Example: commandname\ncommandname -h`

#### Flags

`Flags` is used for the help message. All defined flags here are listed with their shorthand if available. It has to be encoded according the following definition:

Usage: `## Flags: <json-definition>`

This is the minimal usage of a flags definition:

Example: `## Flags: [{"Name":"flag","Usage":"sets the flag option"}]`
Output:

```bash
Flags:
  -h, --help          help for ddev
  -f, --flag          sets the flag option
```

Multiple flags are separated by a comma:

Example: `## Flags: [{"Name":"flag1","Shorthand":"f","Usage":"flag1 usage"},{"Name":"flag2","Usage":"flag2 usage"}]`
Output:

```bash
Flags:
  -h, --help          help for ddev
  -f, --flag1         flag1 usage
      --flag2         flag2 usage
```

The following fields can be used for a flag definition:

* `Name`: the name as it appears on command line
* `Shorthand`: one-letter abbreviated flag
* `Usage`: help message
* `Type`: possible values are `bool`, `string`, `int`, `uint` (defaults to bool)
* `DefValue`: default value for usage message
* `NoOptDefVal`: default value, if the flag is on the command line without any options
* `Annotations`: used by cobra.Command bash autocomple code see <https://github.com/spf13/cobra/blob/master/bash_completions.md>

#### ProjectTypes

If your command should only be visible for a particular project type, `ProjectTypes` will allow you to define the supported types. This is especially useful for global custom commands. See <https://ddev.readthedocs.io/en/stable/users/cli-usage/#quickstart-guides> for more information about the supported project types. Multiple types are separated by a comma.

Usage: `## ProjectTypes: <list-of-project-types>`
Example: `## ProjectTypes: drupal7,drupal8,drupal9,backdrop`

#### OSTypes (host commands only)

If your host command should only run on one or more operating systems, add the `OSTypes` annotation. Multiple types are separated by a comma. Valid types are:

* `darwin` for macOS
* `windows` for Windows
* `linux` for Linux

Usage: `## OSTypes: <list-of-os-types>`
Example: `## OSTypes: darwin,linux`

#### HostBinaryExists (host commands only)

If your host command should only run if a particular file exists, add the `HostBinaryExists` annotation.

Usage: `## HostBinaryExists: <path/to/file>`
Example: `## HostBinaryExists: /Applications/Sequel ace.app`

### Known Windows OS issues

* **Line Endings**: If you are editing a custom command which will run in a container, it must have LF line endings (not traditional Windows CRLF line endings). Remember that a custom command in a container is a script that must execute in a Linux environmet.
* If ddev can't find "bash" to execute it, then the commands can't be used. If you are running inside git-bash in most any terminal, this shouldn't be an issue, and ddev should be able to find git-bash if it's in "C:\Program Files\Git\bin" as well. But if neither of those is true, add the directory of bash.exe to your PATH environment variable.
