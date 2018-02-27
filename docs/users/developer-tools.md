<h1>Using Developer Tools with ddev</h1>

## Developer Tools Included in the Container
We have included several useful developer tools in our containers.

### Command-line Tools
- MySQL Client (mysql) - Command-line interface for interacting with MySQL.
- [Composer](https://getcomposer.org/) - Dependency Manager for PHP.
- [Drush](http://www.drush.org) - Command-line shell and Unix scripting interface for Drupal.
- [WP-CLI](http://wp-cli.org/) - Command-line tools for managing WordPress installations.

These tools can be accessed for single commands using [`ddev exec <command>`](cli-usage.md#executing-commands-in-containers) or [`ddev ssh`](cli-usage.md#ssh-into-containers) for an interactive bash session.

### Email Capture and Review

[MailHog](https://github.com/mailhog/MailHog) is a mail catcher which is configured to capture and display emails sent by PHP in the development environment.

After your project is started, access the MailHog web interface at its default port:

```
http://mysite.ddev.local:8025
```

Please note this will not intercept emails if your application is configured to use SMTP or a 3rd-party ESP integration. If you are using SMTP for outgoing mail handling ([Swiftmailer](https://www.drupal.org/project/swiftmailer) or [SMTP](https://www.drupal.org/project/smtp) modules for example), update your application configuration to use `localhost` on port `1025` as the SMTP server locally in order to use MailHog.

MailHog provides several [configuration options](https://github.com/mailhog/MailHog/blob/master/docs/CONFIG.md). If you need to alter its configuration, you can do so by adding the desired environment variable to the `environment` section for the web container in the `.ddev/docker-compose.yaml` for your project.

## Using Development Tools on the Host Machine

It is possible in many cases to use development tools installed on your host machine on a project provisioned by ddev. Tools that interact with files and require no database connection, such as Git or Composer, can be run from the host machine against the code base for a ddev project with no additional configuration necessary.

### Database Connections from the Host

If you need to connect to the database of your project from the host machine, run `ddev describe` to retrieve the database connection information. The last line of the database credentials will provide your host connection info, similar to this:

```
To connect to mysql from your host machine, use port 32838 on 127.0.0.1
For example: mysql --host 127.0.0.1 --port 32838
```

The port referenced is unique per running project, and randomly chosen from available ports on your system when you run `ddev start`.

**Note:** The host database port is likely to change any time a project is stopped/removed and then later started again.

### Using Drush Installation on Host Machine
If you have Drush installed on your host system, you can use it to interact with a ddev project by defining a `drush.settings.php` file at the docroot of your code base, and referencing it from your `settings.php` file. The `drush.settings.php` file should look similar to below, using the host port information for your project retrieved from `ddev describe`:

```
<?php

$databases['default']['default'] = array(
  'database' => "db",
  'username' => "db",
  'password' => "db",
  'host' => "127.0.0.1",
  'driver' => "mysql",
  'port' => <YOUR_PROJECT_DB_PORT>,
  'prefix' => "",
);
```

The following should be added to the `settings.php` file so that the `drush.settings.php` file is loaded when using Drush on your host machine:

```
// This determines whether or not drush should include a custom settings file which allows
// it to work both within a docker container and natively on the host system.
if (!empty($_SERVER["argv"]) && strpos($_SERVER["argv"][0], "drush") && empty($_ENV['DEPLOY_NAME'])) {
  include __DIR__ . '../../../drush.settings.php';
}
```

These configuration files are auto-generated for you if you run [`ddev import-db`](cli-usage.md#import-db) on a project with no existing settings file.
