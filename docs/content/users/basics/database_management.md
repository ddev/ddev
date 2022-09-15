# Database Management

DDEV provides lots of flexibility for managing your databases between your local, staging and production environments. Most people know about `ddev import-db` and `ddev export-db`, but those tools now have more flexibility and there are plenty of other adaptable ways to work with your databases.

!!!tip
    Remember, you can run `ddev [command] --help` for more info on many of the topics below.

## Database Imports

Import a database with one command, from one of the following file formats:  
**.sql, .sql.gz, .mysql, .mysql.gz, .tar, .tar.gz, and .zip**.

Here’s an example of a database import using DDEV:

```bash
ddev import-db --src=dumpfile.sql.gz
```

You can also:

* Use `ddev mysql` or `ddev psql` or the `mysql` and `psql` commands inside the `web` and `db` containers.
* Use phpMyAdmin for database imports—just be aware it’s much slower.

## Discussion

**Many database backends**: You can use a vast array of different database types, including MariaDB (5.5–10.7) and MySQL (5.5–8.0) PostgreSQL (9–14), see ([docs](../extend/database_types.md#database-server-types)). Note that if you want to _change_ database type, you need to export your database and then `ddev delete` the project (to kill off the existing database), make the change to a new database type, start again, and import.

**Default database**: DDEV creates a default database named `db` and default permissions for the `db` user with password `db`, and it’s on the (inside Docker) hostname `db`.

**Extra databases**: You can easily create and populate additional databases. For example, `ddev import-db --target-db=backend --src=backend.sql.gz` will create the database named `backend` with permissions for that same `db` user and import from the `backend.sql.gz dumpfile`.

**Exporting extra databases**: You can export in the same way: `ddev export-db -f mysite.sql.gz` will export your default database (`db`). `ddev export-db --target-db=backend -f backend-export.sql.gz` will dump the database named `backend`.

**Database snapshots**: Snapshots let you easily save the entire status of all of your databases, which can be great when you’re working incrementally on migrations or updates and want to save state so you can start right back where you were.

Snapshots can be named for easier reference later on. For example, `ddev snapshot --name=two-dbs` would make a snapshot named `two-dbs` in the `.ddev/db_snapshots` directory. It includes the entire state of the db server, so in the case of our two databases above, both databases and the system level `mysql` or `postgres` database will all be snapshotted. Then if you want to delete everything with `ddev delete -O` (omitting the snapshot since we have one already), and then `ddev start` again, we can `ddev snapshot restore two-dbs` and we’ll be right back where we were.

Use `ddev snapshot restore` to interactively choose among snapshots, or append `--latest` to restore the most recent snapshot: `ddev snapshot restore --latest`.

**ddev mysql** and **ddev psql**: `ddev mysql` gives you direct access to the MySQL and PostgreSQL clients in the database container, which can be useful for quickly running commands while you work. You might run `ddev mysql` to use interactive commands like `DROP DATABASE backend;` or `SHOW TABLES;`, or do things like `echo "SHOW TABLES;" | ddev mysql` or `ddev mysql -uroot -proot` to get root privileges.

**mysql/psql clients in containers**: The `web` and `db` containers are each ready with MySQL/PostgreSQL clients, so you can `ddev ssh` or `ddev ssh -s db` and use `mysql` or `psql`.

**mysqldump**: The `web` and `db` containers also come with `mysqldump`. You could `ddev ssh` into the web container, for example, then `mkdir /var/www/html/.tarballs` and `mysqldump db >/var/www/html/.tarballs/db.sql` or `mysqldump db | gzip >/var/www/html/.tarballs/db.sql.gz` to create database dumps. Because `/var/www/html` is mounted into the container from your project root, the `.tarballs` directory will also show up in the root of the project on your host machine.

**pgdump and related commands**: The PostgreSQL database container includes normal `pg` commands like `pgdump`.

**Other database explorers**: There are lots of alternatives for GUI database explorers:

* macOS users can use `ddev sequelpro` to launch the free [Sequel Pro](https://sequelpro.com) database browser, `ddev tableplus` to launch [TablePlus](https://tableplus.com), and `ddev sequelace` to launch [Sequel Ace](https://sequel-ace.com). (Each must be installed before running the command.)
* `ddev describe` displays the URL for the built-in phpMyAdmin GUI. (Something like `https://<yourproject>.ddev.site:8037`.)
* PhpStorm (and all JetBrains tools) have a nice database browser:
    * Choose a static `host_db_port` for your project. For example `host_db_port: 59002` (each project’s database port should be different if you’re running more than one project at a time). Use `ddev start` for it to take effect.
    * Use the “database” tool to create a source from “localhost”, with the proper type “mysql” or “postgresql” and the port you chose, username `db` + password `db`.
    * Explore away!
* There’s a sample custom command that will run the free [MySQL Workbench](https://dev.mysql.com/downloads/workbench/) on macOS, Windows or Linux. To use it, run:
    * `cp ~.ddev/commands/host/mysqlworkbench.example ~.ddev/commands/host/mysqlworkbench`
    * `ddev mysqlworkbench`
