<h1>Using the ddev command line interface (CLI)</h1>

Type `ddev` in a terminal to see the available ddev commands:

```
➜  ddev
This Command Line Interface (CLI) gives you the ability to interact with the ddev to create a local development environment.

Usage:
  ddev [command]

Available Commands:
  config       Create or modify a ddev application config in the current directory
  describe     Get a detailed description of a running ddev site.
  exec         Execute a shell command in the container for a service. Uses the web service by default.
  hostname     Manage your hostfile entries.
  import-db    Import the database of an existing site to the local dev environment.
  import-files Import the uploaded files directory of an existing site to the default public upload directory of your application.
  list         List applications that exist locally
  logs         Get the logs from your running services.
  restart      Restart the local development environment for a site.
  remove       Remove an application's local services.
  sequelpro    Easily connect local site to sequelpro
  ssh          Starts a shell session in the container for a service. Uses the web service by default.
  start        Start the local development environment for a site.
  stop         Stop an application's local services.
  version      print ddev version and component versions

Use "ddev [command] --help" for more information about a command.
```

## Getting Started
Check out the git repository for the site you want to work on. `cd` into the directory and run `ddev config` and follow the prompts.

```
$ cd ~/Projects
$ git clone git@github.com:drud/drupal8.git
$ cd drupal8
$ ddev config
Creating a new ddev project config in the current directory (/Users/username/Projects/drupal8)
Once completed, your configuration will be written to /Users/username/Projects/drupal8/.ddev/config.yaml


Project name (drupal8):

The docroot is the directory from which your site is served. This is a relative path from your application root (/Users/username/Projects/drupal8)
You may leave this value blank if your site files are in the application root
Docroot Location: docroot
Found a drupal8 codebase at /Users/username/Projects/drupal8/docroot
```

Configuration files have now been created for your site. (Available for inspection/modification at .ddev/ddev.yaml).
Now that the configuration has been created, you can start your site with `ddev start` (still from within the project working directory):

```
$ ddev start

Starting environment for drupal8...
Creating local-drupal8-db
Creating local-drupal8-web
Waiting for the environment to become ready. This may take a couple of minutes...
Successfully started drupal8
Your application can be reached at: http://drupal8.ddev.local
```

And you can now visit your working site. Enjoy!

## Listing site information

To see a list of your current sites you can use `ddev list`.

```
➜  ddev list
1 local site found.
NAME     TYPE     LOCATION                 URL                        STATUS
drupal8  drupal8  ~/Projects/ddev/drupal8  http://drupal8.ddev.local  running
```

You can also see more detailed information about a site by running `ddev describe` from its working directory. You can also run `ddev describe [site-name]` from any location to see the detailed information for a running site.

```
NAME     TYPE     LOCATION                 URL                        STATUS
drupal8  drupal8  ~/Projects/ddev/drupal8  http://drupal8.ddev.local  running

MySQL Credentials
-----------------
Username:     	db
Password:     	db
Database name:	db
Host:         	db
Port:         	3306
To connect to mysql from your host machine, use port 32894 on 127.0.0.1
For example: mysql --host 127.0.0.1 --port 32894

Other Services
--------------
MailHog:   	http://drupal8.ddev.local:8025
phpMyAdmin:	http://drupal8.ddev.local:8036
```

## Importing assets for an existing site
An important aspect of local web development is the ability to have a precise recreation of the site you are working on locally, including up-to-date database contents and static assets such as uploaded images and files. ddev provides functionality to help with importing assets to your local environment with two commands.

### Importing a database
The `ddev import-db` command is provided for importing the MySQL database for a site. Running this command will provide a prompt for you to specify the location of your database import.

```
➜  ddev import-db
Provide the path to the database you wish to import.
Import path:
~/Downloads/db.sql
Importing database...
Generating settings.php file for database connection.
Successfully imported database for drupal8
```

