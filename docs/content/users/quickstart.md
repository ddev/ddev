# CMS Quickstarts

DDEV is [ready to go](./project.md) with generic project types for PHP and Python frameworks, and more specific project types for working with popular platforms and CMSes. To learn more about how to manage projects in DDEV visit [Managing Projects](../users/usage/managing-projects.md).

Before proceeding, make sure your installation of DDEV is up to date. In a new and empty project folder, using your favorite shell, run the following commands:

## Backdrop

To get started with [Backdrop](https://backdropcms.org), clone the project repository and navigate to the project directory.

=== "New projects"

    ```bash
    mkdir my-backdrop-site && cd my-backdrop-site
    curl -LJO https://github.com/backdrop/backdrop/releases/latest/download/backdrop.zip
    unzip ./backdrop.zip && rm backdrop.zip && mv -f ./backdrop/{.,}* . && rm -r backdrop
    ddev config --project-type=backdrop
    ddev start
    ddev launch
    ```

=== "Existing projects"

    You can start using DDEV with an existing project, too—but make sure you have a database backup handy!


    ```bash
    # Clone an existing repository (or navigate to a local project directory):
    git clone https://github.com/example/example-site my-backdrop-site
    cd my-backdrop-site

    # Set up the DDEV environment:
    ddev config --project-type=backdrop

    # Boot the project and install Composer packages (if required):
    ddev start
    ddev composer install

    # Import a database backup and open the site in your browser:
    ddev import-db --file=/path/to/db.sql.gz
    ddev launch
    ```

## CakePHP

You can start a new [CakePHP](https://cakephp.org) project or configure an existing one.

The CakePHP project type can be used with any CakePHP project >= 3.x, but it has been fully tested with CakePHP 5.x. DDEV automatically creates the `.env` file with the database information, email transport configuration and a random salt. If `.env` file already exists, `.env.ddev` will be created, so you can take any variable and put it into your `.env` file.

Please note that you will need to change the PHP version to 7.4 to be able to work with CakePHP 3.x.

=== "Composer"

    ```bash
    mkdir my-cakephp-site && cd my-cakephp-site
    ddev config --project-type=cakephp --docroot=webroot
    ddev composer create --prefer-dist --no-interaction cakephp/app:~5.0
    ddev launch
    ```

=== "Git Clone"

    ```bash
    git clone <my-cakephp-repo> my-cakephp-site
    cd my-cakephp-site
    ddev config --project-type=cakephp --docroot=webroot
    ddev start
    ddev composer install
    ddev cake
    ddev launch
    ```

## Craft CMS

Start a new [Craft CMS](https://craftcms.com) project or retrofit an existing one.

!!!tip "Compatibility with Craft CMS 3"
    The `craftcms` project type is best with Craft CMS 4+, which is more opinionated about some settings. If you are using Craft CMS 3 or earlier, you may want to use the `php` project type and [manage settings yourself](https://github.com/ddev/ddev/issues/4650).

Environment variables will be automatically added to your `.env` file to simplify the first boot of a project. For _new_ installations, this means the default URL and database connection settings displayed during installation can be used without modification. If _existing_ projects expect environment variables to be named in a particular way, you are welcome to rename them.

=== "New projects"

    ```bash
    # Create a project directory and move into it:
    mkdir my-craft-site && cd my-craft-site

    # Set up the DDEV environment:
    ddev config --project-type=craftcms --docroot=web

    # Boot the project and install the starter project:
    ddev start
    ddev composer create craftcms/craft
    ddev launch
    ```

    Third-party starter projects can by used the same way—substitute the package name when running `ddev composer create`.

=== "Existing projects"

    You can start using DDEV with an existing project, too—but make sure you have a database backup handy!

    ```bash
    # Clone an existing repository (or navigate to a local project directory):
    git clone https://github.com/example/example-site my-craft-site
    cd my-craft-site

    # Set up the DDEV environment:
    ddev config --project-type=craftcms

    # Boot the project and install Composer packages:
    ddev start
    ddev composer install

    # Import a database backup and open the site in your browser:
    ddev import-db --file=/path/to/db.sql.gz
    ddev launch
    ```

    Craft CMS projects use MySQL 8.0, by default. You can override this setting (and the PHP version) during setup with [`config` command flags](./usage/commands.md#config) or after setup via the [configuration files](./configuration/config.md).

    !!!tip "Upgrading or using a generic project type?"
        If you previously set up DDEV in a Craft project using the generic `php` project type, update the `type:` setting in `.ddev/config.yaml` to `craftcms`, then run [`ddev restart`](../users/usage/commands.md#restart) apply the changes.

### Running Craft in a Subdirectory

In order for `ddev craft` to work when Craft is installed in a subdirectory, you will need to change the location of the `craft` executable by providing the `CRAFT_CMD_ROOT` environment variable to the web container. For example, if the installation lives in `my-craft-site/app`, you would run `ddev config --web-environment-add=CRAFT_CMD_ROOT=./app`. `CRAFT_CMD_ROOT` defaults to `./`, the project root directory. Run `ddev restart` to apply the change.

Read more about customizing the environment and persisting configuration in [Providing Custom Environment Variables to a Container](./extend/customization-extendibility.md#environment-variables-for-containers-and-services).

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

* DDEV will install everything in your `requirements.txt` or `pyproject.toml` into a `venv`. This takes a little while on first startup.
* DDEV appends a stanza to your settings file which includes the DDEV settings only if running in DDEV context.
* You can watch the `pip install` in real time on that first slow startup with `ddev logs -f` in another window.
* If your `requirements.txt` or `pyproject.toml` includes `psycopg2` or `psycopg` it requires build tools, so either set `ddev config --webimage-extra-packages=build-essential` or change your requirement to `psycopg2-binary`.

## Drupal

For all versions of Drupal 8+ the Composer techniques work. The settings configuration is done differently for each Drupal version, but the project type is "drupal".

=== "Drupal 11"

    ```bash
    mkdir my-drupal-site && cd my-drupal-site
    ddev config --project-type=drupal --php-version=8.3 --docroot=web
    ddev start
    ddev composer create drupal/recommended-project:^11
    ddev composer require drush/drush
    ddev config --update
    ddev restart
    ddev drush site:install --account-name=admin --account-pass=admin -y
    ddev launch
    # or automatically log in with
    ddev launch $(ddev drush uli)
    ```

=== "Drupal 10"

    ```bash
    mkdir my-drupal-site && cd my-drupal-site
    ddev config --project-type=drupal --php-version=8.3 --docroot=web
    ddev start
    ddev composer create drupal/recommended-project:^10
    ddev config --update
    ddev composer require drush/drush
    ddev drush site:install --account-name=admin --account-pass=admin -y
    ddev launch
    # or automatically log in with
    ddev launch $(ddev drush uli)
    ```

=== "Drupal 9 (EOL)"

    ```bash
    mkdir my-drupal-site && cd my-drupal-site
    ddev config --project-type=drupal --php-version=8.1 --docroot=web
    ddev start
    ddev composer create drupal/recommended-project:^9
    ddev config --update
    ddev composer require drush/drush
    ddev drush site:install --account-name=admin --account-pass=admin -y
    ddev launch
    # or automatically log in with
    ddev launch $(ddev drush uli)
    ```

=== "Drupal 6/7"

    ```bash
    git clone https://github.com/example/my-drupal-site
    cd my-drupal-site
    ddev config # Follow the prompts to select type and docroot
    ddev start
    ddev launch /install.php
    ```

    Drupal 7 doesn’t know how to redirect from the front page to `/install.php` if the database is not set up but the settings files *are* set up, so launching with `/install.php` gets you started with an installation. You can also run `drush site-install`, then `ddev exec drush site-install --yes`.

    See [Importing a Database](./usage/managing-projects.md#importing-a-database).

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

    ```bash
    mkdir my-ee-site && cd my-ee-site
    # Download the zip archive for ExpressionEngine at https://github.com/ExpressionEngine/ExpressionEngine/releases/latest unarchive and move its content into the root of the my-ee-site directory
    ddev config --database=mysql:8.0
    ddev start
    ddev launch /admin.php # Open installation wizard in browser
    ```

    When the installation wizard prompts for database settings, enter `db` for the _DB Server Address_, _DB Name_, _DB Username_, and _DB Password_.

    Visit your site.

=== "ExpressionEngine Git Checkout"

    Follow these steps based on the [ExpressionEngine Git Repository README.md](https://github.com/ExpressionEngine/ExpressionEngine#how-to-install):

    ```bash
    git clone https://github.com/ExpressionEngine/ExpressionEngine my-ee-site # for example
    cd my-ee-site
    ddev config # Accept the defaults
    ddev start
    ddev composer install
    touch system/user/config/config.php
    echo "EE_INSTALL_MODE=TRUE" >.env.php
    ddev start
    ddev launch /admin.php  # Open installation wizard in browser
    ```

    When the installation wizard prompts for database settings, enter `db` for the _DB Server Address_, _DB Name_, _DB Username_, and _DB Password_.

## Grav

=== "Composer"

    ```bash
    mkdir my-grav-site && cd my-grav-site
    ddev config --omit-containers=db
    ddev start
    ddev composer create getgrav/grav
    ddev exec gpm install admin -y
    ddev launch
    ```

=== "Git Clone"

    ```bash
    mkdir my-grav-site && cd my-grav-site
    git clone -b master https://github.com/getgrav/grav.git .
    ddev config --omit-containers=db
    ddev start
    ddev composer install
    ddev exec grav install
    ddev exec gpm install admin -y
    ddev launch
    ```

!!!tip "How to update?"
    Upgrade Grave core:

    ```bash
    ddev exec gpm selfupgrade -f
    ```

    Update plugins and themes:

    ```bash
    ddev exec gpm update -f
    ```

Visit the [Grav Documentation](https://learn.getgrav.org/17) for more information about Grav in general and visit [Local Development with DDEV](https://learn.getgrav.org/17/webservers-hosting/local-development-with-ddev) for more details about the usage of Grav with DDEV.

## Ibexa DXP

Install [Ibexa DXP](https://www.ibexa.co) OSS Edition.

```bash
mkdir my-ibexa-site && cd my-ibexa-site
ddev config --project-type=php --docroot=public --web-environment-add DATABASE_URL=mysql://db:db@db:3306/db
ddev start
ddev composer create ibexa/oss-skeleton
ddev exec console ibexa:install
ddev exec console ibexa:graphql:generate-schema
ddev launch /admin/login
```

In the web browser, log into your account using `admin` and `publish`.

Visit [Ibexa documentation](https://doc.ibexa.co/en/latest/getting_started/install_with_ddev/) for more cases.

## Joomla

```bash
mkdir my-joomla-site && cd my-joomla-site
tag=$(curl -L "https://api.github.com/repos/joomla/joomla-cms/releases/latest" | docker run -i --rm ddev/ddev-utilities jq -r .tag_name) && curl -L "https://github.com/joomla/joomla-cms/releases/download/$tag/Joomla_$tag-Stable-Full_Package.zip" -o joomla.zip
unzip ./joomla.zip && rm joomla.zip
ddev config --project-type=php --webserver-type=apache-fpm --upload-dirs=images
ddev start
ddev php installation/joomla.php install --site-name="My Joomla Site" --admin-user="Administrator" --admin-username=admin --admin-password=AdminAdmin1! --admin-email=admin@example.com --db-type=mysql --db-encryption=0 --db-host=db --db-user=db --db-pass="db" --db-name=db --db-prefix=ddev_ --public-folder=""
ddev launch /administrator
```

## Kirby CMS

Start a new [Kirby CMS](https://getkirby.com) project or use an existing one.

=== "New projects"

    Create a new Kirby CMS project from the official [Starterkit](https://github.com/getkirby/starterkit) using DDEV’s [`composer create` command](../users/usage/commands.md#composer):

    ```bash
    # Create a new project directory and navigate into it
    mkdir my-kirby-site && cd my-kirby-site

    # Set up the DDEV environment
    ddev config --omit-containers=db

    # Spin up the project and install the Kirby Starterkit
    ddev start
    ddev composer create getkirby/starterkit

    # Open the site in your browser
    ddev launch
    ```

=== "Existing projects"

    You can start using DDEV with an existing project as well:

    ```bash
    # Navigate to a existing project directory (or clone/download an existing project):
    cd my-kirby-site

    # Set up the DDEV environment
    ddev config --omit-containers=db

    # Spin up the project
    ddev start

    # Open the site in your browser
    ddev launch
    ```

!!!tip "Installing Kirby"
    Read more about developing your Kirby project with DDEV in our [extensive DDEV guide](https://getkirby.com/docs/cookbook/setup/ddev).

## Laravel

Use a new or existing Composer project, or clone a Git repository.

The Laravel project type can be used for [StarterKits](https://laravel.com/docs/starter-kits), [Laravel Livewire](https://livewire.laravel.com/) and others, as it is used with basic Laravel. DDEV automatically updates or creates the `.env` file with the database information.

=== "Composer"

    ```bash
    mkdir my-laravel-site && cd my-laravel-site
    ddev config --project-type=laravel --docroot=public
    ddev composer create "laravel/laravel:^11"
    ddev launch
    ```

=== "Git Clone"

    ```bash
    git clone <my-laravel-repo> my-laravel-site
    cd my-laravel-site
    ddev config --project-type=laravel --docroot=public
    ddev start
    ddev composer install
    ddev php artisan key:generate
    ddev launch
    ```

!!!tip "Want to use a SQLite database for Laravel?"
    DDEV defaults to using a MariaDB database to better represent a production environment.

    To select the [Laravel 11 defaults](https://laravel.com/docs/11.x/releases#application-defaults) for SQLite, use this command for `ddev config`:
    ```bash
    ddev config --project-type=laravel --docroot=public --omit-containers=db --disable-settings-management=true
    ```

!!!tip "Add Vite support?"
    Since Laravel v9.19, Vite is included as the default [asset bundler](https://laravel.com/docs/master/vite). There are small tweaks needed in order to use it: [Working with Vite in DDEV - Laravel](https://ddev.com/blog/working-with-vite-in-ddev/#laravel).

## Magento

=== "Magento 2"

    Normal details of a Composer build for Magento 2 are on the [Magento 2 site](https://experienceleague.adobe.com/docs/commerce-operations/installation-guide/composer.html). You must have a public and private key to install from Magento’s repository. When prompted for “username” and “password” in `composer create`, it’s asking for your public key as "username" and private key as "password".

    Note that you can install the Adobe/Magento composer credentials in your global `~/.ddev/homeadditions/.composer/auth.json` and never have to find them again. See [In-Container Home Directory and Shell Configuration](extend/in-container-configuration.md).

    ```bash
    mkdir my-magento2-site && cd my-magento2-site
    ddev config --project-type=magento2 --docroot=pub --disable-settings-management \
    --upload-dirs=media --web-environment-add=COMPOSER_HOME="/var/www/html/.ddev/homeadditions/.composer"

    ddev get ddev/ddev-elasticsearch
    ddev start
    ddev composer create --repository=https://repo.magento.com/ magento/project-community-edition
    rm -f app/etc/env.php
    echo "/auth.json" >.ddev/homeadditions/.composer/.gitignore

    # Change the base-url below to your project's URL
    ddev magento setup:install --base-url="https://my-magento2-site.ddev.site/" \
    --cleanup-database --db-host=db --db-name=db --db-user=db --db-password=db \
    --elasticsearch-host=elasticsearch --search-engine=elasticsearch7 --elasticsearch-port=9200 \
    --admin-firstname=Magento --admin-lastname=User --admin-email=user@example.com \
    --admin-user=admin --admin-password=Password123 --language=en_US

    ddev magento deploy:mode:set developer
    ddev magento module:disable Magento_TwoFactorAuth Magento_AdminAdobeImsTwoFactorAuth
    ddev config --disable-settings-management=false
    ddev php bin/magento info:adminuri
    # Append the URI returned by the previous command either to ddev launch, like for example ddev launch /admin_XXXXXXX, or just run ddev launch and append the URI to the path in the browser
    ddev launch
    ```

    Change the admin name and related information as needed.

    The admin login URL is specified by `frontName` in `app/etc/env.php`.

    You may want to add the [Magento 2 Sample Data](https://experienceleague.adobe.com/docs/commerce-operations/installation-guide/next-steps/sample-data/composer-packages.html) with:

    ```
    ddev magento sampledata:deploy
    ddev magento setup:upgrade
    ```

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

    OpenMage is a huge codebase, and we recommend [using Mutagen for performance](install/performance.md#mutagen) on macOS and traditional Windows.

## Moodle

=== "Composer"

    ```bash
    mkdir my-moodle-site && cd my-moodle-site
    ddev config --composer-root=public --docroot=public --webserver-type=apache-fpm
    ddev start
    ddev composer create moodle/moodle
    ddev exec 'php public/admin/cli/install.php --non-interactive --agree-license --wwwroot=$DDEV_PRIMARY_URL --dbtype=mariadb --dbhost=db --dbname=db --dbuser=db --dbpass=db --fullname="DDEV Moodle Demo" --shortname=Demo --adminpass=password'
    ddev launch /login
    ```

    In the web browser, log into your account using `admin` and `password`.

    Visit the [Moodle Admin Quick Guide](https://docs.moodle.org/400/en/Admin_quick_guide) for more information.

    !!!tip
        Moodle relies on a periodic cron job—don’t forget to set that up! See [ddev/ddev-cron](https://github.com/ddev/ddev-cron).

## Pimcore

=== "Composer"

    Using the [Pimcore skeleton](https://github.com/pimcore/skeleton) repository:

    ``` bash
    mkdir my-pimcore-site && cd my-pimcore-site
    ddev config --docroot=public

    ddev start
    ddev composer create pimcore/skeleton
    ddev exec pimcore-install --mysql-username=db --mysql-password=db --mysql-host-socket=db --mysql-database=db --admin-password=admin --admin-username=admin --no-interaction
    echo "web_extra_daemons:
      - name: consumer
        command: 'while true; do /var/www/html/bin/console messenger:consume pimcore_core pimcore_maintenance pimcore_scheduled_tasks pimcore_image_optimize pimcore_asset_update --memory-limit=250M --time-limit=3600; done'
        directory: /var/www/html" >.ddev/config.pimcore.yaml

    ddev start
    ddev launch /admin
    ```

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
* If your app requires settings, you can add them as environment variables, or otherwise configure your app to use the database, etc. (Database settings are host: `db`, database: `db`, user: `db`, password `db` no matter whether you're using PostgreSQL, MariaDB, or MySQL.)
* You can watch `pip install` output in real time on that first slow startup with `ddev logs -f` in another window.
* If your `requirements.txt` includes `psycopg2` it requires build tools, so either set `ddev config --web-extra-packages=build-essential` or change your requirement to `psycopg2-binary`.

## Shopware

=== "Composer"

    Though you can set up a Shopware 6 environment many ways, we recommend the following technique. DDEV creates a `.env.local` file for you by default; if you already have one DDEV adds necessary information to it. When `ddev composer create` asks if you want to include Docker configuration, answer `x`, as this approach does not use their Docker configuration.

    ```bash
    mkdir my-shopware-site && cd my-shopware-site
    ddev config --project-type=shopware6 --docroot=public
    ddev composer create shopware/production:^v6.5
    # If it asks `Do you want to include Docker configuration from recipes?`
    # answer `x`, as we're using DDEV for this rather than its recipes.
    ddev exec console system:install --basic-setup
    ddev launch /admin
    # Default username and password are `admin` and `shopware`
    ```

    Log into the admin site (`/admin`) using the web browser. The default credentials are username `admin` and password `shopware`. You can use the web UI to install sample data or accomplish many other tasks.

    For more advanced tasks like adding elasticsearch, building and watching storefront and administration, see [susi.dev](https://susi.dev/ddev-shopware-6).

## Silverstripe

Use a new or existing Composer project, or clone a Git repository.

=== "Composer"

    ```bash
    mkdir my-silverstripe-site && cd my-silverstripe-site
    ddev config --project-type=silverstripe --docroot=public
    ddev start
    ddev composer create --prefer-dist silverstripe/installer
    ddev sake dev/build flush=all
    ddev launch /admin
    ```

=== "Git Clone"

    ```bash
    git clone <my-silverstripe-repo> my-silverstripe-site
    cd my-silverstripe-site
    ddev config --project-type=silverstripe --docroot=public
    ddev start
    ddev composer install
    ddev sake dev/build flush=all
    ```

Your Silverstripe project is now ready.
The CMS can be found at /admin, log into the default admin account using `admin` and `password`.

Visit the [Silverstripe documentation](https://userhelp.silverstripe.org/en/5/) for more information.

`ddev sake` can be used as a shorthand for the Silverstripe Make command `ddev exec vendor/bin/sake`

To open the CMS directly from CLI, run `ddev launch /admin`.

## Statamic

Use a new or existing Composer project, or clone a Git repository.

The Laravel project type can be used for [Statamic](https://statamic.com/) like it can for Laravel. DDEV automatically updates or creates the `.env` file with the database information.

=== "Composer"

    ```bash
    mkdir my-statamic-site && cd my-statamic-site
    ddev config --project-type=laravel --docroot=public
    ddev composer create --prefer-dist statamic/statamic
    ddev php please make:user
    ddev launch /cp
    ```
=== "Git Clone"

    ```bash
    git clone <my-statamic-repo> my-statamic-site
    cd my-statamic-site
    ddev config --project-type=laravel --docroot=public
    ddev start
    ddev composer install
    ddev exec "php artisan key:generate"
    ddev launch /cp
    ```

## Sulu

```bash
mkdir my-sulu-site && cd my-sulu-site
ddev config --project-type=php --docroot=public --upload-dirs=uploads --database=mysql:8.0
ddev start
ddev composer create sulu/skeleton
```

Create your default webspace configuration `mv config/webspaces/example.xml config/webspaces/my-sulu-site.xml` and adjust the values for `<name>` and `<key>` so that they are matching your project:

```bash
<?xml version="1.0" encoding="utf-8"?>
<webspace xmlns="http://schemas.sulu.io/webspace/webspace"
          xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
          xsi:schemaLocation="http://schemas.sulu.io/webspace/webspace http://schemas.sulu.io/webspace/webspace-1.1.xsd">
    <!-- See: http://docs.sulu.io/en/latest/book/webspaces.html how to configure your webspace-->

    <name>My Sulu CMS</name>
    <key>my-sulu-cms</key>
```

!!!warning "Caution"
    Changing the `<key>` for a webspace later on causes problems. It is recommended to decide on the value for the key before the database is build in the next step.

The information for the database connection is set in the environment variable `DATABASE_URL`. The installation will have created a `.env.local` file.  Set `DATABASE_URL` in the `.env.local` file so it looks like this:

```bash
APP_ENV=dev
DATABASE_URL="mysql://db:db@db:3306/db?serverVersion=8.0&charset=utf8mb4"
```

Now build the database. Building with the `dev` argument adds a user `admin`with the the password `admin` to your project.

```bash
ddev exec bin/adminconsole sulu:build dev
ddev launch /admin
```

!!!tip
    If you don't want to add an admin user use the `prod` argument instead

    ```bash
    ddev execute bin/adminconsole sulu:build prod
    ```

## Symfony

There are many ways to install Symfony, here are a few of them based on the [Symfony docs](https://symfony.com/doc/current/setup.html).

If your project uses a database you'll want to set the [DB connection string](https://symfony.com/doc/current/doctrine.html#configuring-the-database) in the `.env`. If using the default MariaDB configuration, you'll want `DATABASE_URL="mysql://db:db@db:3306/db?serverVersion=10.11"`. If you're using a different database type or version, see `ddev describe` for the type and version.

=== "Composer"

    ```bash
    mkdir my-symfony-site && cd my-symfony-site
    ddev config --docroot=public
    ddev composer create symfony/skeleton
    ddev composer require webapp
    # When it asks if you want to include docker configuration, say "no" with "x"
    ddev launch
    ```

=== "Symfony CLI"

    ```bash
    mkdir my-symfony-site && cd my-symfony-site
    ddev config --docroot=public
    ddev start
    ddev exec symfony check:requirements
    ddev exec symfony new temp --version="7.0.*" --webapp
    ddev exec 'rsync -rltgopD temp/ ./ && rm -rf temp'
    ddev launch
    ```

=== "Git Clone"

    ```bash
    git clone <my-symfony-repo> my-symfony-site
    cd my-symfony-site
    ddev config --docroot=public
    ddev start
    ddev composer install
    ddev launch
    ```

## TYPO3

TYPO3 provides a [detailed DDEV installation guide](https://docs.typo3.org/m/typo3/tutorial-getting-started/main/en-us/Installation/TutorialDdev.html) for each major version.

=== "Composer"

    ```bash
    mkdir my-typo3-site && cd my-typo3-site
    ddev config --project-type=typo3 --docroot=public --php-version=8.3
    ddev start
    ddev composer create "typo3/cms-base-distribution"
    ddev exec touch public/FIRST_INSTALL
    ddev launch
    ```

=== "Git Clone"

    ```bash
    git clone https://github.com/example/example-site my-typo3-site
    cd my-typo3-site
    ddev config --project-type=typo3 --docroot=public --php-version=8.3
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
    mkdir my-wp-site && cd my-wp-site

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
    mkdir my-wp-bedrock-site && cd my-wp-bedrock-site
    ddev config --project-type=wordpress --docroot=web
    ddev start
    ddev composer create roots/bedrock
    ```

    Rename the file `.env.example` to `.env` in the project root and make the following adjustments:

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
    git clone https://github.com/example/my-site.git my-wp-site
    cd my-wp-site
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

    Now run [`ddev start`](../users/usage/commands.md#start) and continue [Importing a Database](./usage/managing-projects.md#importing-a-database) if you need to.
