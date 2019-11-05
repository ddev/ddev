## Database Server Types

ddev can currently support most versions of MariaDB and MySQL database servers. You can configure the type that you want using the .ddev/config.yaml file:

`mariadb_version: 10.4`
`mysql_version: 8.0`

The default version is MariaDB 10.2 and it works for most purposes.

You can also use `ddev config --mariadb-version=10.4` or `ddev config --mysql-version=8.0`

### Caveats

* If you change the database type or version, the existing database may not be compatible with your change, so you'll want to use `ddev export-db` to save a dump first. 
* When you change database type, it's best to destroy the existing database using `ddev stop --remove-data` before changing, then use `ddev import-db` to import the db you already exported.
* Despite those warnings, the db container will do its best to upgrade if you're upgrading from compatible versions. For example, a change from MariaDB 10.1 to 10.4 is likely to work and upgrade correctly, but your mileage may vary. If you change from MySQL 8.0 to MariaDB 5.5, all hell will break loose and you'll have to `ddev stop --remove-data --omit-snapshot` to get back where you were.
* Snapshots are always per-database-type. So if you have snapshots from MariaDB 10.2 and you switch to MariaDB 10.1, don't expect to be able to restore them.
