# Get Started!

Once you have DDEV installed, getting a project going is just these steps:

1. Clone or create the code for your project.
2. `cd` into the project and `ddev config` to configure it and turn it into a DDEV project. In most cases DDEV will autodetect the project type and docroot, but you may have to provide them in others.
3. `ddev start` and if your project needs it, `ddev composer install`
4. `ddev launch` to launch a browser with your project, or visit the URL given by `ddev start`.
5. Import an upstream database with `ddev import-db`.
6. Import user-files from upstream with `ddev import-files`

Here's a quickstart instructions for a number of different environments:

=== "Any (generic)"

    DDEV works happily with most any PHP or static HTML/js project, although it has special additional support for several CMSs. But you don't need special support if you already know how to configure your project.
    
    1. Create a directory (`mkdir my-new-project`) or clone your project (`git clone <your_project>`)
    2. Change to the new directory (`cd my-new-project`)
    3. Run `ddev config` and set the project type and docroot, which are usually auto-detected, but may not be if there's no code in there yet.
    4. Run `ddev start`
    6. If a composer build, `ddev composer install`
    4. Configure any database settings; host='db', user='db', password='db', database='db'
    5. If needed, import a database with `ddev import-db --src=/path/to/db.sql.gz`.
    6. Visit the project and continue on.

