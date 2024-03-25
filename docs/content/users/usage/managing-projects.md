## Configuration Files

The [`ddev config`](../usage/commands.md#config) and `ddev start` commands attempt to create a CMS-specific settings file pre-populated with DDEV credentials. If you don't want DDEV to do this, set the [`disable_settings_management`](../configuration/config.md#disable_settings_management) config option to `true`.

For **Contao** DDEV settings are added to the `.env.local` file.

For **Craft CMS** DDEV settings are added to the `.env` file.

For **Django 4** DDEV settings are placed in `.ddev/settings/settings.django4.py` and a stanza is added to your `settings.py` that is only invoked in DDEV context.

For **Drupal** and **Backdrop**, DDEV settings are written to a DDEV-managed file, `settings.ddev.php`. The `ddev config` command will ensure these settings are included in your `settings.php` through the following steps:

- Write DDEV settings to `settings.ddev.php`.
- If no `settings.php` file exists, create one that includes `settings.ddev.php`.
- If a `settings.php` file already exists, ensure that it includes `settings.ddev.php`, modifying `settings.php` to write the include if necessary.

For **Magento 1**, DDEV settings go into `app/etc/local.xml`.

In **Magento 2**, DDEV settings go into `app/etc/env.php`.

For **TYPO3**, DDEV settings are written to `AdditionalConfiguration.php`. If `AdditionalConfiguration.php` exists and is not managed by DDEV, it will not be modified.

For **WordPress**, DDEV settings are written to a DDEV-managed file, `wp-config-ddev.php`. The `ddev config` command will attempt to write settings through the following steps:

- Write DDEV settings to `wp-config-ddev.php`.
- If no `wp-config.php` exists, create one that include `wp-config-ddev.php`.
- If a DDEV-managed `wp-config.php` exists, create one that includes `wp-config.php`.
- If a user-managed `wp-config.php` exists, instruct the user on how to modify it to include DDEV settings.

You’ll know DDEV is managing a settings file when you see a comment containing `#ddev-generated` like the one below. Remove the comment and DDEV will not attempt to overwrite it. If you’re letting DDEV create its settings file, we recommended leaving this comment so DDEV can continue to manage it, and make any needed changes in another settings file.

```
/**
 #ddev-generated: Automatically generated Drupal settings.php file.
 ddev manages this file and may delete or overwrite the file unless this comment is removed.
 */
```

If you’re providing the `settings.php` or `wp-config.php` and DDEV is creating `settings.ddev.php` (or `wp-config-local.php`, `AdditionalConfig.php`, or similar), the main settings file must explicitly include the appropriate DDEV-generated settings file. Any changes you need should be included somewhere that loads after DDEV’s settings file, for example in Drupal’s `settings.php` _after_ `settings.ddev.php` is included. (See [Adding Configuration](#adding-configuration) below).

### Adding Configuration

**Drupal and Backdrop**: In `settings.php`, enable loading `settings.local.php` after `settings.ddev.php` is included—creating a new one if it doesn’t already exist—and make changes there. Wrap with `if (getenv('IS_DDEV_PROJECT') == 'true')` as needed.

**WordPress**: Load a `wp-config-local.php` after `wp-config-ddev.php`, and make changes there. Wrap with `if (getenv('IS_DDEV_PROJECT') == 'true')` as needed.

## Listing Project Information

Run [`ddev list`](../usage/commands.md#list) or `ddev list --active-only` current projects.

```

➜  ddev list
NAME          TYPE     LOCATION                   URL(s)                                STATUS
d8git         drupal8  ~/workspace/d8git          <https://d8git.ddev.local>            running
                                                  <http://d8git.ddev.local>
hobobiker     drupal6  ~/workspace/hobobiker.com                                        stopped

```

```

➜  ddev list --active-only
NAME     TYPE     LOCATION             URL(s)                      STATUS
drupal8  drupal8  ~/workspace/drupal8  <http://drupal8.ddev.site>  running
                                       <https://drupal8.ddev.site>

```

You can also see more detailed information about a project by running [`ddev describe`](../usage/commands.md#describe) from its working directory. You can also run `ddev describe [project-name]` from any location to see the detailed information for a running project.

```
NAME        TYPE     LOCATION                URL                           STATUS
d9composer  drupal8  ~/workspace/d9composer  https://d9composer.ddev.site  running

Project Information
-------------------
PHP version:    7.4
MariaDB version 10.3

URLs
----
https://d9composer.ddev.site
https://127.0.0.1:33232
http://d9composer.ddev.site
http://127.0.0.1:33233

MySQL/MariaDB Credentials
-------------------------
Username: "db", Password: "db", Default database: "db"

or use root credentials when needed: Username: "root", Password: "root"

Database hostname and port INSIDE container: ddev-d9-db:3306
To connect to db server inside container or in project settings files:
mysql --host=ddev-d9-dbcomposer --user=db --password=db --database=db
Database hostname and port from HOST: 127.0.0.1:33231
To connect to mysql from your host machine,
mysql --host=127.0.0.1 --port=33231 --user=db --password=db --database=db

Other Services
--------------
Mailpit (HTTPS):    https://d9composer.ddev.site:8026
Mailpit:            http://d9composer.ddev.site:8025

DDEV ROUTER STATUS: healthy
ssh-auth status: healthy
```

## Removing Projects

There are two ways to remove a project from DDEV’s listing.

The first, the [`ddev delete`](../usage/commands.md#delete) command, is destructive. It removes the project from DDEV’s list, deletes its database, and removes the hostname entry from the hosts file:

`ddev delete <projectname>`
or
`ddev delete --omit-snapshot <projectname>`

If you don’t want the project to show up in [`ddev list`](../usage/commands.md#list) anymore, use [`ddev stop`](../usage/commands.md#stop)—which is nondestructive—to unlist the project until the next time you run [`ddev start`](../usage/commands.md#start) or [`ddev config`](../usage/commands.md#config):

```bash
ddev stop --unlist <projectname>
```

## Importing Assets for An Existing Project

An important aspect of local web development is the ability to have a precise local recreation of the project you’re working on, including up-to-date database contents and static assets like uploaded images and files. DDEV provides two commands to help with importing assets to your local environment.

### Importing a Database

The [`ddev import-db`](../usage/commands.md#import-db) command imports the database for a project. Running this command will prompt you to specify the location of your database import. By default `ddev import-db` empties the default `db` database, then loads the provided dump file. Most people use it with command flags, like `ddev import-db --file=.tarballs/db.sql.gz`, but it can also prompt for the location of the dump if you only use `ddev import-db`:

```bash
ddev import-db
Provide the path to the database you wish to import.
Import path:
~/Downloads/db.sql
Importing database...
Successfully imported database for drupal8
```

#### Supported Database Import File Types

Database imports can be any of the following file types:

- Raw SQL Dump (`.sql`)
- Gzipped SQL Dump (`.sql.gz`)
- Xz’d SQL Dump (`.sql.xz`)
- (Gzipped) Tarball Archive (`.tar`, `.tar.gz`, `.tgz`)
- ZIP Archive (`.zip`)
- stdin

If a Tarball Archive or ZIP Archive is provided for the import, you’ll be prompted to specify a path within the archive to use for the import asset. The specified path should provide a raw SQL dump (`.sql`). In the following example, the database we want to import is named `data.sql` and resides at the top level of the archive:

```bash
ddev import-db
Provide the path to the database you wish to import.
Import path:
~/Downloads/site-backup.tar.gz
You provided an archive. Do you want to extract from a specific path in your archive? You may leave this blank if you wish to use the full archive contents
Archive extraction path:
data.sql
Importing database...
A settings file already exists for your application, so ddev did not generate one.
Run 'ddev describe' to find the database credentials for this application.
Successfully imported database for drupal8
```

#### Non-Interactive Usage

If you want to use the [`import-db`](../usage/commands.md#import-db) command without answering prompts, you can use the `--file` flag to provide the path to the import asset. If you’re importing an archive and wish to specify the path within the archive to extract, you can use the `--extract-path` flag in conjunction with the `--file` flag. Examples:

```bash
ddev import-db --file=/tmp/mydb.sql.gz
gzip -dc /tmp/mydb.sql.gz | ddev import-db
ddev import-db <mydb.sql
```

#### Database Import Notes

- Importing from a dump file via stdin will not show progress because there’s no way the import can know how far along through the import it has progressed.
- Use `ddev import-db --target-db <some_database>` to import to a non-default database (other than the default `db` database). This will create the database if it doesn’t already exist.
- Use `ddev import-db --no-drop` to import without first emptying the database.
- If a database already exists and the import does not specify dropping tables, the contents of the imported dumpfile will be _added_ to the database. Most full database dumps do a table drop and create before loading, but if yours does not, you can drop all tables with `ddev stop --remove-data` before importing.
- If imports are stalling or failing, make sure you have plenty of unused space (see [#3360](https://github.com/ddev/ddev/issues/3360)). DDEV has no problems importing large (2G+) databases, but importing requires lots of space. DDEV will show a warning on startup if unused space is getting low.
