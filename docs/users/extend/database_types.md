## Database Server Types

DDEV-Local supports most versions of MariaDB and MySQL database servers, but of course the two types are mutually exclusive.

The default database type is MariaDB, and the default version is currently 10.2, but you can use nearly any MariaDB version (5.5 through 10.5) and nearly any MySQL version (5.5 through 8.0). For example, you can use `ddev config --mariadb-version="" --mysql-version=5.7` to configure for MySQL 5.7.

In the config.yaml, either `mysql_version` or `mariadb_version` should be left blank, for example:

```yaml
mysql_version: ""
mariadb_version: 10.5
```

or

```yaml
mariadb_version: ""
mysql_version: 8.0
```

### Caveats

* If you change the database type or version in an existing project, the existing database may not be compatible with your change, so you'll want to use `ddev export-db` to save a dump first.
* When you change database type, it's best to destroy the existing database using `ddev stop --remove-data` before changing, then use `ddev import-db` to import the db you already exported.
* Despite those warnings, the db container will do its best to upgrade if you're upgrading from compatible versions. For example, a change from MariaDB 10.1 to 10.4 is likely to work and upgrade correctly, but your mileage may vary. If you change from MySQL 8.0 to MariaDB 5.5, all hell will break loose and you'll have to `ddev stop --remove-data --omit-snapshot` to get back where you were.
* Snapshots are always per database type and database version. So if you have snapshots from MariaDB 10.2 and you switch to MariaDB 10.5, don't expect to be able to restore the old snapshot.