=== "WordPress"

    There are several easy ways to use DDEV with WordPress:

    === "wp-cli"

        DDEV has built-in support for [WP-CLI](https://wp-cli.org/), the command-line interface for WordPress.
        
        ```bash
        mkdir my-wp-site
        cd my-wp-site/
        
        # create a new DDEV project inside the newly created folder
        # (the primary URL is automatically set to https://<folder>.ddev.site) 
        
        ddev config --project-type=wordpress
        ddev start
        
        # download latest WordPress (via WP-CLI)
        
        ddev wp core download
        
        # finish the installation in your browser:
        
        ddev launch
        
        # optional: you can use the following installation command 
        # (we need to use single quotes to get the primary site URL from .ddev/config.yaml as variable)
        
        ddev wp core install --url='$DDEV_PRIMARY_URL' --title='New-WordPress' --admin_user=admin --admin_email=admin@example.com --prompt=admin_password
        
        # open WordPress admin dashboard in your browser:
        
        ddev launch wp-admin/
        ```

    === "roots/bedrock"
        
        roots/bedrock is a modern composer-based installation if WordPress:

        ```bash
        mkdir my-wp-bedrock-site
        cd my-wp-bedrock-site
        ddev config --project-type=wordpress --docroot=web --create-docroot
        ddev start
        ddev composer create roots/bedrock
        ```
    
        Now, since [Bedrock](https://roots.io/bedrock/) uses a configuration technique which is unusual for WordPress, edit the .env file which has been created in the project root, and set:
    
        ```
            DB_NAME=db
            DB_USER=db
            DB_PASSWORD=db
            DB_HOST=db
            WP_HOME=${DDEV_PRIMARY_URL}
            WP_SITEURL=${WP_HOME}/wp
            WP_ENV=development
        ```
    
        You can then `ddev start` and `ddev launch`.
    
        For more details see [Bedrock installation](https://roots.io/bedrock/docs/installing-bedrock/).

    === "git clone"
    
        To get started using DDEV with an existing WordPress project, clone the project's repository. Note that the git URL shown here is just an example.
        
        ```bash
        git clone https://github.com/example/example-site.git
        cd example-site
        ddev config
        ```
        
        You'll see a message like:
        
        ```php
        An existing user-managed wp-config.php file has been detected!
        Project ddev settings have been written to:
        
        /Users/rfay/workspace/bedrock/web/wp-config-ddev.php
        
        Please comment out any database connection settings in your wp-config.php and
        add the following snippet to your wp-config.php, near the bottom of the file
        and before the include of wp-settings.php:
        
        // Include for ddev-managed settings in wp-config-ddev.php.
        $ddev_settings = dirname(__FILE__) . '/wp-config-ddev.php';
        if (is_readable($ddev_settings) && !defined('DB_USER')) {
          require_once($ddev_settings);
        }
        
        If you don't care about those settings, or config is managed in a .env
        file, etc, then you can eliminate this message by putting a line that says
        // wp-config-ddev.php not needed
        in your wp-config.php
        ```
        
        So just add the suggested include into your wp-config.php, or take the workaround shown.
        
        Now start your project with `ddev start`
        
        Quickstart instructions regarding database imports can be found under [Database Imports](#database-imports).

=== "Drupal"

    === "Drupal 9 Composer"
        ```bash
        mkdir my-drupal9-site
        cd my-drupal9-site
        ddev config --project-type=drupal9 --docroot=web --create-docroot
        ddev start
        ddev composer create "drupal/recommended-project" --no-install
        ddev composer require drush/drush --no-install
        ddev composer install
        ddev drush site:install -y
        ddev drush uli
        ddev launch
        ```

    === "Drupal 10"
    
        [Drupal 10](https://www.drupal.org/about/10) is not yet released, but lots of people want to test and contribute to it. It's easy to set it up in DDEV:
        
        ```bash
        mkdir my-drupal10-site
        cd my-drupal10-site
        ddev config --project-type=drupal10 --docroot=web --create-docroot
        ddev start
        ddev composer create --no-install drupal/recommended-project:^10@alpha
        ddev composer require drush/drush --no-install
        ddev composer install
        ddev drush site:install -y
        ddev drush uli
        ddev launch
        ```
        
        Note that as Drupal 10 moves from alpha to beta and then release, you'll want to change the tag from `^10@alpha` to `^10@beta` and then `^10`.

    === "Drupal 6/7"
    
        Using DDEV with a Drupal 6 or 7 project is as simple as cloning the project's repository and checking out its directory.
        
        ```bash
        git clone https://github.com/user/my-drupal-site
        cd my-drupal-site
        ddev config # Follow the prompts to select type and docroot
        ddev start
        ddev launch /install.php
        ```
        
        (Drupal 7 doesn't know how to redirect from the front page to the /install.php if the database is not set up but the settings files *are* set up, so launching with /install.php gets you started with an installation. You can also `drush site-install`, `ddev exec drush site-install --yes`)
        
        Quickstart instructions for database imports can be found under [Database Imports](#database-imports).

    === "Git clone"
    
        Note that the git URL shown below is an example only, you'll need to use your own project.
        
        ```bash
        git clone https://github.com/example/example-site
        cd example-site
        ddev config # Follow the prompts to set drupal version and docroot
        ddev composer install  # If a composer build
        ddev launch
        ```

=== "TYPO3"

    === "Composer build"
        
        ```bash
        mkdir my-typo3-site
        cd my-typo3-site
        ddev config --project-type=typo3 --docroot=public --create-docroot
        ddev start
        ddev composer create "typo3/cms-base-distribution" --no-install
        ddev composer install
        ddev exec touch public/FIRST_INSTALL
        ddev launch
        ```

    === "Git clone"
    
        ```bash
        git clone https://github.com/example/example-site
        cd example-site
        ddev config
        ddev composer install
        ddev launch
        ```

=== "OpenMage/Magento 1"

    1. Download OpenMage from [release page](https://github.com/OpenMage/magento-lts/releases).
    2. Make a directory for it, for example `mkdir ~/workspace/OpenMage` and change to the new directory `cd ~/workspace/OpenMage`.
    3. `ddev config` and accept the defaults.
    4. (Install sample data - see below)
    5. Run `ddev start`
    6. Follow the URL to the base site.

    You may want the [Magento 1 Sample Data](https://github.com/Vinai/compressed-magento-sample-data) for experimentation:

    * Download Magento [1.9.1.0 Sample Data](https://raw.githubusercontent.com/Vinai/compressed-magento-sample-data/1.9.1.0/compressed-magento-sample-data-1.9.1.0.tgz).
    * Extract the download, for example `tar -zxf ~/Downloads/compressed-magento-sample-data-1.9.1.0.tgz --strip-components=1`
    * Import the example database "magento_sample_data_for_1.9.1.0.sql" with `ddev import-db --src=magento_sample_data_for_1.9.1.0.sql` to database **before** running OpenMage install.

    Note that OpenMage is a huge codebase and using `mutagen_enabled: true` is recommended for performance on macOS and traditional Windows, see [docs](install/performance.md#using-mutagen).

=== "Magento 2"

    Normal details of a composer build for Magento 2 are on [Magento 2 site](https://devdocs.magento.com/guides/v2.4/install-gde/composer.html) You must have a public and private key to install from Magento's repository; when prompted for "username" and "password" in the composer create it's asking for your public and private keys.
    
    ```bash
    mkdir ddev-magento2 && cd ddev-magento2
    ddev config --project-type=magento2 --php-version=8.1 --docroot=pub --create-docroot --disable-settings-management
    ddev get drud/ddev-elasticsearch
    ddev start
    ddev composer create --no-install --repository=https://repo.magento.com/ magento/project-community-edition -y
    ddev composer install
    rm -f app/etc/env.php
    # Change the base-url below to your project's URL
    ddev magento setup:install --base-url='https://ddev-magento2.ddev.site/' --cleanup-database --db-host=db --db-name=db --db-user=db --db-password=db --elasticsearch-host=elasticsearch --admin-firstname=Magento --admin-lastname=User --admin-email=user@example.com --admin-user=admin --admin-password=admin123 --language=en_US
    ddev magento deploy:mode:set developer
    ddev magento module:disable Magento_TwoFactorAuth
    ddev config --disable-settings-management=false
    ```
    
    Of course, change the admin name and related information is needed.
    
    You may want to add the [Magento 2 Sample Data](https://devdocs.magento.com/guides/v2.4/install-gde/install/sample-data-after-composer.html) with `ddev magento sampledata:deploy && ddev magento setup:upgrade`.
    
    Note that Magento 2 is a huge codebase and using `mutagen_enabled: true` is recommended for performance on macOS and traditional Windows, see [docs](install/performance.md#using-mutagen).

=== "Laravel"

    Get started with Laravel projects on ddev either using a new or existing composer project or by cloning a git repository.
    The Laravel project type can be used for [Lumen](https://lumen.laravel.com/) just as it can for Laravel.
    
    ```bash
    mkdir my-laravel-app
    cd my-laravel-app
    ddev config --project-type=laravel --docroot=public --create-docroot
    ddev start
    ddev composer create --prefer-dist laravel/laravel
    ddev exec "cat .env.example | sed  -E 's/DB_(HOST|DATABASE|USERNAME|PASSWORD)=(.*)/DB_\1=db/g' > .env"
    ddev exec "php artisan key:generate"
    ddev launch
    ```

    
    In the examples above we used a one liner to copy `.env.example` as `env`and set the `DB_HOST`, `DB_DATABASE`, `DB_USERNAME` and `DB_PASSWORD` environment variables to the value of `db`.
    These values are DDEV's default settings for the Database connection.
    
    Instead of setting each connection variable we can add a ddev to the `connections` array in `config/database.php` like this:
    
    ```php
    <?php
    return [
        ...
        'connections' => [
            ...
            'ddev' => [
                'driver' => 'mysql',
                'host' => 'db',
                'port' => 3306,
                'database' => 'db',
                'username' => 'db',
                'password' => 'db',
                'unix_socket' => '',
                'charset' => 'utf8mb4',
                'collation' => 'utf8mb4_unicode_ci',
                'prefix' => '',
                'strict' => true,
                'engine' => null,
            ],
        ],
      ...
    ];
    ```
    
    This way we only need to change the value of `DB_CONNECTION` to `ddev` in the `.env` to work with the `db` service.
    This is very handy if you have a local database installed and you want to switch between the connections faster by changing only one variable in `.env`

=== "Shopware 6"

    You can set up a Shopware 6 environment many ways, but this shows you one recommended technique:
    
    ```bash
    git clone --branch=6.4 https://github.com/shopware/production my-shopware6
    cd my-shopware6
    ddev config --project-type=shopware6 --docroot=public
    ddev start
    ddev composer install
    ddev exec bin/console system:setup --no-interaction --database-url=mysql://db:db@db:3306/db --app-url='${DDEV_PRIMARY_URL}'
    ddev exec bin/console system:install --create-database --basic-setup
    ddev launch /admin
    ```
    
    Now log into the admin site (/admin) using the web browser. The default credentials are username=admin, password=shopware. You can use the web UI to install sample data or accomplish many other tasks.
    
    For more advanced tasks like adding elasticsearch, building and watching storefront and administration, see [susi.dev](https://susi.dev/ddev-shopware-6).


=== "Backdrop"

    To get started with Backdrop, clone the project repository and navigate to the project directory.
    
    ```bash
    git clone https://github.com/example/example-site
    cd example-site
    ddev config
    ddev start
    ddev launch
    ```

## Database Imports

Import a database with just one command; There is support for several file formats, including: **.sql, sql.gz, mysql, mysql.gz, tar, tar.gz, and zip**.

Here's an example of a database import using ddev:

```bash
ddev import-db --src=dumpfile.sql.gz
```

It is also possible to use phpMyAdmin for database imports, but that approach is much slower. Also, the web and db containers container the `mysql` or `psql` client, which can be used for imports, and the `ddev mysql` and `ddev psql` command can be used in the same way you might use `mysql` or `psql` on a server.

!!! note "Backdrop configuration"
        In addition to importing a Backdrop database, you will need to extract a copy of your Backdrop project's configuration into the local `active` directory. The location for this directory can vary depending on the contents of your Backdrop `settings.php` file, but the default location is `[docroot]/files/config_[random letters and numbers]/active`. Please refer to the [Backdrop documentation](https://docs.backdropcms.org/) for more information on moving your Backdrop site into the DDEV environment.


### Configuration files

**Note:** If you're providing the settings.php or wp-config.php and DDEV is creating the settings.ddev.php (or wp-config-local.php, AdditionalConfig.php, or similar), the main settings file must explicitly include the appropriate DDEV-generated settings file.  Any changes you need should be included somewhere that loads after DDEV's settings file, for example in Drupal's settings.php *after* settings.ddev.php is included. (see "Adding Configuration" below).

**Note:** If you do *not* want DDEV-Local to create or manage settings files, set `disable_settings_management: true` in your .ddev/config.yaml or `ddev config --disable-settings-management` and you will be the only one that edits or updates settings files.

The `ddev config` command attempts to create a CMS-specific settings file with DDEV credentials pre-populated.

For **Drupal** and **Backdrop**, DDEV settings are written to a DDEV-managed file, settings.ddev.php. The `ddev config` command will ensure that these settings are included in your settings.php through the following steps:

* Write DDEV settings to settings.ddev.php
* If no settings.php file exists, create one that includes settings.ddev.php
* If a settings.php file already exists, ensure that it includes settings.ddev.php, modifying settings.php to write the include if necessary.

For **Magento 1**, DDEV settings go into `app/etc/local.xml`

In **Magento 2**, DDEV settings go into `app/etc/env.php`

For **TYPO3**, DDEV settings are written to AdditionalConfiguration.php.  If AdditionalConfiguration.php exists and is not managed by DDEV, it will not be modified.

For **WordPress**, DDEV settings are written to a DDEV-managed file, wp-config-ddev.php. The `ddev config` command will attempt to write settings through the following steps:

* Write DDEV settings to wp-config-ddev.php
* If no wp-config.php exists, create one that include wp-config-ddev.php
* If a DDEV-managed wp-config.php exists, create one that includes wp-config.php
* If a user-managed wp-config.php exists, instruct the user on how to modify it to include DDEV settings

How do you know if DDEV manages a settings file? You will see the following comment. Remove the comment and DDEV will not attempt to overwrite it!  If you are letting DDEV create its settings file, it is recommended that you leave this comment so DDEV can continue to manage it, and make any needed changes in another settings file.

```

/**
 #ddev-generated: Automatically generated Drupal settings.php file.
 ddev manages this file and may delete or overwrite the file unless this comment is removed.
 */

```

#### Adding configuration

**Drupal and Backdrop**:  In settings.php, enable loading settings.local.php after settings.ddev.php is included (create a new one if it doesn't already exist), and make changes there (wrapping with `if (getenv('IS_DDEV_PROJECT') == 'true')` as needed).

**WordPress**:  Load a wp-config-local.php after wp-config-ddev.php, and make changes there (wrapping with `if (getenv('IS_DDEV_PROJECT') == 'true')` as needed).

## Listing project information

`ddev list` or `ddev list --active-only` current projects.

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

You can also see more detailed information about a project by running `ddev describe` from its working directory. You can also run `ddev describe [project-name]` from any location to see the detailed information for a running project.

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

## Removing projects from DDEV-Local

To remove a project from DDEV-Local's listing you can use the destructive option (deletes database, removes item from ddev's list, removes hostname entry in hosts file):

`ddev delete <projectname>`
or
`ddev delete --omit-snapshot <projectname>`

Or if you just don't want it to show up in `ddev list` any more, use `ddev stop --unlist <projectname>` to unlist it until the next time you `ddev start` or `ddev config` the project.

## Importing assets for an existing project

An important aspect of local web development is the ability to have a precise recreation of the project you are working on locally, including up-to-date database contents and static assets such as uploaded images and files. ddev provides functionality to help with importing assets to your local environment with two commands.

### Importing a database

The `ddev import-db` command is provided for importing the database for a project. Running this command will provide a prompt for you to specify the location of your database import. By default `ddev import-db` empties the default "db" database and then loads the provided dumpfile. Most people use it with command flags, like `ddev import-db --src=.tarballs/db.sql.gz` but it can also prompt for the location of the dumpfile if you just use `ddev import-db`:

```bash
ddev import-db
Provide the path to the database you wish to import.
Import path:
~/Downloads/db.sql
Importing database...
Successfully imported database for drupal8
```

#### Supported file types

Database import supports the following file types:

* Raw SQL Dump (.sql)
* Gzipped SQL Dump (.sql.gz)
* (Gzipped) Tarball Archive (.tar, .tar.gz, .tgz)
* Zip Archive (.zip)
* stdin

If a Tarball Archive or Zip Archive is provided for the import, you will be provided an additional prompt, allowing you to specify a path within the archive to use for the import asset. The specified path should provide a Raw SQL Dump (.sql). In the following example, the database we want to import is named data.sql and resides at the top-level of the archive:

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

#### Non-interactive usage

If you want to use import-db without answering prompts, you can use the `--src` flag to provide the path to the import asset. If you are importing an archive, and wish to specify the path within the archive to extract, you can use the `--extract-path` flag in conjunction with the `--src` flag. Examples:

```bash
ddev import-db --src=/tmp/mydb.sql.gz
gzip -dc /tmp/mydb.sql.gz | ddev import-db
ddev import-db <mydb.sql
```

#### Database import notes

* Importing from a dumpfile via stdin will not show progress because there's no way the import can know how far along through the import it has progressed.
* Use `ddev import-db --target-db <some_database>` to import to a non-default database (other than the default "db" database). This will create the database if it doesn't exist already.
* Use `ddev import-db --no-drop` to import without first emptying the database.
* If a database already exists and the import does not specify dropping tables, the contents of the imported dumpfile will be *added* to the database. Most full database dumps do a table drop and create before loading, but if yours does not, you can drop all tables with `ddev stop --remove-data` before importing.
