# CMS Quickstarts

DDEV is [ready to go](./project.md) with generic project types for PHP and Python frameworks, and more specific project types for working with popular platforms and CMSes. To learn more about how to manage projects in DDEV visit [Managing Projects](../users/usage/managing-projects.md).

Before proceeding, make sure your installation of DDEV is up to date. In a new and empty project folder, using your favorite shell, run the following commands:

## Backdrop

To get started with [Backdrop](https://backdropcms.org), clone the project repository and navigate to the project directory.

=== "New projects"

    ```bash
    # Create a project directory and move into it:
    git clone https://github.com/backdrop/backdrop my-backdrop-site
    cd my-backdrop-site
    
    # Set up the DDEV environment:
    ddev config --project-type=backdrop
    
    # Boot the project and install the starter project:
    ddev start
    
    # Launch the website and step through the initial setup
    ddev launch
    ```

=== "Existing projects"

    You can start using DDEV with an existing project, too—but make sure you have a database backup handy!


    ```bash
    # Clone an existing repository (or navigate to a local project directory):
    git clone https://github.com/example/example-site example-site
    cd example-site

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
    mkdir my-cakephp-app
    cd my-cakephp-app
    ddev config --project-type=cakephp --docroot=webroot
    ddev composer create --prefer-dist cakephp/app:~5.0
    ddev cake
    ddev launch
    ```

=== "Git Clone"

    ```bash
    git clone <your-cakephp-repo>
    cd <your-cakephp-project>
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
    mkdir my-craft-project
    cd my-craft-project

    # Set up the DDEV environment:
    ddev config --project-type=craftcms --docroot=web

    # Boot the project and install the starter project:
    ddev start
    ddev composer create -y craftcms/craft
    ddev launch
    ```

    Third-party starter projects can by used the same way—substitute the package name when running `ddev composer create`.

=== "Existing projects"

    You can start using DDEV with an existing project, too—but make sure you have a database backup handy!

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

    Craft CMS projects use MySQL 8.0, by default. You can override this setting (and the PHP version) during setup with [`config` command flags](./usage/commands.md#config) or after setup via the [configuration files](./configuration/config.md).

    !!!tip "Upgrading or using a generic project type?"
        If you previously set up DDEV in a Craft project using the generic `php` project type, update the `type:` setting in `.ddev/config.yaml` to `craftcms`, then run [`ddev restart`](../users/usage/commands.md#restart) apply the changes.

### Running Craft in a Subdirectory

In order for `ddev craft` to work when Craft is installed in a subdirectory, you will need to change the location of the `craft` executable by providing the `CRAFT_CMD_ROOT` environment variable to the web container. For example, if the installation lives in `my-craft-project/app`, you would run `ddev config --web-environment-add=CRAFT_CMD_ROOT=./app`. `CRAFT_CMD_ROOT` defaults to `./`, the project root directory. Run `ddev restart` to apply the change.

Read more about customizing the environment and persisting configuration in [Providing Custom Environment Variables to a Container](https://ddev.readthedocs.io/en/latest/users/extend/customization-extendibility/#providing-custom-environment-variables-to-a-container).

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
* If your `requirements.txt` includes `psycopg2` it requires build tools, so either set `ddev config --web-extra-packages=build-essential` or change your requirement to `psycopg2-binary`.

## Drupal

For all versions of Drupal 8+ the Composer techniques work. The settings configuration is done differently for each Drupal version, but the project type is "drupal".

=== "Drupal 10"

    ```bash
    mkdir my-drupal-site
    cd my-drupal-site
    ddev config --project-type=drupal --php-version=8.3 --docroot=web
    ddev start
    ddev composer create drupal/recommended-project:^10
    ddev composer require drush/drush
    ddev drush site:install --account-name=admin --account-pass=admin -y
    # use the one-time link (CTRL/CMD + Click) from the command below to edit your admin account details.
    ddev drush uli
    ddev launch
    ```

=== "Drupal 11 (dev)"

    ```bash
    mkdir my-drupal-site
    cd my-drupal-site
    ddev config --project-type=drupal --php-version=8.3 --docroot=web --corepack-enable
    ddev start
    ddev composer create drupal/recommended-project:^11.x-dev
    ddev composer require drush/drush
    ddev drush site:install --account-name=admin --account-pass=admin -y
    # use the one-time link (CTRL/CMD + Click) from the command below to edit your admin account details.
    ddev drush uli
    ddev launch
    ```

=== "Drupal 9 (EOL)"

    ```bash
    mkdir my-drupal-site
    cd my-drupal-site
    ddev config --project-type=drupal --php-version=8.1 --docroot=web
    ddev start
    ddev composer create drupal/recommended-project:^9
    ddev composer require drush/drush
    ddev drush site:install --account-name=admin --account-pass=admin -y
    # use the one-time link (CTRL/CMD + Click) from the command below to edit your admin account details.
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

    Drupal 7 doesn’t know how to redirect from the front page to `/install.php` if the database is not set up but the settings files *are* set up, so launching with `/install.php` gets you started with an installation. You can also run `drush site-install`, then `ddev exec drush site-install --yes`.

    See [Importing a Database](../usage/managing-projects#importing-a-database).

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
    ddev config --database=mysql:8.0
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

## Grav

=== "Composer"

    ```bash
    mkdir grav
    cd grav
    ddev config --php-version=8.2 --omit-containers=db
    ddev start
    ddev composer create getgrav/grav
    ddev exec bin/gpm install admin -y
    ddev launch
    ```

=== "Git Clone"

    ```bash
    mkdir grav
    cd grav
    git clone -b master https://github.com/getgrav/grav.git .
    ddev config --php-version=8.2 --omit-containers=db
    ddev start
    ddev composer install
    ddev exec bin/grav install
    ddev exec bin/gpm install admin -y
    ddev launch
    ```

!!!tip "How to update?"
    Upgrade Grave core:

    ```bash
    ddev exec bin/gpm selfupgrade -f
    ```

    Update plugins and themes:

    ```bash
    ddev exec bin/gpm update -f
    ```

Visit the [Grav Documentation](https://learn.getgrav.org/17) for more information about Grav in general and visit [Local Development with DDEV](https://learn.getgrav.org/17/webservers-hosting/local-development-with-ddev) for more details about the usage of Grav with DDEV.

## Ibexa DXP

Install [Ibexa DXP](https://www.ibexa.co) OSS Edition.

```bash
mkdir my-ibexa-project && cd my-ibexa-project
ddev config --project-type=php --php-version 8.1 --docroot=public
ddev config --web-environment-add DATABASE_URL=mysql://db:db@db:3306/db
ddev start
ddev composer create ibexa/oss-skeleton
ddev php bin/console ibexa:install
ddev php bin/console ibexa:graphql:generate-schema
ddev launch
```

In the web browser, log into your account using `admin` and `publish`.

Visit [Ibexa documentation](https://doc.ibexa.co/en/latest/getting_started/install_with_ddev/) for more cases.

## Kirby CMS

Start a new [Kirby CMS](https://getkirby.com) project or use an existing one.

=== "New projects"

    Create a new Kirby CMS project from the official [Starterkit](https://github.com/getkirby/starterkit) using DDEV’s [`composer create` command](../users/usage/commands.md#composer):

    ```bash
    # Create a new project directory and navigate into it
    mkdir my-kirby-project
    cd my-kirby-project

    # Set up the DDEV environment
    ddev config --php-version=8.2 --omit-containers=db

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
    cd my-kirby-project

    # Set up the DDEV environment
    ddev config --php-version=8.2 --omit-containers=db

    # Spin up the project
    ddev start

    # Open the site in your browser
    ddev launch
    ```

!!!tip "Installing Kirby"
    Read more about developing your Kirby project with DDEV in our [extensive DDEV guide](https://getkirby.com/docs/cookbook/setup/ddev).

## Laravel

Use a new or existing Composer project, or clone a Git repository.

The Laravel project type can be used for [Lumen](https://lumen.laravel.com/) like it can for Laravel. DDEV automatically updates or creates the `.env` file with the database information.

=== "Composer"

    ```bash
    mkdir my-laravel-app
    cd my-laravel-app
    ddev config --project-type=laravel --docroot=public --php-version=8.2
    ddev composer create --prefer-dist laravel/laravel:^11
    ddev launch
    ```

=== "Git Clone"

    ```bash
    git clone <your-laravel-repo>
    cd <your-laravel-project>
    ddev config --project-type=laravel --docroot=public --php-version=8.2
    ddev start
    ddev composer install
    ddev php artisan key:generate
    ddev launch
    ```

!!!tip "Want to use a SQLite database for Laravel?"
    DDEV defaults to using a MariaDB database to better represent a production environment.

    To select the [Laravel 11 defaults](https://laravel.com/docs/11.x/releases#application-defaults) for SQLite, use this command for `ddev config`:
    ```bash
    ddev config --project-type=laravel --docroot=public --php-version=8.2 --omit-containers=db --disable-settings-management=true
    ```

## Magento

=== "Magento 2"

    Normal details of a Composer build for Magento 2 are on the [Magento 2 site](https://experienceleague.adobe.com/docs/commerce-operations/installation-guide/composer.html). You must have a public and private key to install from Magento’s repository. When prompted for “username” and “password” in `composer create`, it’s asking for your public key as "username" and private key as "password".

    Note that you can install the Adobe/Magento composer credentials in your global `~/.ddev/homeadditions/.composer/auth.json` and never have to find them again. See [In-Container Home Directory and Shell Configuration](extend/in-container-configuration.md).

    ```bash
    SITENAME=ddev-magento2
    mkdir -p ${SITENAME} && cd ${SITENAME}
    ddev config --project-type=magento2 --php-version=8.2 --database=mariadb:10.6 --docroot=pub --disable-settings-management --upload-dirs=media
    ddev config --web-environment-add=COMPOSER_HOME="/var/www/html/.ddev/homeadditions/.composer"
    ddev get ddev/ddev-elasticsearch
    ddev start
    ddev composer create --repository=https://repo.magento.com/ magento/project-community-edition -y
    rm -f app/etc/env.php
    echo "/auth.json" >.ddev/homeadditions/.composer/.gitignore

    # Change the base-url below to your project's URL
    ddev magento setup:install --base-url="https://${SITENAME}.ddev.site/" --cleanup-database --db-host=db --db-name=db --db-user=db --db-password=db --elasticsearch-host=elasticsearch --search-engine=elasticsearch7 --elasticsearch-port=9200 --admin-firstname=Magento --admin-lastname=User --admin-email=user@example.com --admin-user=admin --admin-password=Password123 --language=en_US

    ddev magento deploy:mode:set developer
    ddev magento module:disable Magento_TwoFactorAuth Magento_AdminAdobeImsTwoFactorAuth
    ddev config --disable-settings-management=false
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

    OpenMage is a huge codebase, and we recommend [using Mutagen for performance](install/performance.md#using-mutagen) on macOS and traditional Windows.

## Moodle

=== "Composer"

    ```bash
    ddev config --composer-root=public --docroot=public --webserver-type=apache-fpm
    ddev start
    ddev composer create moodle/moodle -y
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
mkdir my-pimcore && cd my-pimcore
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
    mkdir my-shopware6 && cd my-shopware6
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
    mkdir my-silverstripe-app
    cd my-silverstripe-app
    ddev config --project-type=silverstripe --docroot=public
    ddev start
    ddev composer create --prefer-dist silverstripe/installer -y
    ddev sake dev/build flush=all
    ddev launch /admin
    ```

=== "Git Clone"

    ```bash
    git clone <your-silverstripe-repo>
    cd <your-silverstripe-project>
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
    mkdir my-statamic-app
    cd my-statamic-app
    ddev config --project-type=laravel --docroot=public
    ddev composer create --prefer-dist statamic/statamic
    ddev php please make:user
    ddev launch /cp
    ```
=== "Git Clone"

    ```bash
    git clone <your-statamic-repo>
    cd <your-statamic-project>
    ddev config --project-type=laravel --docroot=public
    ddev start
    ddev composer install
    ddev exec "php artisan key:generate"
    ddev launch
    ```

## Symfony

There are many ways to install Symfony, here are a few of them based on the [Symfony docs](https://symfony.com/doc/current/setup.html).

If your project uses a database you'll want to set the [DB connection string](https://symfony.com/doc/current/doctrine.html#configuring-the-database) in the `.env`. If using the default MariaDB configuration, you'll want `DATABASE_URL="mysql://db:db@db:3306/db?serverVersion=10.11"`. If you're using a different database type or version, see `ddev describe` for the type and version.

=== "Composer"

    ```bash
    mkdir my-symfony && cd my-symfony
    ddev config --docroot=public
    ddev composer create symfony/skeleton:"7.0.*"
    ddev composer require webapp
    # When it asks if you want to include docker configuration, say "no" with "x"
    ddev launch
    ```

=== "Symfony CLI"

    In a future release the Symfony CLI will be provided by default in `ddev-webserver`, but for now it needs to be configured.

    ```bash
    mkdir my-symfony && cd my-symfony
    ddev config --docroot=public
    echo "RUN curl -1sLf 'https://dl.cloudsmith.io/public/symfony/stable/setup.deb.sh' | sudo -E bash
    RUN sudo apt install -y symfony-cli" >.ddev/web-build/Dockerfile.symfony-cli
    ddev restart
    ddev exec symfony check:requirements
    ddev exec symfony new temp --version="7.0.*" --webapp
    ddev exec 'rsync -rltgopD temp/ ./ && rm -rf temp'
    ddev launch
    ```

=== "Git Clone"

    ```bash
    git clone <your-symfony-repo>
    cd <your-symfony-repo>
    ddev config --docroot=public --php-version=8.3
    ddev start
    ddev composer install
    ddev launch
    ```

## TYPO3

=== "Composer"

    ```bash
    mkdir my-typo3-site
    cd my-typo3-site
    ddev config --project-type=typo3 --docroot=public --php-version 8.3
    ddev start
    ddev composer create "typo3/cms-base-distribution"
    ddev exec touch public/FIRST_INSTALL
    ddev launch
    ```

=== "Git Clone"

    ```bash
    git clone https://github.com/example/example-site
    cd example-site
    ddev config --project-type=typo3 --docroot=public --php-version 8.3
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
