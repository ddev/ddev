# CMS Quickstarts

Once DDEV is installed, you can quickly spin up new projects:

1. Clone or create the code for your project.
2. `cd` into the project directory and run [`ddev config`](../users/usage/commands.md#config) to initialize a DDEV project.  
    It automatically detects your project type and docroot—make sure it’s accurate!
3. Run [`ddev start`](../users/usage/commands.md#start) to spin up the project.  
    If your project needs it, don’t forget to run [`ddev composer install`](../users/usage/commands.md#composer).
4. Import a database with [`ddev import-db`](../users/usage/commands.md#import-db).
5. Optionally import user-managed files with [`ddev import-files`](../users/usage/commands.md#import-files).
6. Run [`ddev launch`](../users/usage/commands.md#launch) to open your project in a browser, or visit the URL given by [`ddev start`](../users/usage/commands.md#start).

!!!tip
    While you’re getting your bearings, use [`ddev describe`](../users/usage/commands.md#describe) to get project details, and [`ddev help`](../users/usage/commands.md#help) to investigate commands.

DDEV comes ready to work with any PHP project, and has deeper support for several common PHP platforms and content management systems.

=== "Generic"

    ## Generic

    The `php` project type is the most general, ready for whatever modern PHP or static HTML/JS project you might be working on. It’s just as full-featured as more specific options, just without any app-specific configuration or presets.

    You may even prefer to stick with this flavor despite using one of the apps DDEV supports, simply because you’d rather configure things to your own liking. Please do!
    
    1. Create a directory (`mkdir my-new-project`) or clone your project (`git clone <your_project>`).
    2. Change to the new directory (`cd my-new-project`).
    3. Run [`ddev config`](../users/usage/commands.md#config) and set the project type and docroot, which are usually auto-detected, but may not be if there's no code in there yet.
    4. Run [`ddev start`](../users/usage/commands.md#start).
    6. If you’re using Composer, run [`ddev composer install`](../users/usage/commands.md#composer).
    4. Configure any database settings; host='db', user='db', password='db', database='db'
    5. If needed, import a database with [`ddev import-db --src=/path/to/db.sql.gz`](../users/usage/commands.md#import-db).
    6. Visit the project in a browser, and then build things.

=== "WordPress"

    ## WordPress

    There are several easy ways to use DDEV with WordPress:

    === "WP-CLI"

        ### WP-CLI

        DDEV has built-in support for [WP-CLI](https://wp-cli.org/), the command-line interface for WordPress.
        
        ```bash
        mkdir my-wp-site
        cd my-wp-site/
        
        # Create a new DDEV project inside the newly-created folder
        # (Primary URL automatically set to `https://<folder>.ddev.site`)
        ddev config --project-type=wordpress
        ddev start
        
        # Download WordPress
        ddev wp core download
        
        # Launch in browser to finish installation
        ddev launch
        
        # OR use the following installation command
        # (we need to use single quotes to get the primary site URL from `.ddev/config.yaml` as variable)
        ddev wp core install --url='$DDEV_PRIMARY_URL' --title='New-WordPress' --admin_user=admin --admin_email=admin@example.com --prompt=admin_password
        
        # Launch WordPress admin dashboard in your browser
        ddev launch wp-admin/
        ```

    === "Bedrock"
        
        ### Bedrock

        [Bedrock](https://roots.io/bedrock/) is a modern, Composer-based installation in WordPress:

        ```bash
        mkdir my-wp-bedrock-site
        cd my-wp-bedrock-site
        ddev config --project-type=wordpress --docroot=web --create-docroot
        ddev start
        ddev composer create roots/bedrock
        ```

        Update the `.env` file in the project root for Bedrock’s WordPress configuration convention:

        ```
        DB_NAME=db
        DB_USER=db
        DB_PASSWORD=db
        DB_HOST=db
        WP_HOME=${DDEV_PRIMARY_URL}
        WP_SITEURL=${WP_HOME}/wp
        WP_ENV=development
        ```
    
        You can then run [`ddev start`](../users/usage/commands.md#start) and [`ddev launch`](../users/usage/commands.md#launch).
    
        For more details, see [Bedrock installation](https://docs.roots.io/bedrock/master/installation/).

    === "Git Clone"
    
        ### Git Clone

        To get started using DDEV with an existing WordPress project, clone the project’s repository. Note that the git URL shown here is just an example.
        
        ```bash
        git clone https://github.com/example/example-site.git
        cd example-site
        ddev config
        ```
        
        You’ll see a message like:
        
        ```php
        An existing user-managed wp-config.php file has been detected!
        Project DDEV settings have been written to:
        
        /Users/rfay/workspace/bedrock/web/wp-config-ddev.php
        
        Please comment out any database connection settings in your wp-config.php and
        add the following snippet to your wp-config.php, near the bottom of the file
        and before the include of wp-settings.php:
        
        // Include for DDEV-managed settings in wp-config-ddev.php.
        $ddev_settings = dirname(__FILE__) . '/wp-config-ddev.php';
        if (is_readable($ddev_settings) && !defined('DB_USER')) {
          require_once($ddev_settings);
        }
        
        If you don't care about those settings, or config is managed in a .env
        file, etc, then you can eliminate this message by putting a line that says
        // wp-config-ddev.php not needed
        in your wp-config.php
        ```
        
        So just add the suggested include into your `wp-config.php`, or take the workaround shown.
        
        Now start your project with [`ddev start`](../users/usage/commands.md#start).
        
        Quickstart instructions regarding database imports can be found under [Importing a database](#importing-a-database).

=== "Drupal"

    ## Drupal

    === "Drupal 10"

        ### Drupal 10 via Composer
    
        [Drupal 10](https://www.drupal.org/about/10) is fully supported by DDEV.
        
        ```bash
        mkdir my-drupal10-site
        cd my-drupal10-site
        ddev config --project-type=drupal10 --docroot=web --create-docroot
        ddev start
        ddev composer create drupal/recommended-project
        ddev composer require drush/drush
        ddev drush site:install --account-name=admin --account-pass=admin -y
        ddev drush uli
        ddev launch
        ```


    === "Drupal 9"

        ### Drupal 9 via Composer

        ```bash
        mkdir my-drupal9-site
        cd my-drupal9-site
        ddev config --project-type=drupal9 --docroot=web --create-docroot
        ddev start
        ddev composer create "drupal/recommended-project:^9"
        ddev composer require drush/drush
        ddev drush site:install --account-name=admin --account-pass=admin -y
        ddev drush uli
        ddev launch
        ```

    === "Drupal 6/7"

        ### Drupal 6/7
            
        ```bash
        git clone https://github.com/example/my-drupal-site
        cd my-drupal-site
        ddev config # Follow the prompts to select type and docroot
        ddev start
        ddev launch /install.php
        ```
        
        Drupal 7 doesn’t know how to redirect from the front page to `/install.php` if the database is not set up but the settings files *are* set up, so launching with `/install.php` gets you started with an installation. You can also `drush site-install`, then `ddev exec drush site-install --yes`.
        
        See [Importing a Database](#importing-a-database).

    === "Git Clone"

        ### Git Clone

        ```bash
        git clone https://github.com/example/my-drupal-site
        cd example-site
        ddev config # Follow the prompts to set Drupal version and docroot
        ddev composer install # If a composer build
        ddev launch
        ```

=== "TYPO3"

    ## TYPO3

    === "Composer"
    
        ### Composer
        
        ```bash
        mkdir my-typo3-site
        cd my-typo3-site
        ddev config --project-type=typo3 --docroot=public --create-docroot --php-version 8.1
        ddev start
        ddev composer create "typo3/cms-base-distribution"
        ddev exec touch public/FIRST_INSTALL
        ddev launch
        ```

    === "Git Clone"
        
        ### Git Clone
    
        ```bash
        git clone https://github.com/example/example-site
        cd example-site
        ddev config --project-type=typo3 --docroot=public --create-docroot --php-version 8.1
        ddev composer install
        ddev restart
        ddev exec touch public/FIRST_INSTALL
        ddev launch
        ```

=== "OpenMage/Magento 1"

    ## OpenMage/Magento 1

    1. Download OpenMage from [release page](https://github.com/OpenMage/magento-lts/releases).
    2. Make a directory for it, for example `mkdir ~/workspace/OpenMage` and change to the new directory `cd ~/workspace/OpenMage`.
    3. Run [`ddev config`](../users/usage/commands.md#config) and accept the defaults.
    4. Install sample data. (See below.)
    5. Run [`ddev start`](../users/usage/commands.md#start).
    6. Follow the URL to the base site.

    You may want the [Magento 1 Sample Data](https://github.com/Vinai/compressed-magento-sample-data) for experimentation:

    * Download Magento [1.9.2.4 Sample Data](https://github.com/Vinai/compressed-magento-sample-data/raw/master/compressed-magento-sample-data-1.9.2.4.tgz).
    * Extract the download:  
        `tar -zxf ~/Downloads/compressed-magento-sample-data-1.9.2.4.tgz --strip-components=1`
    * Import the example database `magento_sample_data_for_1.9.2.4.sql` with `ddev import-db --src=magento_sample_data_for_1.9.2.4.sql` to database **before** running OpenMage install.

    OpenMage is a huge codebase, and we recommend [using Mutagen for performance](install/performance.md#using-mutagen) on macOS and traditional Windows.

=== "Magento 2"

    ## Magento 2

    Normal details of a Composer build for Magento 2 are on the [Magento 2 site](https://devdocs.magento.com/guides/v2.4/install-gde/composer.html. You must have a public and private key to install from Magento’s repository. When prompted for “username” and “password” in `composer create`, it’s asking for your public and private keys.
    
    ```bash
    mkdir ddev-magento2 && cd ddev-magento2
    ddev config --project-type=magento2 --php-version=8.1 --docroot=pub --create-docroot --disable-settings-management
    ddev get drud/ddev-elasticsearch
    ddev start
    ddev composer create --repository=https://repo.magento.com/ magento/project-community-edition -y
    rm -f app/etc/env.php
    # Change the base-url below to your project's URL
    ddev magento setup:install --base-url='https://ddev-magento2.ddev.site/' --cleanup-database --db-host=db --db-name=db --db-user=db --db-password=db --elasticsearch-host=elasticsearch --admin-firstname=Magento --admin-lastname=User --admin-email=user@example.com --admin-user=admin --admin-password=admin123 --language=en_US
    ddev magento deploy:mode:set developer
    ddev magento module:disable Magento_TwoFactorAuth
    ddev config --disable-settings-management=false
    ```

    Change the admin name and related information is needed.

    You may want to add the [Magento 2 Sample Data](https://devdocs.magento.com/guides/v2.4/install-gde/install/sample-data-after-composer.html) with `ddev magento sampledata:deploy && ddev magento setup:upgrade`.
    
    Magento 2 is a huge codebase, and we recommend [using Mutagen for performance](install/performance.md#using-mutagen) on macOS and traditional Windows.

=== "Moodle"

    ## Moodle
    
    ```bash
    ddev config --composer-root=public --create-docroot --docroot=public --webserver-type=apache-fpm
    ddev start
    ddev composer create moodle/moodle -y
    ddev exec 'php public/admin/cli/install.php --non-interactive --agree-license --wwwroot=$DDEV_PRIMARY_URL --dbtype=mariadb --dbhost=db --dbname=db --dbuser=db --dbpass=db --fullname="DDEV Moodle Demo" --shortname=Demo --adminpass=password'
    ddev launch /login
    ```
    
    In the web browser, log into your account using `admin` and `password`.

    Visit the [Moodle Admin Quick Guide](https://docs.moodle.org/400/en/Admin_quick_guide) for more information.

    !!!tip
        Moodle relies on a periodic cron job—don’t forget to set that up! See [drud/ddev-cron](https://github.com/drud/ddev-cron).

=== "Laravel"

    ## Laravel

    Use a new or existing Composer project, or clone a Git repository.

    The Laravel project type can be used for [Lumen](https://lumen.laravel.com/) just as it can for Laravel. DDEV automatically updates or creates the .env file with the database information.

    === "Composer"
        ```bash
        mkdir my-laravel-app
        cd my-laravel-app
        ddev config --project-type=laravel --docroot=public --create-docroot
        ddev composer create --prefer-dist --no-install --no-scripts laravel/laravel
        ddev composer install
        ddev exec "php artisan key:generate"
        ddev launch
        ```
    === "Git Clone"
        ```bash
        git clone <your-laravel-repo>
        cd <your-laravel-project>
        ddev config --project-type=laravel --docroot=public --create-docroot
        ddev start
        ddev composer install
        ddev exec "php artisan key:generate"
        ddev launch
        ```

=== "Craft CMS"

    ## Craft CMS

    Start a new [Craft CMS](https://craftcms.com) project or retrofit an existing one.

    !!!tip "Compatibility"
        The `craft` project type was added to DDEV in version [1.21.2](https://github.com/drud/ddev/releases/tag/v1.21.2). Check your current version with the `ddev version` command, and [upgrade](../users/usage/faq.md#how-can-i-updateupgrade-ddev) if necessary!

    Environment variables will be automatically added to your `.env` file to simplify the first boot of a project. For _new_ installations, this means the default URL and database connection settings displayed during installation can be used without modification. If _existing_ projects expect environment variables to be named in a particular way, you are welcome to rename them.

    === "New projects"

        New Craft CMS projects can be created from the official [starter project](https://github.com/craftcms/craft) using DDEV’s [`composer create` command](../users/usage/commands.md#composer):

        ```bash
        # Create a project directory and move into it:
        mkdir my-craft-project
        cd my-craft-project

        # Set up the DDEV environment:
        ddev config --project-type=craftcms --docroot=web --create-docroot

        # Boot the project and install the starter project:
        ddev start
        ddev composer create -y --no-scripts craftcms/craft

        # Run the Craft installer:
        ddev craft install
        ddev launch
        ```

        Third-party starter projects can by used the same way—just substitute the package name when running `ddev composer create`.

    === "Existing projects"

        You can start using DDEV with an existing project, too—just make sure you have a database backup handy!

        ```bash
        # Clone an existing repository (or navigate to a local project directory):
        git clone https://github.com/example/example-site my-craft-project
        cd my-craft-project

        # Set up the DDEV environment:
        ddev config --project-type=craftcms

        # Boot the project and install Composer packages:
        ddev start
        ddev composer install

        # Import a database backup and open the site in your browser:
        ddev import-db --src=/path/to/db.sql.gz
        ddev launch
        ```

    !!!tip "Upgrading or using a generic project type?"
        If you previously set up DDEV in a Craft project using the generic `php` project type, update the `type:` setting in `.ddev/config.yaml` to `craftcms`, then run [`ddev restart`](../users/usage/commands.md#restart) apply the changes.

    ### Running Craft in a Sub-directory

    In order for `ddev craft` to work when Craft is installed in a sub-directory, you will need to change the location of the `craft` executable by providing the `CRAFT_CMD_ROOT` environment variable to the web container. For example, if the installation lives in `my-craft-project/app`, you would run `ddev config --web-environment-add=CRAFT_CMD_ROOT=./app`. `CRAFT_CMD_ROOT` defaults to `./`, the project root directory. Run `ddev restart` to apply the change.

    More information about customizing the environment and persisting configuration can be found in [Providing Custom Environment Variables to a Container](https://ddev.readthedocs.io/en/latest/users/extend/customization-extendibility/#providing-custom-environment-variables-to-a-container).

    !!!tip "Installing Craft"
        Read more about installing Craft in the [official documentation](https://craftcms.com/docs).

=== "Shopware 6"

    ## Shopware 6

    You can set up a Shopware 6 environment many ways, we recommend the following technique:

    ```bash
    git clone --branch=6.4 https://github.com/shopware/production my-shopware6
    cd my-shopware6
    ddev config --project-type=shopware6 --docroot=public
    ddev start
    ddev composer install --no-scripts
    # During system:setup you may have to enter the Database user (db), Database password (db)
    # Database host (db) and Database name (db). 
    ddev exec bin/console system:setup --database-url=mysql://db:db@db:3306/db --app-url='${DDEV_PRIMARY_URL}'
    ddev exec bin/console system:install --create-database --basic-setup
    ddev launch /admin
    ```

    Log into the admin site (`/admin`) using the web browser. The default credentials are username `admin` and password `shopware`. You can use the web UI to install sample data or accomplish many other tasks.

    For more advanced tasks like adding elasticsearch, building and watching storefront and administration, see [susi.dev](https://susi.dev/ddev-shopware-6).

=== "Backdrop"

    ## Backdrop

    To get started with Backdrop, clone the project repository and navigate to the project directory.
    
    ```bash
    git clone https://github.com/example/example-site
    cd example-site
    ddev config
    ddev start
    ddev launch
    ```

## Configuration Files

The [`ddev config`](../users/usage/commands.md#config) command attempts to create a CMS-specific settings file pre-populated with DDEV credentials.

For **Drupal** and **Backdrop**, DDEV settings are written to a DDEV-managed file, `settings.ddev.php`. The `ddev config` command will ensure these settings are included in your `settings.php` through the following steps:

* Write DDEV settings to `settings.ddev.php`.
* If no `settings.php` file exists, create one that includes `settings.ddev.php`.
* If a `settings.php` file already exists, ensure that it includes `settings.ddev.php`, modifying `settings.php` to write the include if necessary..

For **Magento 1**, DDEV settings go into `app/etc/local.xml`

In **Magento 2**, DDEV settings go into `app/etc/env.php`

For **TYPO3**, DDEV settings are written to `AdditionalConfiguration.php`. If `AdditionalConfiguration.php` exists and is not managed by DDEV, it will not be modified.

For **WordPress**, DDEV settings are written to a DDEV-managed file, `wp-config-ddev.php`. The `ddev config` command will attempt to write settings through the following steps:

* Write DDEV settings to `wp-config-ddev.php`.
* If no `wp-config.php` exists, create one that include `wp-config-ddev.php`.
* If a DDEV-managed `wp-config.php` exists, create one that includes `wp-config.php`.
* If a user-managed `wp-config.php` exists, instruct the user on how to modify it to include DDEV settings.

You’ll know DDEV is managing a settings file when you see the comment below. Remove the comment and DDEV will not attempt to overwrite it! If you’re letting DDEV create its settings file, we recommended leaving this comment so DDEV can continue to manage it, and make any needed changes in another settings file.

```

/**
 #ddev-generated: Automatically generated Drupal settings.php file.
 ddev manages this file and may delete or overwrite the file unless this comment is removed.
 */

```

If you’re providing the `settings.php` or `wp-config.php` and DDEV is creating `settings.ddev.php` (or `wp-config-local.php`, `AdditionalConfig.php`, or similar), the main settings file must explicitly include the appropriate DDEV-generated settings file. Any changes you need should be included somewhere that loads after DDEV’s settings file, for example in Drupal’s `settings.php` *after* `settings.ddev.php` is included. (See [Adding Configuration](#adding-configuration) below).

!!!note "Completely Disabling Settings Management"

    If you do *not* want DDEV to create or manage settings files, set `disable_settings_management: true` in `.ddev/config.yaml` or run `ddev config --disable-settings-management`. Once you’ve done that, it’s solely up to you to manually edit those settings.

### Adding Configuration

**Drupal and Backdrop**: In `settings.php`, enable loading `settings.local.php` after `settings.ddev.php` is included—creating a new one if it doesn’t already exist—and make changes there. Wrap with `if (getenv('IS_DDEV_PROJECT') == 'true')` as needed.

**WordPress**: Load a `wp-config-local.php` after `wp-config-ddev.php`, and make changes there. Wrap with `if (getenv('IS_DDEV_PROJECT') == 'true')` as needed.

## Listing Project Information

Run [`ddev list`](../users/usage/commands.md#list) or `ddev list --active-only` current projects.

```

➜  ddev list
NAME          TYPE     LOCATION                   URL(s)                                STATUS
d8git         drupal8  ~/workspace/d8git          <https://d8git.ddev.local>              running
                                                  <http://d8git.ddev.local>
hobobiker     drupal6  ~/workspace/hobobiker.com                                        stopped

```

```

➜  ddev list --active-only
NAME     TYPE     LOCATION             URL(s)                      STATUS
drupal8  drupal8  ~/workspace/drupal8  <http://drupal8.ddev.site>   running
                                       <https://drupal8.ddev.site>

```

You can also see more detailed information about a project by running [`ddev describe`](../users/usage/commands.md#describe) from its working directory. You can also run `ddev describe [project-name]` from any location to see the detailed information for a running project.

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
MailHog (https):    https://d9composer.ddev.site:8026
MailHog:            http://d9composer.ddev.site:8025
phpMyAdmin (https): https://d9composer.ddev.site:8037
phpMyAdmin:         http://d9composer.ddev.site:8036

DDEV ROUTER STATUS: healthy
ssh-auth status: healthy
```

## Removing Projects

There are two ways to remove a project from DDEV’s listing.

The first, the [`ddev delete`](../users/usage/commands.md#delete) command, is destructive. It removes the project from DDEV’s list, deletes its database, and removes the hostname entry from the hosts file:

`ddev delete <projectname>`  
or  
`ddev delete --omit-snapshot <projectname>`

If you simply don’t want the project to show up in [`ddev list`](../users/usage/commands.md#list) anymore, use [`ddev stop`](../users/usage/commands.md#stop)—which is nondestructive—to unlist the project until the next time you run [`ddev start`](../users/usage/commands.md#start) or [`ddev config`](../users/usage/commands.md#config):

```bash
ddev stop --unlist <projectname>
```

## Importing Assets for An Existing Project

An important aspect of local web development is the ability to have a precise local recreation of the project you’re working on, including up-to-date database contents and static assets like uploaded images and files. DDEV provides two commands to help with importing assets to your local environment.

### Importing a Database

The [`ddev import-db`](../users/usage/commands.md#import-db) command imports the database for a project. Running this command will prompt you to specify the location of your database import. By default `ddev import-db` empties the default `db` database, then loads the provided dump file. Most people use it with command flags, like `ddev import-db --src=.tarballs/db.sql.gz`, but it can also prompt for the location of the dump if you only use `ddev import-db`:

```bash
ddev import-db
Provide the path to the database you wish to import.
Import path:
~/Downloads/db.sql
Importing database...
Successfully imported database for drupal8
```

#### Supported File Types

Database imports can be any of the following file types:

* Raw SQL Dump (`.sql`)
* Gzipped SQL Dump (`.sql.gz`)
* Xz’d SQL Dump (`.sql.xz`)
* (Gzipped) Tarball Archive (`.tar`, `.tar.gz`, `.tgz`)
* Zip Archive (`.zip`)
* stdin

If a Tarball Archive or Zip Archive is provided for the import, you’ll be prompted to specify a path within the archive to use for the import asset. The specified path should provide a raw SQL dump (`.sql`). In the following example, the database we want to import is named `data.sql` and resides at the top level of the archive:

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

If you want to use the [`import-db`](../users/usage/commands.md#import-db) command without answering prompts, you can use the `--src` flag to provide the path to the import asset. If you’re importing an archive and wish to specify the path within the archive to extract, you can use the `--extract-path` flag in conjunction with the `--src` flag. Examples:

```bash
ddev import-db --src=/tmp/mydb.sql.gz
gzip -dc /tmp/mydb.sql.gz | ddev import-db
ddev import-db <mydb.sql
```

#### Database Import Notes

* Importing from a dump file via stdin will not show progress because there’s no way the import can know how far along through the import it has progressed.
* Use `ddev import-db --target-db <some_database>` to import to a non-default database (other than the default `db` database). This will create the database if it doesn’t already exist.
* Use `ddev import-db --no-drop` to import without first emptying the database.
* If a database already exists and the import does not specify dropping tables, the contents of the imported dumpfile will be *added* to the database. Most full database dumps do a table drop and create before loading, but if yours does not, you can drop all tables with `ddev stop --remove-data` before importing.
* If imports are stalling or failing, make sure you have plenty of unused space (see [#3360](https://github.com/drud/ddev/issues/3360)). DDEV has no problems importing large (2G+) databases, but importing requires lots of space. DDEV will show a warning on startup if unused space is getting low.
