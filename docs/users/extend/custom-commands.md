<h1>Custom Commands</h1>

It's quite easy to add custom commands to ddev; they can execute either on the host or in the various containers. The basic idea is to add a bash script to `.ddev/commands/host` or `.ddev/commands/<containername>`

There are example commands provided in `ddev/commands/*/*.example` that can just be copied or moved and used as commands. For example, [.ddev/commands/host/mysqlworkshop.example](https://github.com/drud/ddev/tree/master/cmd/ddev/cmd/asssets/commands/host/mysqlworkshop.example) can be used to add a "ddev mysqlworkshop" command, just change it from "mysqlworkshop.example" to "mysqlworkshop". Also, a new `ddev mysql` command has been added using this technique (as a db container command). See [mysql command](https://github.com/drud/ddev/tree/master/cmd/ddev/cmd/asssets/commands/db/mysql).

## Host commands

To provide host commands, place a bash script in .ddev/commands/host. For example, a PHPStorm launcher to make the `ddev PHPStorm` command might go in .ddev/commands/host/phpstorm` with these contents:

```
#!/bin/bash

## Description: Open PHPStorm with the current project
## Usage: phpstorm
## Example: "ddev phpstorm"

# Example is macOS-specific, but easy to adapt to any OS
open -a PHPStorm.app ${DDEV_APPROOT}
```

## Container commands

To provide a command which will execute in a container, add a bash script to `.ddev/commands/<container_name>`, for example, `.ddev/commands/web/mycommand`. The bash script will be executed inside the named container. For example, the [drush.example](https://github.com/drud/ddev/tree/master/cmd/ddev/cmd/asssets/commands/web/drush.example), which executes Drupal's drush inside the container with the arguments provided, would go in `.ddev/commands/web/drush` as:

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

## Windows: paths

## Environment variables provided


