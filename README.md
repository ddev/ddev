[![CircleCI](https://circleci.com/gh/drud/ddev.svg?style=shield)](https://circleci.com/gh/drud/ddev) [![Go Report Card](https://goreportcard.com/badge/github.com/drud/ddev)](https://goreportcard.com/report/github.com/drud/ddev) ![project is maintained](https://img.shields.io/maintenance/yes/2017.svg)




# ddev

The purpose of *ddev* is to support developers with a local copy of a site for development purposes. It runs the site in Docker containers.

You can see all "ddev" usages using the help commands, like `ddev -h`, `ddev start -h`, etc.

## System Requirements
- MacOS Sierra or Linux*
- [docker](https://www.docker.com/community-edition)

* *Currently only tested on Ubuntu, please [let us know](https://github.com/drud/ddev/issues/new) if you hit an issue on another flavor.*

## Installation
### Installation Script
You can paste this line of code to your terminal to download, verify, and install ddev using our [install script](https://github.com/drud/ddev/blob/master/install_ddev.sh):
```
curl https://raw.githubusercontent.com/drud/ddev/master/install_ddev.sh | bash
```

### Manual Installation
You can also easily perform the installation manually if preferred:
- Download and extract the latest [ddev release](https://github.com/drud/ddev/releases) for your architecture.
- Make ddev executable: `chmod ugo+x ddev`
- Move ddev to /usr/local/bin: `mv ddev /usr/local/bin/` (may require sudo)
- Run `ddev` to test your installation. You should see usage output similar to below.

---

## Usage
```
➜  ddev
This Command Line Interface (CLI) gives you the ability to interact with the ddev to create a local development environment.

Usage:
  ddev [command]

Available Commands:
  config       Create or modify a ddev application config in the current directory
  describe     Get a detailed description of a running ddev site.
  exec         Execute a Linux shell command in the webserver container.
  hostname     Manage your hostfile entries.
  import-db    Import the database of an existing site to the local dev environment.
  import-files Import the uploaded files directory of an existing site to the default public upload directory of your application.
  list         List applications that exist locally
  logs         Get the logs from your running services.
  restart      Restart the local development environment for a site.
  rm           Remove an application's local services.
  sequelpro    Easily connect local site to sequelpro
  ssh          SSH to an app container.
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

## Listing sites

To see a list of your current sites you can use `ddev list`.

```
➜  ddev list
1 local site found.
NAME     TYPE     LOCATION                 URL                        STATUS
drupal8  drupal8  ~/Projects/ddev/drupal8  http://drupal8.ddev.local  running
```

You can also see more detailed information about a site by running `ddev describe` or `ddev describe [site-name]`.

```
NAME     TYPE     LOCATION                 URL                        STATUS
drupal8  drupal8  ~/Projects/ddev/drupal8  http://drupal8.ddev.local  running

MySQL Credentials
-----------------
Username:       	root
Password:       	root
Database name:  	data
Connection Info:	drupal8.ddev.local:3306

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

## Removing a site

You can remove a site by going to the working directory for the site and running `ddev rm`.

## Interacting with your Site
All of the commands can be performed by explicitly specifying the sitename or, to save time, you can execute commands from the site directory. All of the following examples assume you are in the working directory of your site.

### Retrieve Site Metadata
To view information about a specific site (such as URL, MySQL credentials, mailhog credentials), run `ddev describe` from within the working directory of the site. To view information for any site, use `ddev describe sitename`.

### Executing Commands
To run a command against your site use `ddev exec`. e.g. `ddev exec 'drush core-status'` would execute `drush core-status` against your site root. Commands ran in this way are executed in the webserver docroot. You are free to use any of [the tools included in the container](#tools-included-in-the-container).

### SSH Into The Container
The `ddev ssh` command will open a bash shell session to the web container of your site. You can also access the database container with `ddev ssh -s db`.

### Log Access
The `ddev logs` command allows you to easily retrieve error logs from the web server. To follow the webserver  error log (watch the lines in real time), run `ddev logs -f`. When you are done, press CTRL+C to exit from the log trail.

Additional logging can be accessed by using `ddev ssh` to manually retrieve the log files you are after. The web server stores access logs at `/var/log/nginx/access.log`, and PHP-FPM logs at `/var/log/php7.0-fpm.log`.

## Tools Included in the Container
We have included several useful tools for Developers in our containers.

### Command-line Tools
- [Composer](https://getcomposer.org/) - Dependency Manager for PHP
- [Drush](http://www.drush.org) - Command-line shell and Unix scripting interface for Drupal.
- [WP-CLI](http://wp-cli.org/) - Command-line tools for managing WordPress installations.

### Email
[MailHog](https://github.com/mailhog/MailHog) is a mail catcher we have installed and configured to catch emails sent by PHP.

Its web interface can be accessed at its default port after your site has been started. e.g.:
```
http://mysite.ddev.local:8025
```

Please note this will not intercept emails if your application is configured to use SMTP or a 3rd-party ESP integration. If you are using SMTP for outgoing mail handling ([Swiftmailer](https://www.drupal.org/project/swiftmailer) or [SMTP](https://www.drupal.org/project/smtp) modules for example), update your application configuration to use `localhost:1025` as the SMTP server locally in order to use MailHog.

MailHog provides several [configuration options](https://github.com/mailhog/MailHog/blob/master/docs/CONFIG.md). If you need to alter its configuration, you can do so by adding the desired environment variable to the `environment` section for the web container in the `.ddev/docker-compose.yaml` for your site.

### PHP Step-Debugging with an IDE and ddev site

Instructions for IDE setup are in [step-debugging](docs/step-debugging.md).

## Building

 ```
 make
 make linux
 make darwin
 make test
 make clean
 ```

 Note that although this git repository contains submodules (in the containers/ directory) they are not used in a normal build, but rather by the nightly build. You can safely ignore the git submodules and the containers/ directory.

## Testing
Normal test invocation is just `make test`. Run a single test with an invocation like `go test -v -run TestDevAddSites ./pkg/...`

If you set the environment variable DRUD_DEBUG=true you can see what ddev commands are being executed in the tests.
