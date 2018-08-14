<h1>Using Developer Tools with ddev</h1>

## Developer Tools Included in the Container
We have included several useful developer tools in our containers. Run `ddev describe` to see the project information and services available for your project and how to access them.   

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

### Database Management

[phpMyAdmin](https://www.phpmyadmin.net/) is a free software tool to manage MySQL and MariaDB databases from a browser. phpMyAdmin comes installed with ddev. 

After your project is started, access the phpMyAdmin web interface at its default port:

```
http://mysite.ddev.local:8036
```

If you use the free [Sequel Pro](https://www.sequelpro.com/) database browser for macOS, run `ddev sequelpro` within a project folder, and Sequel Pro will launch and access the database for that project. 

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
If you have PHP and Drush installed on your host system, you can use it to interact with a ddev project. On the host system the extra include ddev_drush_settings.php is written on startup and will allow drush to access the database server. This may not work for all drush commands because of course the actual webserver environment is not available.