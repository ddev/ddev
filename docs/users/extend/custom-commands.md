<h1>Custom Commands</h1>

It's quite easy to add custom commands to ddev; they can execute either on the host or in the various containers. The basic idea is to add a bash script to `.ddev/commands/host` or `.ddev/commands/<containername>`

There are example commands provided in `ddev/commands/*/*.example` that can just be copied or moved (or symlinked) and used as commands. For example, [.ddev/commands/host/mysqlworkbench.example](https://github.com/drud/ddev/blob/master/cmd/ddev/cmd/dotddev_assets/commands/host/mysqlworkbench.example) can be used to add a "ddev mysqlworkbench" command, just change it from "mysqlworkbench.example" to "mysqlworkbench". Also, a new `ddev mysql` command has been added using this technique (as a db container command). See [mysql command](https://github.com/drud/ddev/blob/master/cmd/ddev/cmd/dotddev_assets/commands/db/mysql). If you're on macOS or Linux (or some configurations of Windows) you can just `cd .ddev/commands/host && ln -s mysqlworkbench.example mysqlworkbench`.

## Notes for all command types

- Script files should be set to executable (`chmod +x <scriptfile>`). ddev does _not_ need to be restarted to see new commands.
- The command filename is not what determines the name of the command.  That comes from the Usage doc line (`## Usage: commandname`).
- To confirm that your custom command is available, run `ddev -h`, and look for it in the list.

## Host commands

To provide host commands, place a bash script in .ddev/commands/host. For example, a PHPStorm launcher to make the `ddev PHPStorm` command might go in .ddev/commands/host/phpstorm` with these contents:

```
#!/usr/bin/env bash

## Description: Open PHPStorm with the current project
## Usage: phpstorm
## Example: "ddev phpstorm"

# Example is macOS-specific, but easy to adapt to any OS
open -a PHPStorm.app ${DDEV_APPROOT}
```

## Container commands

To provide a command which will execute in a container, add a bash script to `.ddev/commands/<container_name>`, for example, `.ddev/commands/web/mycommand`. The bash script will be executed inside the named container. For example, the [drush.example](https://github.com/drud/ddev/blob/master/cmd/ddev/cmd/dotddev_assets/commands/web/drush.example), which executes Drupal's drush inside the container with the arguments provided, would go in `.ddev/commands/web/drush` as:

```
#!/bin/bash

## Description: Run drush inside the web container
## Usage: drush [flags] [args]
## Example: "ddev drush uli" or "ddev drush sql-cli" or "ddev drush --version"

drush $@
```

In addition to commands that run in the standard ddev containers like "web" and "db", you can run commands in custom containers, just using the service name, like `.ddev/commands/solr/<command>`. Not, however, that your service must mount /mnt/ddev_config as the web and db containers do, so the `volumes` section of docker-compose.<servicename>.yaml needs: 

```
    volumes:
    - ".:/mnt/ddev_config"
``` 

For example, to add a "solrtail" command that runs in a solr service, add `.ddev/commands/solr/solrtail` with:

```
#!/bin/bash

## Description: Tail the main solr log
## Usage: solrtail
## Example: ddev solrtail

tail -f /opt/solr/server/logs/solr.log

```

## Environment variables provided

A number of environment variables are provided to the script. Useful variables for host scripts are:

- DDEV_APPROOT: file system location of the project on the host)
- DDEV_HOST_DB_PORT: Localhost port of the database server
- DDEV_HOST_WEBSERVER_PORT: Localhost port of the webserver
- DDEV_HOST_HTTPS_PORT: Localhost port for https on webserver
- DDEV_DOCROOT: Relative path from approot to docroot
- DDEV_HOSTNAME: Comma-separated list of FQDN hostnames
- DDEV_PHP_VERSION
- DDEV_WEBSERVER_TYPE: nginx-fpm, apache-fpm, apache-cgi
- DDEV_PROJECT_TYPE: drupal8, typo3, backdrop, wordpress, etc.
- DDEV_ROUTER_HTTP_PORT: Router port for http
- DDEV_ROUTER_HTTPS_PORT: Router port for https

Useful variables for container scripts are:

- DDEV_DOCROOT: Relative path from approot to docroot
- DDEV_HOSTNAME: Comma-separated list of FQDN hostnames
- DDEV_PHP_VERSION
- DDEV_WEBSERVER_TYPE: nginx-fpm, apache-fpm, apache-cgi
- DDEV_PROJECT_TYPE: drupal8, typo3, backdrop, wordpress, etc.
- DDEV_ROUTER_HTTP_PORT: Router port for http
- DDEV_ROUTER_HTTPS_PORT: Router port for https


## Known Windows OS issues

* **Line Endings**: If you are editing a custom command which will run in a container, it must have LF line endings (not traditional Windows CRLF line endings). Remember that a custom command in a container is a script that must execute in a Linux environmet. 
* If ddev can't find "bash" to execute it, then the commands can't be used. If you are running inside git-bash in most any terminal, this shouldn't be an issue, and ddev should be able to find git-bash if it's in "C:\Program Files\Git\bin" as well. But if neither of those is true, add the directory of bash.exe to your PATH environment variable.
* If you're using Docker Toolbox, the IP address for things like `ddev mysql` is not 127.0.0.1, it's likely 192.168.99.100. 
