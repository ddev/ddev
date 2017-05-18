<h1>Developer Tools Included in the Container</h1>
We have included several useful developer tools in our containers.

### Command-line Tools
- [Composer](https://getcomposer.org/) - Dependency Manager for PHP
- [Drush](http://www.drush.org) - Command-line shell and Unix scripting interface for Drupal.
- [WP-CLI](http://wp-cli.org/) - Command-line tools for managing WordPress installations.

### Email Catching
[MailHog](https://github.com/mailhog/MailHog) is a mail catcher which is configured to capture and display emails sent by PHP in the development environment.

Its web interface can be accessed at its default port after your site has been started. e.g.:
```
http://mysite.ddev.local:8025
```

Please note this will not intercept emails if your application is configured to use SMTP or a 3rd-party ESP integration. If you are using SMTP for outgoing mail handling ([Swiftmailer](https://www.drupal.org/project/swiftmailer) or [SMTP](https://www.drupal.org/project/smtp) modules for example), update your application configuration to use `localhost` on port `1025` as the SMTP server locally in order to use MailHog.

MailHog provides several [configuration options](https://github.com/mailhog/MailHog/blob/master/docs/CONFIG.md). If you need to alter its configuration, you can do so by adding the desired environment variable to the `environment` section for the web container in the `.ddev/docker-compose.yaml` for your site.
