<h1>Using the ddev command line interface (CLI)</h1>

Type `ddev` or `ddev -h`in a terminal windows to see the available ddev commands. There are commands to configure a project, start, stop, remove, describe, etc. Each command also has help. For example, `ddev describe -h`.


## Quickstart Guides

These are quickstart instructions WordPress, Drupal 7, Drupal 8, TYPO3, and Backdrop.

**Prerequisites:** Before you start [check the system requirements](../index.md#system-requirements) and follow the [installation instructions](../index.md#installation).

### WordPress Quickstart
To get started using ddev with a WordPress project clone the project's repository and checkout its directory.

```
git clone https://github.com/example-user/example-wordpress-site
cd example-wordpress-site
```

From here we can start setting up ddev. Inside your project's working directory, enter the command:

```
ddev config
```

_Note: ddev config will prompt you for a project name, docroot, and app type._

After you've run `ddev config`, you're ready to start running your project. To start running ddev, enter:

```
ddev start
```

When `ddev start` runs, it outputs status messages to indicate the project environment is starting. When startup is complete, ddev outputs a message like the one below with a link to access your project in a browser.

```
Successfully started example-wordpress-site
Your project can be reached at: http://example-wordpress-site.ddev.local
```

Quickstart instructions regarding database imports can be found under [Database Imports](#database-imports).

### Drupal 6/7 Quickstart

Beginning to use ddev with a Drupal 6 or 7 project is as simple as cloning the project's repository and checking out its directory.

```
git clone https://github.com/user/my-drupal-site
cd my-drupal-site
```

Now to start working with ddev. In your project's working directory, enter the following command:

```
ddev config
```

_Note: ddev config will prompt you for a project name, docroot, and project type._

After you've run `ddev config` you're ready to start running your project. Run ddev using:

```
ddev start
```

When `ddev start` runs, it outputs status messages to indicate the project environment is starting. When startup is complete, ddev outputs a message like the one below with a link to access your project in a browser.

```
Successfully started my-drupal7-site
Your project can be reached at: http://my-drupal7-site.ddev.local
```

Quickstart instructions for database imports can be found under [Database Imports](#database-imports).

### Drupal 8 Quickstart

Get started with Drupal 8 projects on ddev either by cloning a git repository or using a new or existing composer project.

**Git Clone Example**

```
git clone https://github.com/example-user/my-drupal8-site
cd my-drupal8-site
```

**Composer Setup Example**

```
composer create-project drupal-composer/drupal-project:8.x-dev my-drupal8-site --stability dev --no-interaction
cd my-drupal8-site
```

_Find more information on composer and how to use it [here](https://github.com/drupal-composer/drupal-project)._

The next step is to configure ddev. In your project's working directory, enter the following command:

```
ddev config
```

_Note: ddev config will prompt you for a project name, docroot, and app type._

After you've run `ddev config` you're ready to start up your project. Run ddev using:

```
ddev start
```

When `ddev start` runs, it outputs status messages to indicate the project environment is starting. When startup is complete, ddev outputs a message like the one below with a link to access your project in a browser.

```
Successfully started my-drupal8-site
Your project can be reached at: http://my-drupal8-site.ddev.local
```

### TYPO3 Quickstart

To get started using ddev with a TYPO3 project, clone the project's repository and checkout its directory.

```
git clone https://github.com/example-user/example-typo3-site
cd example-typo3-site
```

If necessary, run build steps that you may require, like `composer install` in the correct directory.

_Note: ddev assumes that the files created by a site install have already been created, including the typo3conf, typo3temp, uploads, and fileadmin directories._

From here we can start setting up ddev. In your project's working directory, enter the command:

```
ddev config
```

_Note: ddev config will prompt you for a project name, docroot, and project type._

After you've run `ddev config`, you're ready to start running your project. To start running ddev enter:

```
ddev start
```

When `ddev start` runs, it outputs status messages to indicate the project environment is starting. When startup is complete, ddev outputs a message like the one below with a link to access your project in a browser.

```
Successfully started example-typo3-site
Your application can be reached at: http://example-typo3-site.ddev.local
```

### Backdrop Quickstart

To get started with Backdrop, clone the project repository and navigate to the project directory.

```
git clone https://github.com/example-user/example-backdrop-site
cd example-backdrop-site
```

To set up ddev for your project, enter the command:

```
ddev config
```

_Note: ddev config will prompt you for a project name, docroot, and project type._

After you've run `ddev config`, you're ready to start running your project. To start running ddev enter:

```
ddev start
```

When `ddev start` runs, it outputs status messages to indicate the project environment is starting. When startup is complete, ddev outputs a message like the one below with a link to access your project in a browser.

```
Successfully started example-backdrop-site
Your application can be reached at: http://example-backdrop-site.ddev.local
```

### Database Imports

**Important:** Before importing any databases for your project, please remove its' wp-config.php if using WordPress - or settings.php file in the case of Drupal 7/8, if present.

_ddev will create a wp_config.php or settings.php file automatically if one does not exist. If you already have one you'll need to set the database credentials (user=db, password=db, host=db, database=db)._

Import a database with just one command; We offer support for several file formats, including: **.sql, sql.gz, tar, tar.gz, and zip**.

Here's an example of a database import using ddev:

```
ddev import-db --src=dumpfile.sql.gz
```

For in-depth application monitoring, use the `ddev describe` command to see details about the status of your ddev app.

**Note for Backdrop users:** In addition to importing a Backdrop database, you will need to extract a copy of your Backdrop site's configuration into the local `active` directory. The location for this directory can vary depending on the contents of your Backdrop `settings.php` file, but the default location is `[docroot]/files/config_[random letters and numbers]/active`. Please refer to the Backdrop documentation for more information on [moving your Backdrop site](https://backdropcms.org/user-guide/moving-backdrop-site) into the `ddev` environment.

## Getting Started

Check out the git repository for the project you want to work on. `cd` into the directory and run `ddev config` and follow the prompts.

```
$ cd ~/Projects
$ composer create-project drupal-composer/drupal-project:8.x-dev drupal8 --stability dev --no-interaction
$ cd drupal8
$ ddev config
Creating a new ddev project config in the current directory (/Users/username/Projects/drupal8)
Once completed, your configuration will be written to /Users/username/Projects/drupal8/.ddev/config.yaml


Project name (drupal8):

The docroot is the directory from which your site is served. This is a relative path from your application root (/Users/username/Projects/drupal8)
You may leave this value blank if your site files are in the application root
Docroot Location: web
Found a drupal8 codebase at /Users/username/Projects/drupal8/web
```

Configuration files have now been created for your project. (Take a look at the file on the project's .ddev/ddev.yaml file).

Now that the configuration has been created, you can start your project with `ddev start` (still from within the project working directory):

```
$ ddev start

Starting environment for drupal8...
Creating local-drupal8-db
Creating local-drupal8-web
Waiting for the environment to become ready. This may take a couple of minutes...
Successfully started drupal8
Your project can be reached at: http://drupal8.ddev.local
```

And you can now visit your working project. Enjoy!

## Listing project information

To see a list of your current projects you can use `ddev list`.

```
➜  ddev list
NAME     TYPE     LOCATION             URL(s)                      STATUS
drupal8  drupal8  ~/workspace/drupal8  http://drupal8.ddev.local   running
                                       https://drupal8.ddev.local
```

You can also see more detailed information about a project by running `ddev describe` from its working directory. You can also run `ddev describe [project-name]` from any location to see the detailed information for a running project.

```
NAME     TYPE     LOCATION             URL(s)                      STATUS
drupal8  drupal8  ~/workspace/drupal8  http://drupal8.ddev.local   running
                                       https://drupal8.ddev.local

Project Information
-----------------
PHP version:	7.0

MySQL Credentials
-----------------
Username:     	db
Password:     	db
Database name:	db
Host:         	db
Port:         	3306
To connect to mysql from your host machine, use port 32768 on 127.0.0.1.
For example: mysql --host=127.0.0.1 --port=32768 --user=db --password=db --database=db

Other Services
--------------
MailHog:   	http://drupal8.ddev.local:8025
phpMyAdmin:	http://drupal8.ddev.local:8036

DDEV ROUTER STATUS: healthy
```

## Importing assets for an existing project

An important aspect of local web development is the ability to have a precise recreation of the project you are working on locally, including up-to-date database contents and static assets such as uploaded images and files. ddev provides functionality to help with importing assets to your local environment with two commands.

### Importing a database

The `ddev import-db` command is provided for importing the MySQL database for a project. Running this command will provide a prompt for you to specify the location of your database import.

```
➜  ddev import-db
Provide the path to the database you wish to import.
Import path:
~/Downloads/db.sql
Importing database...
Generating settings.php file for database connection.
Successfully imported database for drupal8
```

A database connection file will be generated for your project if one does not exist (`settings.php` for Drupal, `wp-config.php` for WordPress). If you have already created a connection file, you will need to ensure your connection credentials match the ones provided in `ddev describe`.

<h4>Supported file types</h4>

Database import supports the following file types:

- Raw SQL Dump (.sql)
- Gzipped SQL Dump (.sql.gz)
- (Gzipped) Tarball Archive (.tar, .tar.gz, .tgz)
- Zip Archive (.zip)

If a Tarball Archive or Zip Archive is provided for the import, you will be provided an additional prompt, allowing you to specify a path within the archive to use for the import asset. The specified path should provide a Raw SQL Dump (.sql). In the following example, the database we want to import is named data.sql and resides at the top-level of the archive:

```
➜  ddev import-db
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

<h4>Non-interactive usage</h4>
If you want to use import-db without answering prompts, you can use the `--src` flag to provide the path to the import asset. If you are importing an archive, and wish to specify the path within the archive to extract, you can use the `--extract-path` flag in conjunction with the `--src` flag. Example:

`ddev import-db --src=/tmp/mydb.sql.gz`

### Importing static file assets
The `ddev import-files` command is provided for importing the static file assets for a project, such as uploaded images and documents. Running this command will provide a prompt for you to specify the location of your asset import. The assets will then be imported to the default public upload directory of the platform for the project. For Drupal projects, this is the "sites/default/files" directory. For WordPress projects, this is the "wp-content/uploads" directory.

```
➜  ddev import-files
Provide the path to the directory or archive you wish to import. Please note, if the destination directory exists, it will be replaced with the import assets specified here.
Import path:
~/Downloads/files.tar.gz
Successfully imported files for drupal8
```

<h4>Supported file types</h4>

Static asset import supports the following file types:

- A directory containing static assets
- (Gzipped) Tarball Archive (.tar, .tar.gz, .tgz)
- Zip Archive (.zip)

If a Tarball Archive or Zip Archive is provided for the import, you will be provided an additional prompt, allowing you to specify a path within the archive to use for the import asset. In the following example, the assets we want to import reside at "web/sites/default/files":

```
➜  ddev import-files
Provide the path to the directory or archive you wish to import. Please note, if the destination directory exists, it will be replaced with the import assets specified here.
Import path:
~/Downloads/site-backup.tar.gz
You provided an archive. Do you want to extract from a specific path in your archive? You may leave this blank if you wish to use the full archive contents
Archive extraction path:
web/sites/default/files
Successfully imported files for drupal8
```

<h4>Non-interactive usage</h4>
If you want to use import-files without answering prompts, you can use the `--src` flag to provide the path to the import asset. If you are importing an archive, and wish to specify the path within the archive to extract, you can use the `--extract-path` flag in conjunction with the `--src` flag. Example:

`ddev import-files --src=/tmp/files.tgz`

## Interacting with your project
ddev provides several commands to facilitate interacting with your project in the development environment. These commands can be run within the working directory of your project while the project is running in ddev.

### Executing Commands in Containers
The `ddev exec` command allows you to run shell commands in the container for a ddev service. By default, commands are executed on the web service container, in the docroot path of your project. This allows you to use [the developer tools included in the web container](developer-tools.md). For example, to run the Drush CLI in the web container, you would run `ddev exec drush status`.

To run a shell command in the container for a different service, use the `--service` flag at the beginning of your exec command to specify the service the command should be run against. For example, to run the mysql client in the database, container, you would run `ddev exec --service db mysql`.

Commands can also be executed using the shorter `ddev . <cmd>` alias.

### SSH Into Containers
The `ddev ssh` command will open an interactive bash shell session to the container for a ddev service. The web service is connected to by default. The session can be ended by typing `exit`. To connect to another service, use the `--service` flag to specify the service you want to connect to. For example, to connect to the database container, you would run `ddev ssh --service db`.

### Log Access
The `ddev logs` command allows you to easily retrieve error logs from the web server. To follow the web server error log (watch the lines in real time), run `ddev logs -f`. When you are done, press CTRL+C to exit from the log trail.

Additional logging can be accessed by using `ddev ssh` to manually review the log files you are after. The web server stores access logs at `/var/log/nginx/access.log`, and PHP-FPM logs at `/var/log/php7.0-fpm.log`.

## Stopping a project

Stop a project's containers without losing data by using `ddev stop` in the working directory of the project. You can also stop any running project's containers by providing the project name as an argument, e.g. `ddev stop <projectname>`.

## Removing a project

To remove a project's containers run `ddev remove` in the working directory of the project. To remove any running project's containers, providing the project name as an argument, e.g. `ddev remove <projectname>`.

`ddev remove` is *not* destructive. It removes the docker containers but does not remove the database for the project. This allows you to have many configured projects with databases loaded without wasting docker containers on unused projects. `ddev remove` does not affect the project code base and files.

To remove the imported database for a project, use `ddev remove --remove-data` instead of just `ddev remove`, and the database files will also be destroyed.

## ddev Command Auto-Completion

ddev bash auto-completion is available. If you have installed ddev via homebrew (on macOS) it will already be installed. Otherwise, download the [latest release](https://github.com/drud/ddev/releases) tarball for your platform and the ddev_bash_completions.sh inside it can be installed wherever your bash_completions.d is. For example, `cp ddev_bash_completions.sh /etc/bash_completion.d/ddev`
