# Managing CMS Settings

Any CMS-specific project type, meaning any of the non-generic [CMS Quickstarts](../../users/quickstart.md), has settings that DDEV manages to save you time and optimize configuration for local development.

Generally, DDEV will:

* Create a main settings file if none exists, like Drupal’s `settings.php`.
* Create a specialty config file with DDEV-specific settings, like `AdditionalSettings.php` for TYPO3 or `settings.ddev.php` for Drupal.
* Add an include of the specialty file if needed, like adding `settings.ddev.php` include to the bottom of Drupal’s `settings.php`.

While this reduces setup time for new users, makes it easier to try out a CMS, and speeds up project creation, you may still want to modify or override DDEV’s CMS-specific behavior.

## Controlling or Removing CMS Settings

There are several ways to back off DDEV’s CMS settings management:

1. **Take control of files by removing the `#ddev-generated` comment.**  
DDEV will automatically update any it’s added containing a `#ddev-generated` comment. This means you don’t need to touch that file, but also that any changes you make will be overwritten. As soon as you remove the comment, DDEV will ignore that file and leave you fully in control over it. (Don’t forget to check it into version control!)

    !!!tip "Reversing the change"
        If you change your mind and want DDEV to take over the file again, delete it and run [`ddev start`](../usage/commands.md#start). DDEV will recreate its own version, which you may want to remove from your Git project.

2. **Disable settings management.**  
You can tell DDEV to use a specific project type without creating settings files by either setting [`disable_settings_management`](../configuration/config.md#disable_settings_management) to `true` or running [`ddev config --disable-settings-management`](../configuration/config.md#type).

3. **Switch to the generic PHP project type.**  
If you don’t want DDEV’s CMS-specific settings, you can switch your project to the generic `php` type by editing [`type: php`](../configuration/config.md#type) in the project’s settings or running [`ddev config --project-type=php`](../usage/commands.md#config). DDEV will no longer create or tweak any settings files. You’ll lose any perks from the nginx configuration for the CMS, but you can always customize [nginx settings](../extend/customization-extendibility.md#custom-nginx-configuration) or [Apache settings](../extend/customization-extendibility.md#custom-apache-configuration) separately.

4. **Un-set the `$IS_DDEV_PROJECT` environment variable.**  
This environment variable is set `true` by default in DDEV’s environment, and can be used to fence off DDEV-specific behavior. When it’s empty, the important parts of `settings.ddev.php` and `AdditionalSettings.php` (for TYPO3) are not executed. This means that DDEV’s `settings.ddev.php` won’t be invoked if it somehow ends up in a production environment or in a non-DDEV local development environment.

!!!tip "Ignore `.ddev/.gitignore`"
    The `.ddev/.gitignore` file is created when you run `ddev start` and `disable_settings_management` is `false`. You should _not_ check this file in, since it ignores itself and DDEV’s temporary and automatically-managed files. This makes it easier for teams to share the `.ddev` folder via Git, even if the `.ddev/.gitignore` file changes with different versions.

## CMS-Specific Help and Techniques

### Drupal Specifics

#### Drupal Settings Files

By default, DDEV will create settings files for your project that work out of the box. It creates a `sites/default/settings.ddev.php` and adds an include in `sites/default/settings.php` to bring that in. There are guards to prevent the `settings.ddev.php` from being active when the project is not running under DDEV, but it still should not be checked in and is gitignored.

#### Database requirements for Drupal 9.5+

* Using MySQL or MariaDB, Drupal requires `SET GLOBAL TRANSACTION ISOLATION LEVEL READ COMMITTED` and DDEV does this for you on [`ddev start`](../usage/commands.md#start).
* Using PostgreSQL, Drupal requires the`pg_trm` extension. DDEV creates this extension automatically for you on `ddev start`.

#### Twig Debugging

With the default Drupal configuration, it’s very difficult to debug Twig templates; you need to use `development.services.yml` instead of `services.yml`. Add this line in your `settings.php` or `settings.local.php`. See discussion at [drupal.org](https://www.drupal.org/forum/support/module-development-and-code-questions/2019-09-02/ddev-twig-debugging) and the Drupal documentation.

```php
$settings['container_yamls'][] = DRUPAL_ROOT . '/sites/development.services.yml';
```

#### Multisite

1. Start with the [DDEV Drupal 8 Multisite Recipe](<https://github.com/ddev/ddev-contrib/tree/master/recipes/drupal8-multisite>).
2. Update configuration files.
    1. Update each `site/{site_name}/settings.php`:

        ```php
        /**
         * DDEV environments will have $databases (and other settings) set
         * by an auto-generated file. Make alterations here for this site
         * in a multisite environment.
         */
        elseif (getenv('IS_DDEV_PROJECT') == 'true') {
          /**
           * Alter database settings and credentials for DDEV environment.
           * Includes loading the DDEV-generated `default/settings.ddev.php`.
           */
          include $app_root . '/' . $site_path . '/settings.databases.ddev.inc';
        }
        ```

    2. Add a `settings.databases.ddev.inc` in each `site/{site_name}/`:

        ```php
        /**
         * Fetch DDEV-generated database credentials and other settings.
         */
        require $app_root . '/sites/default/settings.ddev.php';

        /**
         * Alter default database for this site. `settings.ddev.php` will have
         * “reset” this to 'db'.
         */
        $databases['default']['default']['database'] = 'site_name';
        ```

    3. Update your [`web_environment`](../configuration/config.md#web_environment) config option if you’re using site aliases:

        ```yaml
        web_environment:
          # Make DDEV Drush shell PIDs last for entire life of the container
          # so `ddev drush site:set @alias` persists for all Drush connections.
          # https://chrisfromredfin.dev/posts/drush-use-ddev/
          - DRUSH_SHELL_PID=PERMANENT
        ```

### TYPO3 Specifics

#### Settings Files

On [`ddev start`](../usage/commands.md#start), DDEV creates a `public/typo3conf/AdditionalConfiguration.php` file with database configuration in it.

#### Setup a Base Variant (since TYPO3 9.5)

Since TYPO3 9.5 you have to setup a `Site Configuration` for each site you like to serve. To be able to browse the site on your local environment, you have to set up a `Base Variant` in your `Site Configuration` depending on your local context. In this example we assume a `Application Context` `Development/DDEV` which can be set in the DDEV’s `config.yaml`:

```yaml
web_environment:
- TYPO3_CONTEXT=Development/DDEV
```

This variable will be available after the project start or restart.

Afterwards add a `Base Variant` to your `Site Configuration`:

```yaml
baseVariants:
  -
    base: 'https://example.com.ddev.site/'
    condition: 'applicationContext == "Development/DDEV"'
```

See also [TYPO3 Documentation](https://docs.typo3.org/m/typo3/reference-coreapi/main/en-us/ApiOverview/SiteHandling/BaseVariants.html).
