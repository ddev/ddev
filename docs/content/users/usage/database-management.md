# Database Management

DDEV provides lots of flexibility for managing your databases between your local, staging and production environments. You may commonly use the [`ddev import-db`](../usage/commands.md#import-db) and [`ddev export-db`](../usage/commands.md#export-db) commands, but there are plenty of other adaptable ways to work with your databases.

!!!tip
    You can run `ddev [command] --help` for more info on many of the topics below.

## Database Imports

Import a database with one command, from one of the following file formats:  
**`.sql`, `.sql.gz`, `.mysql`, `.mysql.gz`, `.tar`, `.tar.gz`, and `.zip`**.

Here’s an example of a database import using DDEV:

```bash
ddev import-db --file=dumpfile.sql.gz
```

You can also:

* Use [`ddev mysql`](../usage/commands.md#mysql) or `ddev psql` or the `mysql` and `psql` commands inside the `web` and `db` containers.
* Use a [database client](#database-clients) or [database GUI](#database-guis) to import and browse data.

## Database Backends and Defaults

You can use a [variety of different database types](../extend/database-types.md#database-server-types), including MariaDB (5.5–10.8), MySQL (5.5–8.0), and PostgreSQL (9–15). If you want to _change_ database type, you need to export your database, run [`ddev delete`](../usage/commands.md#delete) to remove the project (and its existing database), change to a new database type, run [`ddev start`](../usage/commands.md#start) again, and [import your data](../usage/commands.md#import-db).

DDEV creates a default database named `db` and default permissions for the `db` user with password `db`, and it’s on the (inside Docker) hostname `db`.

## Extra Databases

You can easily create and populate additional databases. For example, `ddev import-db --target-db=backend --file=backend.sql.gz` will create the database named `backend` with permissions for that same `db` user and import from the `backend.sql.gz dumpfile`.

You can export in the same way: `ddev export-db -f mysite.sql.gz` will export your default database (`db`). `ddev export-db --target-db=backend -f backend-export.sql.gz` will dump the database named `backend`.

## Snapshots

Snapshots let you easily save the entire status of all of your databases, which can be great when you’re working incrementally on migrations or updates and want to save state so you can start right back where you were.

Snapshots can be named for easier reference later on. For example, [`ddev snapshot --name=two-dbs`](../usage/commands.md#snapshot) would make a snapshot named `two-dbs` in the `.ddev/db_snapshots` directory. It includes the entire state of the db server, so in the case of our two databases above, both databases and the system level `mysql` or `postgres` database will all be snapshotted. Then if you want to delete everything with `ddev delete -O` (omitting the snapshot since we have one already), and then [`ddev start`](../usage/commands.md#start) again, we can `ddev snapshot restore two-dbs` and we’ll be right back where we were.

Use the [`ddev snapshot restore`](../usage/commands.md#snapshot-restore) command to interactively choose among snapshots, or append `--latest` to restore the most recent snapshot: `ddev snapshot restore --latest`.

## Database Clients

The `ddev mysql` and `ddev psql` commands give you direct access to the `mysql` and `psql` clients in the database container, which can be useful for quickly running commands while you work. You might run `ddev mysql` to use interactive commands like `DROP DATABASE backend;` or `SHOW TABLES;`, or do things like `echo "SHOW TABLES;" | ddev mysql` or `ddev mysql -uroot -proot` to get root privileges.

The `web` and `db` containers are each ready with MySQL/PostgreSQL clients, so you can [`ddev ssh`](../usage/commands.md#ssh) or `ddev ssh -s db` and use `mysql` or `psql`.

## `mysqldump` and `pgdump`

The `web` and `db` containers come with `mysqldump`. You could run [`ddev ssh`](../usage/commands.md#ssh) to enter the web container, for example, then `mkdir /var/www/html/.tarballs` and run `mysqldump db >/var/www/html/.tarballs/db.sql` or run `mysqldump db | gzip >/var/www/html/.tarballs/db.sql.gz` to create database dumps. Because `/var/www/html` is mounted into the container from your project root, the `.tarballs` directory will also show up in the root of the project on your host machine.

The PostgreSQL database container includes normal `pg` commands like `pgdump`.

## Database GUIs

If you’d like to use a GUI database client, you’ll need the right connection details and there may even be a command to launch it for you:

* phpMyAdmin, formerly built into DDEV core, can be installed by running `ddev get ddev/ddev-phpmyadmin`.
* Adminer can be installed with `ddev get ddev/ddev-adminer`
* The [`ddev describe`](../usage/commands.md#describe) command displays the `Host:` details you’ll need to connect to the `db` container externally, for example if you're using an on-host database browser like SequelAce.
* macOS users can use `ddev sequelace` to launch the free [Sequel Ace](https://sequel-ace.com/) database browser, [`ddev tableplus`](../usage/commands.md#tableplus) to launch [TablePlus](https://tableplus.com), [`ddev querious`](../usage/commands.md#querious) for [Querious](https://www.araelium.com/querious), and the obsolete Sequel Pro is also supported with `ddev sequelpro`. (Each must be installed for the command to exist.)
* PhpStorm (and all JetBrains tools) have a nice database browser. (If you use the [DDEV Integration plugin](https://plugins.jetbrains.com/plugin/18813-ddev-integration) this is all done for you.)
    * Choose a static [`host_db_port`](../configuration/config.md#host_db_port) setting for your project. For example `host_db_port: 59002` (each project’s database port should be different if you’re running more than one project at a time). Use [`ddev start`](../usage/commands.md#start) for it to take effect.
    * Use the “database” tool to create a source from “localhost”, with the proper type “mysql” or “postgresql” and the port you chose, username `db` + password `db`.
    * Explore away!
* There’s a sample custom command that will run the free MySQL Workbench on macOS, Windows or Linux. To use it, run:
    * `cp ~/.ddev/commands/host/mysqlworkbench.example ~/.ddev/commands/host/mysqlworkbench`
    * `ddev mysqlworkbench`
