# CMS Quickstarts

With the generic `php` and `python` project types DDEV is [ready to go](./project.md) with any PHP and Python based content management system (CMS) or framework. In addition there are  preconfigured project types for many popular platforms and CMSes. To learn more about how to manage projects in DDEV visit the [Managing Projects](../users/usage/managing-projects.md) page.

Before proceeding, make sure your installation of DDEV is up-to-date. In a new and empty project folder, using your favorite shell, run the following commands:

## Backdrop

To get started with [Backdrop](https://backdropcms.org), clone the project repository and navigate to the project directory.

```bash
git clone https://github.com/example/example-site
cd example-site
ddev config
ddev start
ddev launch
```

## Craft CMS

Start a new [Craft CMS](https://craftcms.com) project or retrofit an existing one.

!!!tip "Compatibility with Craft CMS 3"
    The `craftcms` project type is best with Craft CMS 4+, which is more opinionated about some settings. If you are using Craft CMS 3, you may want to use the `php` project type and [manage settings yourself](https://github.com/ddev/ddev/issues/4650).

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
    ddev import-db --file=/path/to/db.sql.gz
    ddev launch
    ```

    Craft CMS projects use PHP 8.1 and MySQL 8.0, by default. You can override these settings during setup with [`config` command flags](./usage/commands.md#config) or after setup via the [configuration files](./configuration/config.md).

    !!!tip "Upgrading or using a generic project type?"
        If you previously set up DDEV in a Craft project using the generic `php` project type, update the `type:` setting in `.ddev/config.yaml` to `craftcms`, then run [`ddev restart`](../users/usage/commands.md#restart) apply the changes.

### Running Craft in a Sub-directory

In order for `ddev craft` to work when Craft is installed in a sub-directory, you will need to change the location of the `craft` executable by providing the `CRAFT_CMD_ROOT` environment variable to the web container. For example, if the installation lives in `my-craft-project/app`, you would run `ddev config --web-environment-add=CRAFT_CMD_ROOT=./app`. `CRAFT_CMD_ROOT` defaults to `./`, the project root directory. Run `ddev restart` to apply the change.

More information about customizing the environment and persisting configuration can be found in [Providing Custom Environment Variables to a Container](https://ddev.readthedocs.io/en/latest/users/extend/customization-extendibility/#providing-custom-environment-variables-to-a-container).

!!!tip "Installing Craft"
    Read more about installing Craft in the [official documentation](https://craftcms.com/docs).

## Django 4 (Experimental)

```bash
git clone https://github.com/example/my-django-site
cd my-django-site
ddev config # Follow the prompts
# If your settings file is not `settings.py` you must add a DJANGO_SETTINGS_MODULE
ddev config --web-environment-add=DJANGO_SETTINGS_MODULE=<myapp.settings.local>
ddev start
# If your app requires setup, do it here:
# ddev python manage.py migrate
ddev launch
```

* DDEV will install all everything in your `requirements.txt` or `pyproject.toml` into a `venv`. This takes a little while on first startup.
* DDEV appends a stanza to your settings file which includes the DDEV settings only if running in DDEV context.
* You can watch the `pip install` in real time on that first slow startup with `ddev logs -f` in another window.
* If your `requirements.txt` includes `psycopg2` it requires build tools, so either set `ddev config --web-extra-packages=build-essential` or change yourrequirement to `psycopg2-binary`.

## Drupal

=== "Drupal 10"

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

    ```bash
    git clone https://github.com/example/my-drupal-site
    cd my-drupal-site
    ddev config # Follow the prompts to set Drupal version and docroot
    ddev composer install # If a composer build
    ddev launch
    ```

## ExpressionEngine

=== "ExpressionEngine ZIP File Download"

    Download the ExpressionEngine code from [expressionengine.com](https://expressionengine.com/), then follow these steps based on the [official installation instructions](https://docs.expressionengine.com/latest/installation/installation.html):

    ```bash
    mkdir my-ee && cd my-ee
    unzip /path/to/ee-zipfile.zip
    ddev config # Accept the defaults
    ddev start
    ddev launch /admin.php # Open installation wizard in browser
    ```

    When the installation wizard prompts for database settings, enter `db` for the _DB Server Address_, _DB Name_, _DB Username_, and _DB Password_.

    Visit your site.

=== "ExpressionEngine Git Checkout"

    Follow these steps based on the [ExpressionEngine Git Repository README.md](https://github.com/ExpressionEngine/ExpressionEngine#how-to-install):

    ```bash
    git clone https://github.com/ExpressionEngine/ExpressionEngine # for example
    cd ExpressionEngine
    ddev config # Accept the defaults
    ddev start
    ddev composer install
    touch system/user/config/config.php
    echo "EE_INSTALL_MODE=TRUE" >.env.php
    ddev start
    ddev launch /admin.php  # Open installation wizard in browser
    ```

    When the installation wizard prompts for database settings, enter `db` for the _DB Server Address_, _DB Name_, _DB Username_, and _DB Password_.

## Laravel

Use a new or existing Composer project, or clone a Git repository.

The Laravel project type can be used for [Lumen](https://lumen.laravel.com/) just as it can for Laravel. DDEV automatically updates or creates the `.env` file with the database information.

=== "Composer"
            ```bash
            mkdir my-laravel-app
            cd my-laravel-app
            ddev config --project-type=laravel --docroot=public --create-docroot --php-version=8.1
            ddev composer create --prefer-dist --no-install --no-scripts laravel/laravel -y
            ddev composer install
            ddev exec "php artisan key:generate"
            ddev launch
            ```
=== "Git Clone"
            ```bash
            git clone <your-laravel-repo>
            cd <your-laravel-project>
            ddev config --project-type=laravel --docroot=public --create-docroot --php-version=8.1
            ddev start
            ddev composer install
            ddev exec "php artisan key:generate"
            ddev launch
            ```

## Magento

=== "Magento 2"

    Normal details of a Composer build for Magento 2 are on the [Magento 2 site](https://devdocs.magento.com/guides/v2.4/install-gde/composer.html). You must have a public and private key to install from Magento’s repository. When prompted for “username” and “password” in `composer create`, it’s asking for your public and private keys.

    ```bash
    mkdir ddev-magento2 && cd ddev-magento2
    ddev config --project-type=magento2 --php-version=8.1 --docroot=pub --create-docroot --disable-settings-management
    ddev get ddev/ddev-elasticsearch
    ddev start
    ddev composer create --repository=https://repo.magento.com/ magento/project-community-edition -y
    rm -f app/etc/env.php

    # Change the base-url below to your project's URL
    ddev magento setup:install --base-url='https://ddev-magento2.ddev.site/' --cleanup-database --db-host=db --db-name=db --db-user=db --db-password=db --elasticsearch-host=elasticsearch --search-engine=elasticsearch7 --elasticsearch-port=9200 --admin-firstname=Magento --admin-lastname=User --admin-email=user@example.com --admin-user=admin --admin-password=admin123 --language=en_US

    ddev magento deploy:mode:set developer
    ddev magento module:disable Magento_TwoFactorAuth Magento_AdminAdobeImsTwoFactorAuth
    ddev config --disable-settings-management=false
    ```

    Change the admin name and related information is needed.

    You may want to add the [Magento 2 Sample Data](https://devdocs.magento.com/guides/v2.4/install-gde/install/sample-data-after-composer.html) with `ddev magento sampledata:deploy && ddev magento setup:upgrade`.

    Magento 2 is a huge codebase, and we recommend [using Mutagen for performance](install/performance.md#using-mutagen) on macOS and traditional Windows.

=== "OpenMage/Magento 1"

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
    * Import the example database `magento_sample_data_for_1.9.2.4.sql` with `ddev import-db --file=magento_sample_data_for_1.9.2.4.sql` to database **before** running OpenMage install.

    OpenMage is a huge codebase, and we recommend [using Mutagen for performance](install/performance.md#using-mutagen) on macOS and traditional Windows.

## Moodle

```bash
ddev config --composer-root=public --create-docroot --docroot=public --webserver-type=apache-fpm
ddev start
ddev composer create moodle/moodle -y
ddev exec 'php public/admin/cli/install.php --non-interactive --agree-license --wwwroot=$DDEV_PRIMARY_URL --dbtype=mariadb --dbhost=db --dbname=db--dbuser=db --dbpass=db --fullname="DDEV Moodle Demo" --shortname=Demo --adminpass=password'
ddev launch /login
```

In the web browser, log into your account using `admin` and `password`.

Visit the [Moodle Admin Quick Guide](https://docs.moodle.org/400/en/Admin_quick_guide) for more information.

!!!tip
    Moodle relies on a periodic cron job—don’t forget to set that up! See [ddev/ddev-cron](https://github.com/ddev/ddev-cron).

## Python/Flask (Experimental)

```bash
git clone https://github.com/example/my-python-site
cd my-python-site
ddev config # Follow the prompts
# Tell gunicorn where your app is (WSGI_APP)
ddev config --web-environment-add=WSGI_APP=<my-app:app>
ddev start
# If you need to do setup before the site can go live, do it:
# ddev exec flask forge
ddev launch
```

* DDEV will install all everything in your `requirements.txt` or `pyproject.toml` into a `venv`. This takes a little while on first startup.
* If your app requires settings, you can add them as environment variables, or otherwise configure your app to use the database, etc. (Database settingsare host: `db`, database: `db`, user: `db`, password `db` no matter whether you're using PostgreSQL, MariaDB, or MySQL.)
* You can watch `pip install` output in real time on that first slow startup with `ddev logs -f` in another window.
* If your `requirements.txt` includes `psycopg2` it requires build tools, so either set `ddev config --web-extra-packages=build-essential` or change yourrequirement to `psycopg2-binary`.

## Shopware

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

## Statamic

Use a new or existing Composer project, or clone a Git repository.

The Laravel project type can be used for [Statamic](https://statamic.com/) just as it can for Laravel. DDEV automatically updates or creates the `.env` file with the database information.

=== "Composer"

    ```bash
    mkdir my-statamic-app
    cd my-statamic-app
    ddev config --project-type=laravel --docroot=public --create-docroot
    ddev composer create --prefer-dist --no-install --no-scripts statamic/statamic
    ddev composer install
    ddev exec "php artisan key:generate"
    ddev launch
    ```
=== "Git Clone"

    ```bash
    git clone <your-statamic-repo>
    cd <your-statamic-project>
    ddev config --project-type=laravel --docroot=public --create-docroot
    ddev start
    ddev composer install
    ddev exec "php artisan key:generate"
    ddev launch
    ```

## TYPO3

=== "Composer"

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

    ```bash
    git clone https://github.com/example/example-site
    cd example-site
    ddev config --project-type=typo3 --docroot=public --create-docroot --php-version 8.1
    ddev composer install
    ddev restart
    ddev exec touch public/FIRST_INSTALL
    ddev launch
    ```

## WordPress

There are several easy ways to use DDEV with WordPress:

=== "WP-CLI"

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

    To get started using DDEV with an existing WordPress project, clone the project’s repository.

    ```bash
    git clone https://github.com/example/my-site.git
    cd my-site
    ddev config
    ```

    You’ll see a message like:

    > An existing user-managed wp-config.php file has been detected!
    > Project DDEV settings have been written to:
    >
    > /Users/rfay/workspace/bedrock/web/wp-config-ddev.php

    Comment out any database connection settings in your `wp-config.php` file and add the following snippet to your `wp-config.php`, near the bottom of the file and before the include of `wp-settings.php`:

    ```php
    // Include for DDEV-managed settings in wp-config-ddev.php.
    $ddev_settings = dirname(__FILE__) . '/wp-config-ddev.php';
    if (is_readable($ddev_settings) && !defined('DB_USER')) {
    require_once($ddev_settings);
    }
    ```

    If you don't care about those settings, or config is managed elsewhere (like in a `.env` file), you can eliminate this message by adding a comment to `wp-config.php`:

    ```php
    // wp-config-ddev.php not needed
    ```

    Now run [`ddev start`](../users/usage/commands.md#start) and continue [importing a database](#importing-a-database) if you need to.
