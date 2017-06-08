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

## Importing an existing site
Two commands are provided for importing assets from an existing site, `ddev import-db` and `ddev import-files`. Running either of these commands will provide a prompt to enter the location of the assets to import. You can also skip the prompt by specifying the location using the `--src` flag.

### import-db

```
➜  ddev import-db
Provide the path to the database you wish to import.
Import path:
~/Downloads/db.sql
Importing database...
Generating settings.php file for database connection.
Successfully imported database for drupal8
```

The `import-db` command allows you to specify the location of a SQL dump to be imported as the active database for your site. The database may be provided as a `.sql` file, `.sql.gz` or tar archive. The provided dump will be imported into the database named `data` in the database container for your site. A database connection file will be generated for your site if one does not exist (`settings.php` for Drupal, `wp-config.php` for WordPress). If you have already created a connection file, you will need to ensure your connection credentials match the ones provided in `ddev describe`.


### import-files

```
➜  ddev import-files
Provide the path to the directory or archive you wish to import. Please note, if the destination directory exists, it will be replaced with the import assets specified here.
Import path:
~/Downloads/files.tar.gz
Successfully imported files for drupal8
```

The `import-files` command allows you to specify the location of uploaded file assets to import for your site. For Drupal, this is the public files directory, located at `sites/default/files` by default. For WordPress, this is the uploads directory, located at `wp-content/uploads` by default. The files may be provided as a directory or tar archive containing the contents of the uploads folder. The contents of the directory or archive provided will be copied to the default location of the upload directory for your site.

## Interacting with your Site
ddev provides several commands to facilitate interacting with your site in the development environment. These commands can be run within the working directory of your project while the site is running in ddev. 

### Executing Commands in Containers
The `ddev exec` command allows you to run shell commands in the container for a ddev service. By default, commands are executed on the web service container, in the docroot path of your site. This allows you to use [the developer tools included in the web container](developer-tools.md). For example, to run the Drush CLI in the web container, you would run `ddev exec drush status`.

To run a shell command in the container for a different service, use the `--service` flag at the beginning of your exec command to specify the service the command should be run against. For example, to run the mysql client in the database, container, you would run `ddev exec --service db mysql`.

Commands can also be executed using the shorter `ddev . <cmd>` alias.

### SSH Into Containers
The `ddev ssh` command will open an interactive bash shell session to the container for a ddev service. The web service is connected to by default. The session can be ended by typing `exit`. To connect to another service, use the `--service` flag to specify the service you want to connect to. For example, to connect to the database container, you would run `ddev ssh --service db`.

### Log Access
The `ddev logs` command allows you to easily retrieve error logs from the web server. To follow the webserver  error log (watch the lines in real time), run `ddev logs -f`. When you are done, press CTRL+C to exit from the log trail.

Additional logging can be accessed by using `ddev ssh` to manually review the log files you are after. The web server stores access logs at `/var/log/nginx/access.log`, and PHP-FPM logs at `/var/log/php7.0-fpm.log`.

## Removing a site
You can remove a site by going to the working directory for the site and running `ddev remove`. `ddev remove` destroys all data and containers associated with the site.
