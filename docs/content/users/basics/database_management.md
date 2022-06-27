# Database Management

DDEV provides lots and lots of flexibility for you in managing your databases between your local development, staging and production environments. Most people know about `ddev import-db` and `ddev export-db` but those tools now have more flexibility and there are plenty of other adaptable ways to work with your databases.

Remember, you can run `ddev [command] --help` for more info on many of the topics below.

## Database Imports

Import a database with just one command; There is support for several file formats, including: **.sql, sql.gz, mysql, mysql.gz, tar, tar.gz, and zip**.

Here's an example of a database import using ddev:

```bash
ddev import-db --src=dumpfile.sql.gz
```

You can also:

* Use `ddev mysql` or `ddev psql` or the `mysql` and `psql` commands inside the web and db containers.
* Use phpMyAdmin for database imports, but that approach is much slower.

## Discussion

**Many database backends**: You can use a vast array of different database types, including MariaDB (5.5-10.7) and MySQL (5.5-8.0) Postgresql (9-14), see ([docs](../extend/database_types.md#database-server-types)). Note that if you want to _change_ database type, you need to export your database and then `ddev delete` the project (to kill off the existing database), make the change to a new db type, start again, and import.

**Default database**: DDEV creates a default database named `db` and default permissions for the `db` user with password `db`, and it's on the (inside Docker) hostname `db`.

**Extra databases**: In DDEV you can easily create and populate other databases as well. For example, `ddev import-db --target-db=backend --src=backend.sql.gz` will create the database named `backend` with permissions for that same `db` user and import from the `backend.sql.gz dumpfile`.

**Exporting extra databases**: You can export in the same way: `ddev export-db -f mysite.sql.gz` will export your default database (`db`). `ddev export-db --target-db=backend -f backend-export.sql.gz` will dump the database named `backend`.

**Database snapshots**: With _snapshots_ you can easily save the entire status of all of your databases. It's great for when you're working incrementally on migrations or updates and want to save state so you can start right back where you were.

I like to name my snapshots so I can find them later, so `ddev snapshot --name=two-dbs` would make a snapshot named `two-dbs` in the `.ddev/db_snapshots` directory. It includes the entire state of the db server, so in the case of our two databases above, both databases and the system level `mysql` or `postgres` database will all be snapshotted. Then if you want to delete everything with `ddev delete -O` (omitting the snapshot since we have one already), and then `ddev start` again, we can `ddev snapshot restore two-dbs` and we'll be right back where we were.

Don't forget about `ddev snapshot restore --latest` and that `ddev snapshot restore` will interactively let you choose among snapshots.

**ddev mysql** and **ddev psql**: `ddev mysql` gives you direct access to the mysql client in the db container. I like to use it for lots of things because I like the command line. I might just `ddev mysql` and give an interactive command like `DROP DATABASE backend;`. Or `SHOW TABLES;`. You can also do things like ``echo "SHOW TABLES;" | ddev mysql or `ddev mysql -uroot -proot` `` to get root privileges.

**mysql/psql clients in containers**: Both the web and db containers have the mysql/psql clients all set up and configured, so you can just `ddev ssh` or `ddev ssh -s db` and then use `mysql` or `psql` however you choose to.

**mysqldump**: The web and db containers also have `mysqldump` so you can use it any way you want inside there. I like to `ddev ssh` (into the web container) and then `mkdir /var/www/html/.tarballs` and `mysqldump db >/var/www/html/.tarballs/db.sql` or `mysqldump db | gzip >/var/www/html/.tarballs/db.sql.gz` (Because /var/www/html is mounted into the container from your project root, the .tarballs folder will also show up in the root of the project on the host.)

**pgdump and related commands**: The postgres db container has all the normal `pg` commands like `pgdump`.

**Other database explorers**: There are lots of alternatives for GUI database explorers:

* macOS users love `ddev sequelpro`, which launches the free Sequelpro database browser. However, it's gotten little love in recent years, so ddev now supports TablePlus and SequelAce if they're installed. `ddev tableplus` and `ddev sequelace`.
* `ddev describe` tells you the URL for the built-in PHPMyAdmin database browser (Hint: It's `http://<yourproject>.ddev.site:8036`).
* PHPStorm (and all JetBrains tools) have a nice database browser:
    * Choose a static `host_db_port` for your project. For example `host_db_port: 59002` (each project's db port should be different if you're running more than one project at a time). (`ddev start` to make it take effect)
    * Use the "database" tool to create a source from "localhost", with the proper type "mysql" or "postgresql" and the port you chose, credentials username: db and password: db
    * Explore away!
* There's a sample custom command that will run the free [mysqlworkbench](https://dev.mysql.com/downloads/workbench/) GUI database explorer on macOS, Windows or Linux. You just have to:
    * `cp ~.ddev/commands/host/mysqlworkbench.example ~.ddev/commands/host/mysqlworkbench`
    * and then `ddev mysqlworkbench`
