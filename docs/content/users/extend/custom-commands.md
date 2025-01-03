---
search:
  boost: 2
---
# Custom Commands

Custom commands can easily be added to DDEV, to be executed on the host or in containers.

This involves adding a Bash script to the project in `.ddev/commands/host`, a specific container in `.ddev/commands/<containername>`, or globally in `~/.ddev/commands`.

Example commands in `ddev/commands/*/*.example` can be copied, moved, or symlinked.

For example, [.ddev/commands/host/mysqlworkbench.example](https://github.com/ddev/ddev/blob/main/pkg/ddevapp/global_dotddev_assets/commands/host/mysqlworkbench.example) can be used to add a `ddev mysqlworkbench` command. Rename it from `mysqlworkbench.example` to `mysqlworkbench`. If you’re on macOS or Linux (or some configurations of Windows) you can `cd .ddev/commands/host && ln -s mysqlworkbench.example mysqlworkbench`.

The [`ddev mysql`](../usage/commands.md#mysql) runs the `mysql` client inside the `db` container command using this technique. See the [`ddev mysql` command](https://github.com/ddev/ddev/blob/main/pkg/ddevapp/global_dotddev_assets/commands/db/mysql).

## Notes for All Command Types

* The command filename is not what determines the name of the command. That comes from the “Usage” doc line (`## Usage: commandname`).
* To confirm that your custom command is available, run `ddev -h` and look for it in the list.

## Host Commands

To provide host commands, place a Bash script in `.ddev/commands/host`. For example, a PhpStorm launcher to make the `ddev phpstorm` command might go in `.ddev/commands/host/phpstorm` with these contents. The `OSTypes` and `HostBinaryExists` annotations are optional, but are useful to prevent the command from showing up if it's not useful to the user.

```bash
#!/usr/bin/env bash

## Description: Open PhpStorm with the current project
## Usage: phpstorm
## Example: "ddev phpstorm"
## OSTypes: darwin
## HostBinaryExists: "/Applications/PhpStorm.app"

# Example is macOS-specific, but easy to adapt to any OS
open -a PhpStorm.app ${DDEV_APPROOT}
```

## Container Commands

To provide a command which will execute in a container, add a Bash script to `.ddev/commands/<container_name>`, for example, `.ddev/commands/web/mycommand`. The Bash script will be executed inside the named container. For example, see the [several standard DDEV script-based web container commands](https://github.com/ddev/ddev/blob/main/pkg/ddevapp/global_dotddev_assets/commands/web).

You can run commands in custom containers as well as standard DDEV `web` and `db` containers. Use the service name, like `.ddev/commands/solr/<command>`. The only catch with a custom container is that your service must mount `/mnt/ddev-global-cache` like the `web` and `db` containers do; the `volumes` section of `docker-compose.<servicename>.yaml` needs:

```
    volumes:
      - ddev-global-cache:/mnt/ddev-global-cache
```

For example, to add a `solrtail` command that runs in a Solr service, add `.ddev/commands/solr/solrtail` with:

```bash
#!/usr/bin/env bash

## Description: Tail the main solr log
## Usage: solrtail
## Example: ddev solrtail

tail -f /opt/solr/server/logs/solr.log
```

## Global Commands

Global commands work exactly the same as project-level commands, but they need to go in your *global* `.ddev` directory. Your home directory has a `.ddev/commands` in it, where you can add host, web, or db commands.

Changes to the command files in the global `.ddev` directory need a `ddev start` for changes to be picked up by a project, as the global commands are copied to the project on start.

## Shell Command Examples

There are many examples of [global](https://github.com/ddev/ddev/tree/main/pkg/ddevapp/global_dotddev_assets/commands) and [project-level](https://github.com/ddev/ddev/tree/main/pkg/ddevapp/dotddev_assets/commands) custom/shell commands that ship with DDEV you can adapt for your own use. They can be found in your `~/.ddev/commands/*` directories and in your project’s `.ddev/commands/*` directories. There you’ll see how to provide usage, examples, and how to use arguments provided to the commands. For example, the [`xdebug` command](https://github.com/ddev/ddev/blob/main/pkg/ddevapp/global_dotddev_assets/commands/web/xdebug) shows simple argument processing and the [launch command](https://github.com/ddev/ddev/blob/main/pkg/ddevapp/global_dotddev_assets/commands/host/launch) demonstrates flag processing.

## Command Line Completion

If your custom command has a set of pre-determined valid arguments it can accept, you can use the [`AutocompleteTerms`](#autocompleteterms-annotation).

For dynamic completion, you can create a separate script with the same name in a directory named `autocomplete`.
For example, if your command is in `~/.ddev/commands/web/my-command`, your autocompletion script will be in `~/.ddev/commands/web/autocomplete/my-command`.

When you press tab on the command line after your command, the associated autocomplete script will be executed. The current command line (starting with the name of your command) will be passed into the completion script as arguments. If there is a space at the end of the command line, an empty argument will be included.

For example:

* `ddev my-command <tab>` will pass `my-command` and an empty argument into the autocomplete script.
* `ddev my-command som<tab>` will pass `my-command`, and `som` into the autocomplete script.

The autocomplete script should echo the valid arguments as a string separated by line breaks. You don't need to filter the arguments by the last argument string (e.g. if the last argument is `som`, you don't need to filter out any arguments that don't start with `som`). That will be handled for you before the result is given to your shell as completion suggestions.

The web container's [`nvm` autocomplete script](https://github.com/ddev/ddev/blob/main/pkg/ddevapp/global_dotddev_assets/commands/web/autocomplete/nvm) shows how this can be used to forward completion requests to a relevant script in the container.

## Environment Variables Provided

A number of environment variables are provided to these command scripts. These are generally supported, but please avoid using undocumented environment variables. Useful variables for host scripts are:

* `DDEV_APPROOT`: File system location of the project on the host
* `DDEV_DATABASE`: Database in use, in format `type:version` (example: `mariadb:10.5`)
* `DDEV_DATABASE_FAMILY`: Database "family" (example: `mysql`, `postgres`), useful for database connection URLs
* `DDEV_DOCROOT`: Relative path from approot to docroot
* `DDEV_GID`: The GID the web container runs as
* `DDEV_HOSTNAME`: Comma-separated list of FQDN hostnames
* `DDEV_HOST_DB_PORT`: Localhost port of the database server
* `DDEV_HOST_HTTPS_PORT`: Localhost port for HTTPS on web server
* `DDEV_HOST_MAILPIT_PORT`: Localhost port for Mailpit
* `DDEV_HOST_WEBSERVER_PORT`: Localhost port of the web server
* `DDEV_MAILPIT_HTTP_PORT`: Router Mailpit port for HTTP
* `DDEV_MAILPIT_HTTPS_PORT`: Router Mailpit port for HTTPS
* `DDEV_MUTAGEN_ENABLED`: `true` if Mutagen is enabled
* `DDEV_PHP_VERSION`: Current PHP version
* `DDEV_PRIMARY_URL`: Primary project URL
* `DDEV_PROJECT`: Project name, like `d8composer`
* `DDEV_PROJECT_STATUS`: Project status determined from the `web` and `db` services health, like `starting`, `running`, `stopped`, `paused`, or another status returned from Docker, including `healthy`, `unhealthy`, `exited`, `restarting`
* `DDEV_PROJECT_TYPE`: `backdrop`, `drupal`, `typo3`,`wordpress`, etc.
* `DDEV_ROUTER_HTTP_PORT`: Router port for HTTP
* `DDEV_ROUTER_HTTPS_PORT`: Router port for HTTPS
* `DDEV_SITENAME`: Project name, like `d8composer`
* `DDEV_TLD`: Top-level project domain, like `ddev.site`
* `DDEV_UID`: The UID the web container runs as
* `DDEV_WEBSERVER_TYPE`: `nginx-fpm` or `apache-fpm`
* `GOARCH`: Architecture (`arm64`, `amd64`)
* `GOOS`: Operating system (`windows`, `darwin`, `linux`)

Useful variables for container scripts are:

* `DDEV_DOCROOT`: Relative path from approot to docroot
* `DDEV_FILES_DIR`: *Deprecated*, first directory of user-uploaded files
* `DDEV_FILES_DIRS`: Comma-separated list of directories of user-uploaded files
* `DDEV_HOSTNAME`: Comma-separated list of FQDN hostnames
* `DDEV_MUTAGEN_ENABLED`: `true` if Mutagen is enabled
* `DDEV_PHP_VERSION`: Current PHP version
* `DDEV_PRIMARY_URL`: Primary URL for the project
* `DDEV_PROJECT`: Project name, like `d8composer`
* `DDEV_PROJECT_TYPE`: `backdrop`, `drupal`, `typo3`,`wordpress`, etc.
* `DDEV_ROUTER_HTTP_PORT`: Router port for HTTP
* `DDEV_ROUTER_HTTPS_PORT`: Router port for HTTPS
* `DDEV_SITENAME`: Project name, like `d8composer`
* `DDEV_TLD`: Top-level project domain, like `ddev.site`
* `DDEV_WEBSERVER_TYPE`: `nginx-fpm` or `apache-fpm`
* `IS_DDEV_PROJECT`: If `true`, PHP is running under DDEV

## Annotations Supported

Custom commands support various annotations in the header for providing additional information to the user.

### `Description` Annotation

`Description` should briefly describe the command in its help message.

Usage: `## Description: <command-description>`

Example: `## Description: my great custom command`

### `Usage` Annotation

`Usage` should explain how to use the command in its help message.

Usage: `## Usage: <command-usage>`

Example: `## Usage: commandname [flags] [args]`

### `Example` Annotation

`Example` should demonstrate how the command might be used. Use `\n` to force a line break.

Usage: `## Example: <command-example>`

Example: `## Example: commandname\ncommandname -h`

### `Aliases` Annotation

If your command should have one or more aliases, add the `Aliases` annotation. Multiple aliases are separated by a comma:

Usage: `## Aliases: <list-of-aliases>`

Example: `## Aliases: cacheclear,cache-clear,cache:clear`

### `Flags` Annotation

`Flags` should explain any available flags, including their shorthand when relevant, for the help message. It has to be encoded according the following definition:

If no flags are specified, the command will have its flags parsing disabled. Global flags like `--help` will not work unless the command supports them.

You can still do `ddev help <command>` to see the command's provided usage help.

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
* `Type`: possible values are `bool`, `string`, `int`, `uint` (defaults to `bool`)
* `DefValue`: default value for usage message
* `NoOptDefVal`: default value, if the flag is on the command line without any options
* `Annotations`: used by cobra.Command Bash autocomplete code (see <https://github.com/spf13/cobra/blob/main/site/content/completions/bash.md>)

### `AutocompleteTerms` Annotation

If your command accepts specific arguments, and you know ahead of time what those arguments are, you can use this annotation to provide those arguments for autocompletion.

Usage: `## AutocompleteTerms: [<list-of-valid-arguments>]`

Example: `## AutocompleteTerms: ["enable","disable","toggle","status"]`

### `CanRunGlobally` Annotation

This annotation is only available for global host commands.

Use `CanRunGlobally: true` if your global host command can be safely run even if the current working directory isn't inside a DDEV project.

This will make your command available to run regardless of what your current working directory is when you run it.

This annotation will have no effect if you are also using one of the following annotations:

* `ProjectTypes`
* `DBTypes`

Example: `## CanRunGlobally: true`

### `ProjectTypes` Annotation

If your command should only be visible for a specific project type, `ProjectTypes` will allow you to define the supported types. This is especially useful for global custom commands. See [Quickstart for many CMSes](../../users/quickstart.md) for more information about the supported project types. Multiple types are separated by a comma.

Usage: `## ProjectTypes: <list-of-project-types>`

Example: `## ProjectTypes: drupal7,drupal,backdrop`

### `OSTypes` Annotation (Host Commands Only)

If your host command should only run on one or more operating systems, add the `OSTypes` annotation. Multiple types are separated by a comma. Valid types are:

* `darwin` for macOS
* `windows` for Windows
* `linux` for Linux

Usage: `## OSTypes: <list-of-os-types>`

Example: `## OSTypes: darwin,linux`

### `HostBinaryExists` Annotation (Host Commands Only)

If your host command should only run if a particular file exists, add the `HostBinaryExists` annotation.

Usage: `## HostBinaryExists: <path/to/file>`

Example: `## HostBinaryExists: /Applications/Sequel ace.app`

### `DBTypes` Annotation

If your command should only be available for a particular database type, add the `DBTypes` annotation. Multiple types are separated by a comma. Valid types the available database types.

Usage: `## DBTypes: <type>`

Example: `## DBTypes: postgres`

### `HostWorkingDir` Annotation (Container Commands Only)

If your container command should run from the directory you are running the command in the host, add the `HostWorkingDir` annotation.

Example: `## HostWorkingDir: true`

### `ExecRaw` Annotation (Container Commands Only)

Use `ExecRaw: true` to pass command arguments directly to the container as-is.

For example, when `ExecRaw` is true, `ddev yarn --help` returns the help for `yarn`, not DDEV's help for the `ddev yarn` command.

We recommend  using this annotation for all container commands. The default behavior is retained to avoid breaking existing commands.

Example: `## ExecRaw: true`

## Known Windows Issues

### Line Endings

If you’re editing a custom command to be run in a container, it must have LF line endings and not traditional Windows CRLF line endings. Remember that a custom command in a container is a script that must execute in a Linux environment.

### Bash

Commands can’t be executed if DDEV can’t find `bash`. If you’re running inside Git Bash in most any terminal, this shouldn’t be an issue, and DDEV should be able to find `git-bash` if it’s in `C:\Program Files\Git\bin` as well. But if neither of those is true, add the directory of `bash.exe` to your `PATH` environment variable.
