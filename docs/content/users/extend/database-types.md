# Database Server Types

DDEV supports many versions of the MariaDB, MySQL, and PostgreSQL database servers.

The default database type is MariaDB, and the default version is currently 10.4, but you can use MariaDB versions 5.5-10.8 and 10.11, MySQL 5.5-8.0, and Postgres 9-15. (New LTS versions of each of these are typically added soon after release. The very old versions are kept for compatibility with older projects.)

You could set these using the [`ddev config`](../usage/commands.md#config) command like this:

- `ddev config --database=mysql:5.7`
- `ddev config --database=mariadb:10.11`
- `ddev config --database=postgres:14`.

Or by editing the [`database`](../configuration/config.md#database) setting in `.ddev/config.yaml`:

```yaml
database:
  type: mysql
  version: 5.7
```

```yaml
database:
  type: mariadb
  version: 10.11
```

```yaml
database:
  type: postgres
  version: 14
```

## Checking the Existing Database and/or Migrating

Since the existing binary database may not be compatible with changes to your configuration, you need to check and/or migrate your database.

- [`ddev debug get-volume-db-version`](../usage/commands.md#debug-get-volume-db-version) will show the current binary database type.
- [`ddev debug check-db-match`](../usage/commands.md#debug-check-db-match) will show if your configured project matches the binary database type.
- [`ddev debug migrate-database`](../usage/commands.md#debug-migrate-database) allows an automated attempt at migrating your database to a different type/version.
    - This only works with databases of type `mysql` or `mariadb`.
    - MySQL 8.0 has diverged in syntax from most of its predecessors, including earlier MySQL and all MariaDB versions. As a result, you may not be able to migrated *from* databases of type `mysql:8.0` because dumps from MySQL 8.0 often have keywords or other features not supported elsewhere.
    - Examples: `ddev debug migrate-database mariadb:10.7`, `ddev debug migrate-database mysql:8.0`.

## Caveats

- If you change the database type or version in an existing project, the existing database will not be compatible with your change, so you’ll want to use [`ddev export-db`](../usage/commands.md#export-db) to save a dump first.
- When you change database type, destroy the existing database using [`ddev delete --omit-snapshot`](../usage/commands.md#delete) before changing, then after [`ddev start`](../usage/commands.md#start) use [`ddev import-db`](../usage/commands.md#import-db) to import the dump you saved.
- Snapshots are always per database type and database version. So if you have snapshots from MariaDB 10.2 and you switch to MariaDB 10.5, don’t expect to be able to restore the old snapshot.
