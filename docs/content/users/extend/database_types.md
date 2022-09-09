# Database Server Types

DDEV supports most versions of MariaDB, MySQL, and Postgresql database servers.

The default database type is MariaDB, and the default version is currently 10.3, but you can use nearly any MariaDB versions 5.5-10.7  MySQL 5.5-8.0), and Postgres 9-14. For example, you can use `ddev config --database=mysql:5.7`, `ddev config --database=mariadb:10.6`, `ddev config --database=postgres:14`.

In the config.yaml, either any of these, for example:

```yaml
database: 
  type: mariadb
  version: 10.6
```

## Checking the existing database and/or migrating

Since the existing binary database may not be compatible with changes to your configuration, you need to check and/or migrate your database.

* `ddev debug get-volume-db-version` will show the current binary database type.
* `ddev debug check-db-match` will show if your configured project matches the binary database type.
* `ddev debug migrate-database` allows an automated attempt at migrating your database to a different type/version.
    * This only works with databases of type `mysql` or `mariadb`.
    * It often can't work to migrate *from* databases of type `mysql:8.0` because dumps of that type most often can't be easily imported into all other database types.
    * Examples: `ddev debug migrate-database mariadb:10.7`, `ddev debug migrate-database mysql:8.0`

## Caveats

* If you change the database type or version in an existing project, the existing database will not be compatible with your change, so you'll want to use `ddev export-db` to save a dump first.
* When you change database type, destroy the existing database using `ddev delete --omit-snapshot` before changing, then after `ddev start` use `ddev import-db` to import the db you exported.
* Snapshots are always per database type and database version. So if you have snapshots from MariaDB 10.2 and you switch to MariaDB 10.5, don't expect to be able to restore the old snapshot.