A database connection file will be generated for your site if one does not exist (`settings.php` for Drupal, `wp-config.php` for WordPress). If you have already created a connection file, you will need to ensure your connection credentials match the ones provided in `ddev describe`.

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
If you want to use import-db without answering prompts, you can use the `--src` flag to provide the path to the import asset. If you are importing an archive, and wish to specify the path within the archive to extract, you can use the `--extract-path` flag in conjunction with the `--src` flag.

### Importing static file assets
The `ddev import-files` command is provided for importing the static file assets for a site, such as uploaded images and documents. Running this command will provide a prompt for you to specify the location of your asset import. The assets will then be imported to the default public upload directory of the platform for the site. For Drupal sites, this is the "sites/default/files" directory. For WordPress sites, this is the "wp-content/uploads" directory. 

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

If a Tarball Archive or Zip Archive is provided for the import, you will be provided an additional prompt, allowing you to specify a path within the archive to use for the import asset. In the following example, the assets we want to import reside at "docroot/sites/default/files":

```
➜  ddev import-files
Provide the path to the directory or archive you wish to import. Please note, if the destination directory exists, it will be replaced with the import assets specified here.
Import path:
~/Downloads/site-backup.tar.gz
You provided an archive. Do you want to extract from a specific path in your archive? You may leave this blank if you wish to use the full archive contents
Archive extraction path:
docroot/sites/default/files
Successfully imported files for drupal8
```

<h4>Non-interactive usage</h4>
If you want to use import-files without answering prompts, you can use the `--src` flag to provide the path to the import asset. If you are importing an archive, and wish to specify the path within the archive to extract, you can use the `--extract-path` flag in conjunction with the `--src` flag.

## Interacting with your Site
ddev provides several commands to facilitate interacting with your site in the development environment. These commands can be run within the working directory of your project while the site is running in ddev. 

### Executing Commands in Containers
The `ddev exec` command allows you to run shell commands in the container for a ddev service. By default, commands are executed on the web service container, in the docroot path of your site. This allows you to use [the developer tools included in the web container](developer-tools.md). For example, to run the Drush CLI in the web container, you would run `ddev exec drush status`.

To run a shell command in the container for a different service, use the `--service` flag at the beginning of your exec command to specify the service the command should be run against. For example, to run the mysql client in the database, container, you would run `ddev exec --service db mysql`.

Commands can also be executed using the shorter `ddev . <cmd>` alias.

### SSH Into Containers
The `ddev ssh` command will open an interactive bash shell session to the container for a ddev service. The web service is connected to by default. The session can be ended by typing `exit`. To connect to another service, use the `--service` flag to specify the service you want to connect to. For example, to connect to the database container, you would run `ddev ssh --service db`.

### Log Access
The `ddev logs` command allows you to easily retrieve error logs from the web server. To follow the web server error log (watch the lines in real time), run `ddev logs -f`. When you are done, press CTRL+C to exit from the log trail.

Additional logging can be accessed by using `ddev ssh` to manually review the log files you are after. The web server stores access logs at `/var/log/nginx/access.log`, and PHP-FPM logs at `/var/log/php7.0-fpm.log`.

## Stopping a site
You can stop a site's containers without losing data by using `ddev stop` in the working directory of the site. You can also stop any running site's containers by providing the site name as an argument, e.g. `ddev stop <sitename>`.

## Removing a site
You can remove a site's containers by running `ddev remove` in the working directory of the site. You can also remove any running site's containers by providing the site name as an argument, e.g. `ddev remove <sitename>`. **Note:** `ddev remove` is destructive. It will remove all containers for the site, destroying database contents in the process. Your project code base and files will not be affected.

## ddev Command Auto-Completion
ddev bash auto-completion is available. If you have installed ddev via homebrew (on OSX) it will already be installed. Otherwise, you can download the [latest release](https://github.com/drud/ddev/releases) tarball for your platform and the ddev_bash_completions.sh inside it can be installed wherever your bash_completions.d is. For example, `cp ddev_bash_completions.sh /etc/bash_completion.d/ddev`