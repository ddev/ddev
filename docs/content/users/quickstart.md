# CMS Quickstarts

DDEV is [ready to go](./project.md) with generic project types for PHP frameworks, and more specific project types for working with popular platforms and CMSes. To learn more about how to manage projects in DDEV visit [Managing Projects](../users/usage/managing-projects.md).

Before proceeding, make sure your installation of DDEV is up to date. In a new and empty project folder, using your favorite shell, run the following commands:

## Backdrop

You can start a new [Backdrop](https://backdropcms.org) project or configure an existing one.

=== "New projects"

    ```bash
    mkdir my-backdrop-site && cd my-backdrop-site
    ddev config --project-type=backdrop
    # Add the official Bee CLI add-on
    ddev add-on get backdrop-ops/ddev-backdrop-bee
    ddev start
    # Download Backdrop core
    ddev bee download-core
    # Create admin user
    ddev bee si --username=admin --password=Password123 --db-name=db --db-user=db --db-pass=db --db-host=db --auto
    # Login using `admin` user and `Password123` password
    ddev launch
    ```

=== "Existing projects"

    You can start using DDEV with an existing project, too—but make sure you have a database backup handy!


    ```bash
    mkdir my-backdrop-site && cd my-backdrop-site

    # Use your own repository URL, this is an example
    git clone https://github.com/ddev/test-backdrop.git .

    # Set up the DDEV environment:
    ddev config --project-type=backdrop

    # Add the official Bee CLI add-on
    ddev add-on get backdrop-ops/ddev-backdrop-bee

    # Start the project
    ddev start

    # Import a database backup:
    ddev import-db --file=/path/to/db.sql.gz

    # Import files backup
    ddev import-files --source=/path/to/files.tar.gz

    # open the site in your browser
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
    ddev start
    ddev composer create-project --prefer-dist --no-interaction cakephp/app:~5.0
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

## CiviCRM (Standalone)

[CiviCRM Standalone](https://civicrm.org/blog/ufundo/next-steps-civicrm-standalone) allows running [CiviCRM](https://civicrm.org/) without a CMS. Visit [Install CiviCRM (Standalone)](https://docs.civicrm.org/installation/en/latest/standalone) for more installation details.

```bash
mkdir my-civicrm-site && cd my-civicrm-site
ddev config --project-type=php --composer-root=core --upload-dirs=public/media
ddev start
ddev exec "curl -LsS https://download.civicrm.org/latest/civicrm-STABLE-standalone.tar.gz -o /tmp/civicrm-standalone.tar.gz"
ddev exec "tar --strip-components=1 -xzf /tmp/civicrm-standalone.tar.gz"
ddev composer require civicrm/cli-tools --no-scripts
# You can now install CiviCRM manually in your browser using `ddev launch`
# and selecting `db` for the server and `db` for database/username/password
# or do the same automatically using the command below:
# The parameter `-m loadGenerated=1` includes sample data
ddev exec cv core:install \
    --cms-base-url='$DDEV_PRIMARY_URL' \
    --db=mysql://db:db@db/db \
    -m loadGenerated=1 \
    -m extras.adminUser=admin \
    -m extras.adminPass=admin \
    -m extras.adminEmail=admin@example.com
ddev launch
```

## Contao

Further information on the DDEV procedure can also be found in the [Contao documentation](https://docs.contao.org/manual/en/guides/local-installation/ddev/).

=== "Composer"

    ```bash
    mkdir my-contao-site && cd my-contao-site
    ddev config --project-type=php --docroot=public --webserver-type=apache-fpm --php-version=8.2
    ddev composer create-project contao/managed-edition:5.3

    # Set DATABASE_URL and MAILER_DSN in .env.local
    ddev dotenv set .env.local --database-url=mysql://db:db@db:3306/db --mailer-dsn=smtp://localhost:1025

    # Create the database
    ddev exec contao-console contao:migrate --no-interaction

    # Create backend user
    ddev exec contao-console contao:user:create --username=admin --name=Administrator --email=admin@example.com --language=en --password=Password123 --admin

    # Access the administration area
    ddev launch contao
    ```

=== "Contao Manager"

    Like most PHP projects, Contao could be installed and updated with Composer. The [Contao Manager](https://docs.contao.org/manual/en/installation/contao-manager/) is a tool that provides a graphical user interface to manage a Contao installation.

    ```bash
    mkdir my-contao-site && cd my-contao-site
    ddev config --project-type=php --docroot=public --webserver-type=apache-fpm --php-version=8.2

    # set DATABASE_URL and MAILER_DSN in .env.local
    ddev dotenv set .env.local --database-url=mysql://db:db@db:3306/db --mailer-dsn=smtp://localhost:1025

    # Download the Contao Manager
    ddev start
    ddev exec "wget -O public/contao-manager.phar.php https://download.contao.org/contao-manager/stable/contao-manager.phar"

    # Follow the further steps within the Contao Manager
    ddev launch contao-manager.phar.php
    ```

=== "Demo Website"

    The [Contao demo website](https://demo.contao.org/) is maintained for the currently supported Contao versions and can be [optionally installed](https://github.com/contao/contao-demo).
    Via the Contao Manager you can select this option during the first installation.

## Craft CMS

Start a new [Craft CMS](https://craftcms.com) project or retrofit an existing one.

!!!tip "Compatibility with Craft CMS 3"
    The `craftcms` project type is best with Craft CMS 4+, which is more opinionated about some settings. If you are using Craft CMS 3 or earlier, you may want to use the `php` project type and [manage settings yourself](https://github.com/ddev/ddev/issues/4650).

Environment variables will be automatically added to the `.ddev/.env.web` file to simplify project configuration. This means that the primary site URL and the database connection settings can be used without any modification. To disable this behavior, see [`disable_settings_management`](./configuration/config.md#disable_settings_management).

=== "New projects"

    ```bash
    # Create a project directory and move into it:
    mkdir my-craft-site && cd my-craft-site

    # Set up the DDEV environment:
    ddev config --project-type=craftcms --docroot=web

    # Boot the project
    ddev start

    # Install the starter project:
    ddev composer create-project --no-scripts craftcms/craft

    # Install Craft CMS:
    ddev craft install \
        --username=admin \
        --password=Password123 \
        --email=admin@example.com \
        --site-name='My Craft Site' \
        --language=en \
        --site-url='$PRIMARY_SITE_URL'

    # Login using `admin` user and `Password123` password
    ddev launch /admin
    ```

    Third-party starter projects can by used the same way—substitute the package name when running `ddev composer create-project`.

=== "Existing projects"

    You can start using DDEV with an existing project, too—but make sure you have a database backup handy!

    ```bash
    # Clone an existing repository (or navigate to a local project directory):
    git clone https://github.com/example/example-site my-craft-site
    cd my-craft-site

    # Set up the DDEV environment:
    ddev config --project-type=craftcms

    # Boot the project and install Composer packages:
    ddev composer install

    # Import a database backup and open the site in your browser:
    ddev craft db/restore /path/to/db.sql.gz
    ddev launch
    ```

    Craft CMS projects use MySQL 8.0, by default. You can override this setting (and the PHP version) during setup with [`config` command flags](./usage/commands.md#config) or after setup via the [configuration files](./configuration/config.md).


### Running Craft in a Subdirectory

In order for `ddev craft` to work when Craft is installed in a subdirectory, you will need to change the location of the `craft` executable by providing the `CRAFT_CMD_ROOT` environment variable to the web container. For example, if the installation lives in `my-craft-site/app`, you would run `ddev config --web-environment-add=CRAFT_CMD_ROOT=./app`. `CRAFT_CMD_ROOT` defaults to `./`, the project root directory. Run `ddev restart` to apply the change.

Read more about customizing the environment and persisting configuration in [Providing Custom Environment Variables to a Container](./extend/customization-extendibility.md#environment-variables-for-containers-and-services).

!!!tip "Installing Craft"
    Read more about installing Craft in the [official documentation](https://craftcms.com/docs).

## Drupal

=== "Drupal 11"

    ```bash
    mkdir my-drupal-site && cd my-drupal-site
    ddev config --project-type=drupal11 --docroot=web
    ddev start
    ddev composer create-project "drupal/recommended-project:^11"
    ddev composer require drush/drush
    ddev drush site:install --account-name=admin --account-pass=admin -y
    ddev launch
    # or automatically log in with
    ddev launch $(ddev drush uli)
    ```

    Read more about: [Drupal Core](https://new.drupal.org/about/overview/technical) & [Documentation](https://www.drupal.org/docs)

=== "Drupal CMS"

    ```bash
    mkdir my-drupal-site && cd my-drupal-site
    ddev config --project-type=drupal11 --docroot=web
    ddev start
    ddev composer create-project drupal/cms
    ddev launch
    ```

    Read more about: [Drupal CMS](https://new.drupal.org/drupal-cms) & [Documentation](https://new.drupal.org/docs/drupal-cms)

=== "Drupal 10"

    ```bash
    mkdir my-drupal-site && cd my-drupal-site
    ddev config --project-type=drupal10 --docroot=web
    ddev start
    ddev composer create-project "drupal/recommended-project:^10"
    ddev composer require drush/drush
    ddev drush site:install --account-name=admin --account-pass=admin -y
    ddev launch
    # or automatically log in with
    ddev launch $(ddev drush uli)
    ```

    Read more about: [Drupal Core](https://new.drupal.org/about/overview/technical) & [Documentation](https://www.drupal.org/docs)

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
    PROJECT_GIT_URL=https://github.com/ddev/test-drupal11.git
    git clone ${PROJECT_GIT_URL} my-drupal-site
    cd my-drupal-site
    ddev config --project-type=drupal11 --docroot=web
    ddev start
    ddev composer install # If a composer build
    ddev drush site:install --account-name=admin --account-pass=admin -y
    ddev launch
    ```

    Read more about: [Drupal Core](https://new.drupal.org/about/overview/technical) & [Documentation](https://www.drupal.org/docs)

## ExpressionEngine

=== "ExpressionEngine ZIP File Download"

    ```bash
    mkdir my-ee-site && cd my-ee-site
    curl -o ee.zip -L $(curl -sL https://api.github.com/repos/ExpressionEngine/ExpressionEngine/releases/latest | docker run -i --rm ddev/ddev-utilities jq -r '.assets | map(select(.name | test("^ExpressionEngine.*\\.zip$")))[0].browser_download_url')
    unzip ee.zip && rm -f ee.zip
    ddev config --database=mysql:8.0
    ddev start
    ddev launch /admin.php # Open installation wizard in browser
    ```

    When the installation wizard prompts for database settings, enter `db` for the _DB Server Address_, _DB Name_, _DB Username_, and _DB Password_.

    Visit your site.

=== "ExpressionEngine Git Checkout"

    Follow these steps based on the [ExpressionEngine Git Repository README.md](https://github.com/ExpressionEngine/ExpressionEngine#how-to-install):

    ```bash
    mkdir my-ee-site && cd my-ee-site
    git clone https://github.com/ExpressionEngine/ExpressionEngine .
    ddev config --database=mysql:8.0
    ddev start
    ddev composer install
    touch system/user/config/config.php
    echo "EE_INSTALL_MODE=TRUE" >.env.php
    ddev launch /admin.php  # Open installation wizard in browser
    ```

    When the installation wizard prompts for database settings, enter `db` for the _DB Server Address_, _DB Name_, _DB Username_, and _DB Password_.

## Generic (FrankenPHP)

This example of the `webserver_type: generic` puts [FrankenPHP](https://frankenphp.dev/) into DDEV as an experimental first step in using the innovative Golang-based PHP interpreter. It is in its infancy and may someday become a full-fledged `webserver_type`. Your feedback and improvements are welcome.

This particular example uses a `drupal11` project with FrankenPHP, which then uses its own PHP 8.4 interpreter. The normal DDEV database container is used for database access.

In this example, inside the web container the normal `php` CLI use used for CLI activities. Xdebug (and `ddev xdebug`) do not yet work.

The `generic` `webserver_type` is used here, so the `ddev-webserver` does not start the `nginx` or `php-fpm` daemons, and the `frankenphp` process does all the work.

```bash
export FRANKENPHP_SITENAME=my-frankenphp-site
mkdir ${FRANKENPHP_SITENAME} && cd ${FRANKENPHP_SITENAME}
ddev config --project-type=drupal11 --webserver-type=generic --docroot=web --php-version=8.4
ddev start

cat <<'EOF' > .ddev/config.frankenphp.yaml
web_extra_daemons:
    - name: "frankenphp"
      command: "frankenphp php-server --listen=0.0.0.0:80 --root=\"/var/www/html/${DDEV_DOCROOT:-}\" -v -a"
      directory: /var/www/html
web_extra_exposed_ports:
    - name: "frankenphp"
      container_port: 80
      http_port: 80
      https_port: 443
EOF

cat <<'DOCKERFILEEND' >.ddev/web-build/Dockerfile.frankenphp
RUN curl -s https://frankenphp.dev/install.sh | sh
RUN mv frankenphp /usr/local/bin/
RUN mkdir -p /usr/local/etc && ln -s /etc/php/${DDEV_PHP_VERSION}/fpm /usr/local/etc/php
DOCKERFILEEND

ddev composer create-project drupal/recommended-project
ddev composer require drush/drush
ddev restart
ddev drush site:install demo_umami --account-name=admin --account-pass=admin -y
ddev launch
# or automatically log in with
ddev launch $(ddev drush uli)
```

## Grav

=== "Composer"

    ```bash
    mkdir my-grav-site && cd my-grav-site
    ddev config --omit-containers=db
    ddev start
    ddev composer create-project getgrav/grav
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
    Upgrade Grav core:

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
ddev composer create-project ibexa/oss-skeleton
ddev exec console ibexa:install
ddev exec console ibexa:graphql:generate-schema
ddev launch /admin/login
```

In the web browser, log into your account using `admin` and `publish`.

Visit [Ibexa documentation](https://doc.ibexa.co/en/latest/getting_started/install_with_ddev/) for more cases.

## Joomla

```bash
mkdir my-joomla-site && cd my-joomla-site
# Download the latest version of Joomla! and unzip it.
# This can be manually downloaded from https://downloads.joomla.org/ or done using curl as here.
curl -o joomla.zip -L $(curl -sL https://api.github.com/repos/joomla/joomla-cms/releases/latest | docker run -i --rm ddev/ddev-utilities jq -r '.assets | map(select(.name | test("^Joomla.*Stable-Full_Package\\.zip$")))[0].browser_download_url')
unzip joomla.zip && rm -f joomla.zip
ddev config --project-type=php --webserver-type=apache-fpm --upload-dirs=images
ddev start
ddev php installation/joomla.php install --site-name="My Joomla Site" --admin-user="Administrator" --admin-username=admin --admin-password=AdminAdmin1! --admin-email=admin@example.com --db-type=mysql --db-encryption=0 --db-host=db --db-user=db --db-pass="db" --db-name=db --db-prefix=ddev_ --public-folder=""
ddev launch /administrator
```

## Kirby CMS

Start a new [Kirby CMS](https://getkirby.com) project or use an existing one.

=== "New projects"

    Create a new Kirby CMS project from the official [Starterkit](https://github.com/getkirby/starterkit) using DDEV’s [`composer create-project` command](../users/usage/commands.md#composer):

    ```bash
    # Create a new project directory and navigate into it
    mkdir my-kirby-site && cd my-kirby-site

    # Set up the DDEV environment
    ddev config --omit-containers=db --webserver-type=apache-fpm

    # Spin up the project and install the Kirby Starterkit
    ddev start
    ddev composer create-project getkirby/starterkit

    # Open the site in your browser
    ddev launch
    ```

=== "Existing projects"

    You can start using DDEV with an existing project as well:

    ```bash
    # Navigate to a existing project directory (or clone/download an existing project):
    cd my-kirby-site

    # Set up the DDEV environment
    ddev config --omit-containers=db --webserver-type=apache-fpm

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

    Laravel defaults to SQLite, but we use MariaDB to better mimic a production environment:

    ```bash
    mkdir my-laravel-site && cd my-laravel-site
    ddev config --project-type=laravel --docroot=public
    ddev start
    ddev composer create-project "laravel/laravel:^12"
    ddev launch
    ```

=== "Composer (SQLite)"

    To use the SQLite configuration provided by Laravel:

    ```bash
    mkdir my-laravel-site && cd my-laravel-site
    ddev config --project-type=laravel --docroot=public --omit-containers=db --disable-settings-management=true
    ddev start
    ddev composer create-project "laravel/laravel:^12"
    ddev launch
    ```

    To switch an existing Laravel project to SQLite:

    ```bash
    ddev config --project-type=laravel --docroot=public --omit-containers=db --disable-settings-management=true
    ddev restart
    ddev composer run-script post-root-package-install
    ddev dotenv set .env --db-connection=sqlite
    ddev composer run-script post-create-project-cmd
    ddev launch
    ```

=== "Laravel Installer"

    ```bash
    mkdir my-laravel-site && cd my-laravel-site

    # To use MariaDB, apply the following command
    ddev config --project-type=laravel --docroot=public

    # To use SQLite, uncomment and use the following command instead
    #ddev config --project-type=laravel --docroot=public --omit-containers=db --disable-settings-management=true

    # Temporarily add the Laravel installer
    # as /usr/local/bin/laravel in the web container
    cat <<'DOCKERFILEEND' >.ddev/web-build/Dockerfile.laravel
    ARG COMPOSER_HOME=/usr/local/composer
    RUN composer global require laravel/installer
    RUN ln -s $COMPOSER_HOME/vendor/bin/laravel /usr/local/bin/laravel
    DOCKERFILEEND

    # Start the project
    ddev start

    # Follow the prompts, select a starter kit of your choice (or none),
    # and agree to run npm commands
    # (SQLite is used here as other database types would fail due to
    # the .env file not being ready, which DDEV will fix on 'ddev restart')
    ddev exec laravel new temp --database=sqlite

    # 'laravel new' can't install in the current directory right away,
    # so we use 'rsync' to move the installed files one level up
    ddev exec 'rsync -rltgopD temp/ ./ && rm -rf temp'

    # Remove the Laravel installer and the .env file
    rm -f .ddev/web-build/Dockerfile.laravel .env

    # Restart the project
    ddev restart

    # Execute the post-install actions and launch the project
    ddev composer run-script post-root-package-install
    ddev composer run-script post-create-project-cmd
    ddev launch
    ```

=== "Git Clone"

    ```bash
    git clone <my-laravel-repo> my-laravel-site
    cd my-laravel-site
    ddev config --project-type=laravel --docroot=public
    ddev start
    ddev composer install
    ddev composer run-script post-root-package-install
    ddev composer run-script post-create-project-cmd
    ddev launch
    ```

!!!tip "Add Vite support?"
    Since Laravel v9.19, Vite is included as the default [asset bundler](https://laravel.com/docs/vite). There are small tweaks needed in order to use it: [Working with Vite in DDEV - Laravel](https://ddev.com/blog/working-with-vite-in-ddev/#laravel).

## Magento 2

Normal details of a Composer build for Magento 2 are on the [Magento 2 site](https://experienceleague.adobe.com/docs/commerce-operations/installation-guide/composer.html). You must have a public and private key to install from Magento’s repository. When prompted for “username” and “password” in `composer create`, it’s asking for your public key as "username" and private key as "password".

!!!tip "Store Adobe/Magento Composer credentials in the global DDEV config"
    If you have Composer installed on your workstation and have an `auth.json` you can reuse the `auth.json` by making a symlink. See [In-Container Home Directory and Shell Configuration](extend/in-container-configuration.md):

    ```
    mkdir -p ~/.ddev/homeadditions/.composer && ln -s ~/.composer/auth.json ~/.ddev/homeadditions/.composer/auth.json
    ```

    Alternately, you can install the Adobe/Magento Composer credentials in your global `~/.ddev/homeadditions/.composer/auth.json` and never have to enter them again (see below):

    ??? "Script to store Adobe/Magento Composer credentials (click me)"
        ```bash
        # Enter your username/password and agree to store your credentials
        ddev_dir="$(ddev version -j | docker run -i --rm ddev/ddev-utilities jq -r ".raw.\"global-ddev-dir\" | select (.!=null) // \"$HOME/.ddev\"" 2>/dev/null)"
        mkdir -p $ddev_dir/homeadditions/.composer
        docker_command=("docker" "run" "-it" "--rm" "-v" "$ddev_dir/homeadditions/.composer:/composer" "--workdir=/tmp" "-e" "COMPOSER_HOME=/composer" "--user" "$(id -u):$(id -g)")
        auth_json_path="$ddev_dir/homeadditions/.composer/auth.json"
        if [ -L "$auth_json_path" ]; then
            # If auth.json is a symlink, add the optional mount
            auth_json_dir=$(dirname "$(readlink -f "$auth_json_path")")
            docker_command+=("-v" "$auth_json_dir:$auth_json_dir")
        fi
        image="$(ddev version -j | docker run -i --rm ddev/ddev-utilities jq -r ".raw.web | select (.!=null)" 2>/dev/null)"
        docker_command+=("$image" "bash" "-c" "composer create --repository https://repo.magento.com/ magento/project-community-edition --no-install")
        # Execute the command to store credentials
        "${docker_command[@]}"
        ```

```bash
export MAGENTO_HOSTNAME=my-magento2-site
mkdir ${MAGENTO_HOSTNAME} && cd ${MAGENTO_HOSTNAME}
ddev config --project-type=magento2 --docroot=pub --upload-dirs=media --disable-settings-management
ddev add-on get ddev/ddev-opensearch
ddev start
ddev composer create-project --repository https://repo.magento.com/ magento/project-community-edition
rm -f app/etc/env.php

ddev magento setup:install --base-url="https://${MAGENTO_HOSTNAME}.ddev.site/" \
    --cleanup-database --db-host=db --db-name=db --db-user=db --db-password=db \
    --opensearch-host=opensearch --search-engine=opensearch --opensearch-port=9200 \
    --admin-firstname=Magento --admin-lastname=User --admin-email=user@example.com \
    --admin-user=admin --admin-password=Password123 --language=en_US

ddev magento deploy:mode:set developer
ddev magento module:disable Magento_TwoFactorAuth Magento_AdminAdobeImsTwoFactorAuth
ddev config --disable-settings-management=false
# Change the backend frontname URL to /admin_ddev
ddev magento setup:config:set --backend-frontname="admin_ddev" --no-interaction
# Login using `admin` user and `Password123` password
ddev launch /admin_ddev
```

Change the admin name and related information as needed.

The admin login URL is specified by `frontName` in `app/etc/env.php`.

You may want to add the [Magento 2 Sample Data](https://experienceleague.adobe.com/docs/commerce-operations/installation-guide/next-steps/sample-data/composer-packages.html) with:

```
ddev magento sampledata:deploy
ddev magento setup:upgrade
```

## Moodle

=== "Composer"

    ```bash
    mkdir my-moodle-site && cd my-moodle-site
    ddev config --composer-root=public --docroot=public --webserver-type=apache-fpm
    ddev start
    ddev composer create-project moodle/moodle
    ddev exec 'php public/admin/cli/install.php --non-interactive --agree-license --wwwroot=$DDEV_PRIMARY_URL --dbtype=mariadb --dbhost=db --dbname=db --dbuser=db --dbpass=db --fullname="DDEV Moodle Demo" --shortname=Demo --adminpass=password'
    ddev launch /login
    ```

    In the web browser, log into your account using `admin` and `password`.

    Visit the [Moodle Admin Quick Guide](https://docs.moodle.org/400/en/Admin_quick_guide) for more information.

    !!!tip
        Moodle relies on a periodic cron job—don’t forget to set that up! See [ddev/ddev-cron](https://github.com/ddev/ddev-cron).

## Node.js

=== "SvelteKit"

    This example installation sets up the SvelteKit demo in DDEV with the `generic` webserver.

    Node.js support as in this example is experimental, and your suggestions and improvements are welcome.

    ```bash
    export SVELTEKIT_SITENAME=my-sveltekit-site
    mkdir ${SVELTEKIT_SITENAME} && cd ${SVELTEKIT_SITENAME}
    ddev config --project-type=generic --webserver-type=generic
    ddev start

    cat <<EOF > .ddev/config.sveltekit.yaml
    web_extra_exposed_ports:
    - name: svelte
      container_port: 3000
      http_port: 80
      https_port: 443
    web_extra_daemons:
    - name: "sveltekit-demo"
      command: "node build"
      directory: /var/www/html
    EOF

    ddev exec "npx sv create --template=demo --types=ts --no-add-ons --no-install ."
    # When it prompts "Directory not empty. Continue?", choose Yes.

    # Install an example svelte.config.js that uses adapter-node
    ddev exec curl -s -OL https://raw.githubusercontent.com/ddev/test-sveltekit/main/svelte.config.js
    # Install an example vite.config.ts that sets the port and allows all hostnames
    ddev exec curl -s -OL https://raw.githubusercontent.com/ddev/test-sveltekit/main/vite.config.ts
    ddev npm install @sveltejs/adapter-node
    ddev npm install
    ddev npm run build
    ddev restart
    ddev launch
    ```

    SvelteKit requires just a bit of configuration to make it run. There are many ways to make any Node.js site work, these are just examples. The `svelte.config.js` and `vite.config.js` used above can be adapted in many ways.

    * `svelte.config.js` example uses `adapter-node`.
    * `vite.config.js` uses port 3000 and `allowedHosts: true`

=== "Node.js Web Server"

    ```bash
    export NODEJS_SITENAME=my-nodejs-site
    mkdir ${NODEJS_SITENAME} && cd ${NODEJS_SITENAME}
    ddev config --project-type=generic --webserver-type=generic
    ddev start
    ddev npm install express

    cat <<EOF > .ddev/config.nodejs.yaml
    web_extra_exposed_ports:
    - name: node-example
      container_port: 3000
      http_port: 80
      https_port: 443

    web_extra_daemons:
    - name: "node-example"
      command: "node server.js"
      directory: /var/www/html
    EOF

    ddev exec curl -s -O https://raw.githubusercontent.com/ddev/test-nodejs/main/server.js
    ddev restart
    ddev launch
    ```

    The [`server.js`](https://github.com/ddev/test-nodejs/blob/main/server.js) used here is a trivial Express-based Node.js webserver. Yours will be more extensive.

## OpenMage

Visit [OpenMage Docs](https://docs.openmage.org) for more installation details.

=== "Composer"

    ```bash
    mkdir my-openmage-site && cd my-openmage-site
    ddev config --project-type=magento --docroot=public_test --php-version=8.1 --web-environment-add=MAGE_IS_DEVELOPER_MODE=1
    ddev start
    ddev composer init --name "openmage/composer-test" --description "OpenMage starter project" --type "project" -l "OSL-3.0" -s "dev" -q
    ddev composer config extra.magento-root-dir "public_test"
    ddev composer config extra.enable-patching true
    ddev composer config extra.magento-core-package-type "magento-source"
    ddev composer config allow-plugins.cweagans/composer-patches true
    ddev composer config allow-plugins.magento-hackathon/magento-composer-installer true
    ddev composer config allow-plugins.aydin-hassan/magento-core-composer-installer true
    ddev composer config allow-plugins.openmage/composer-plugin true
    ddev composer require --no-update "aydin-hassan/magento-core-composer-installer":"^2.1.0" "openmage/magento-lts":"^20.13"
    ddev exec wget -O .ddev/commands/web/openmage-install https://raw.githubusercontent.com/OpenMage/magento-lts/refs/heads/main/.ddev/commands/web/openmage-install
    ddev composer install
    # Silent OpenMage install with sample data
    # See `ddev openmage-install -h` for more options
    ddev openmage-install -q
    # Login using `admin` user and `veryl0ngpassw0rd` password
    ddev launch /admin
    ```

    !!!note "Make sure that `docroot` is set correctly"
        DDEV config `--docroot` has to match Composer config `extra.magento-root-dir`.

=== "Git Clone (for contributors)"

    ```bash
    mkdir my-openmage-site && cd my-openmage-site
    git clone https://github.com/OpenMage/magento-lts .
    ddev config --project-type=magento --php-version=8.1 --web-environment-add=MAGE_IS_DEVELOPER_MODE=1
    ddev start
    ddev composer install
    # Silent OpenMage install with sample data
    # See `ddev openmage-install -h` for more options
    ddev openmage-install -q
    # Login using `admin` user and `veryl0ngpassw0rd` password
    ddev launch /admin
    ```

    Note that OpenMage itself provides several custom DDEV commands, including `openmage-install`, `openmage-admin`, `phpmd`, `rector`, `phpcbf`, `phpstan`, `vendor-patches`, and `php-cs-fixer`.

## Pimcore

=== "Composer"

    Using the [Pimcore skeleton](https://github.com/pimcore/skeleton) repository:

    ``` bash
    mkdir my-pimcore-site && cd my-pimcore-site
    ddev config --project-type=php --docroot=public --webimage-extra-packages='php${DDEV_PHP_VERSION}-amqp'

    ddev start
    ddev composer create-project pimcore/skeleton
    ddev exec pimcore-install \
        --mysql-username=db \
        --mysql-password=db \
        --mysql-host-socket=db \
        --mysql-database=db \
        --admin-password=admin \
        --admin-username=admin
    echo "web_extra_daemons:
      - name: consumer
        command: 'while true; do /var/www/html/bin/console messenger:consume pimcore_core pimcore_maintenance pimcore_scheduled_tasks pimcore_image_optimize pimcore_asset_update --memory-limit=250M --time-limit=3600; done'
        directory: /var/www/html" >.ddev/config.pimcore.yaml

    ddev restart
    ddev launch /admin
    ```

## ProcessWire

To get started with [ProcessWire](https://processwire.com/), create a new directory and use the ZIP file download, composer, or Git checkout to build. These instructions are adapted from [ProcessWire Install Documentation](https://processwire.com/docs/start/install/new/#installing-processwire).

=== "ZIP File"

    ```bash
    mkdir my-processwire-site && cd my-processwire-site
    curl -LJOf https://github.com/processwire/processwire/archive/master.zip
    unzip processwire-master.zip && rm -f processwire-master.zip && mv processwire-master/* . && mv processwire-master/.* . 2>/dev/null && rm -rf processwire-master
    ddev config --project-type=php --webserver-type=apache-fpm
    ddev start
    ddev launch
    ```

=== "Composer"

    ```bash
    mkdir my-processwire-site && cd my-processwire-site
    ddev config --project-type=php --webserver-type=apache-fpm
    ddev start
    ddev composer create-project "processwire/processwire:^3"
    ddev launch
    ```

=== "Git"

    ```bash
    mkdir my-processwire-site && cd my-processwire-site

    # clone the main branch (stable release) into the current directory
    git clone https://github.com/processwire/processwire.git .

    # clone the dev branch (latest features) into the current directory
    # git clone -b dev https://github.com/processwire/processwire.git .

    ddev config --webserver-type=apache-fpm
    ddev start
    ddev launch
    ```

When the installation wizard prompts for database settings, enter:

- `DB Name` = `db`
- `DB User` = `db`
- `DB Pass` = `db`
- `DB Host` = `db` (**not** `localhost`!)
- `DB Charset` = `utf8mb4`
- `DB Engine` = `InnoDB`

If you get a warning about "Apache mod_rewrite" during the compatibility check, Click "refresh".

**After installation,** configure `upload_dirs` to specify where user-generated files are managed by Processwire:

    ```
    ddev config --upload-dirs=sites/assets/files && ddev restart
    ```

If you have any questions there is lots of help in the [DDEV thread in the ProcessWire forum](https://processwire.com/talk/topic/27433-using-ddev-for-local-processwire-development-tips-tricks/).

## Shopware

=== "Composer"

    Though you can set up a Shopware 6 environment many ways, we recommend the following technique. DDEV creates a `.env.local` file for you by default; if you already have one DDEV adds necessary information to it. When `ddev composer create-project` asks if you want to include Docker configuration, answer `x`, as this approach does not use their Docker configuration.

    ```bash
    mkdir my-shopware-site && cd my-shopware-site
    ddev config --project-type=shopware6 --docroot=public
    ddev start
    ddev composer create-project shopware/production
    # If it asks `Do you want to include Docker configuration from recipes?`
    # answer `x`, as we're using DDEV for this rather than its recipes.
    ddev exec console system:install --basic-setup
    ddev launch /admin
    # Default username and password are `admin` and `shopware`
    ```

    Log into the admin site (`/admin`) using the web browser. The default credentials are username `admin` and password `shopware`. You can use the web UI to install sample data or accomplish many other tasks.

    For more advanced tasks like adding elasticsearch, building and watching storefront and administration, see [susi.dev](https://susi.dev/ddev-shopware-6).

## Silverstripe CMS

Use a new or existing Composer project, or clone a Git repository.

=== "Composer"

    ```bash
    mkdir my-silverstripe-site && cd my-silverstripe-site
    ddev config --project-type=silverstripe --docroot=public
    ddev start
    ddev composer create-project --prefer-dist silverstripe/installer
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

Your Silverstripe CMS project is now ready.
The CMS can be found at `/admin`, log into the default admin account using `admin` and `password`.

Visit the Silverstripe CMS [user documentation](https://userhelp.silverstripe.org/) and [developer documentation](https://docs.silverstripe.org/) for more information.

`ddev sake` can be used as a shorthand for the Silverstripe CLI command `ddev exec vendor/bin/sake`.

To open the CMS directly from CLI, run `ddev launch /admin`.

## Statamic

Use a new or existing Composer project, or clone a Git repository.

The Laravel project type can be used for [Statamic](https://statamic.com/) like it can for Laravel. DDEV automatically updates or creates the `.env` file with the database information.

=== "Composer"

    ```bash
    mkdir my-statamic-site && cd my-statamic-site
    ddev config --project-type=laravel --docroot=public
    ddev composer create-project --prefer-dist statamic/statamic
    ddev php please make:user admin@example.com --password=admin1234 --super --no-interaction
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
ddev composer create-project sulu/skeleton
```

Create your default webspace configuration `mv config/webspaces/website.xml config/webspaces/my-sulu-site.xml` and adjust the values for `<name>` and `<key>` so that they are matching your project:

```bash
<?xml version="1.0" encoding="utf-8"?>
<webspace xmlns="http://schemas.sulu.io/webspace/webspace"
          xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
          xsi:schemaLocation="http://schemas.sulu.io/webspace/webspace http://schemas.sulu.io/webspace/webspace-1.1.xsd">
    <!-- See: http://docs.sulu.io/en/latest/book/webspaces.html how to configure your webspace-->

    <name>My Sulu Site</name>
    <key>my-sulu-site</key>
```

Alternatively, use the following commands to adjust the values for `<name>` and `<key>` to match your project setup:

```bash
export SULU_PROJECT_NAME="My Sulu Site"
export SULU_PROJECT_KEY="my-sulu-site"
export SULU_PROJECT_CONFIG_FILE="config/webspaces/my-sulu-site.xml"
ddev exec "mv config/webspaces/website.xml ${SULU_PROJECT_CONFIG_FILE}"
ddev exec "sed -i -e 's|<name>.*</name>|<name>${SULU_PROJECT_NAME}</name>|g' -e 's|<key>.*</key>|<key>${SULU_PROJECT_KEY}</key>|g' ${SULU_PROJECT_CONFIG_FILE}"
```

!!!warning "Caution"
    Changing the `<key>` for a webspace later on causes problems. It is recommended to decide on the value for the key before the database is build in the next step.

Now build the database. Building with the `dev` argument adds the user `admin` with the password `admin` to your project.

```bash
# Set APP_ENV and DATABASE_URL in .env.local
ddev dotenv set .env.local --app-env=dev --database-url="mysql://db:db@db:3306/db?serverVersion=8.0&charset=utf8mb4"
ddev exec bin/adminconsole sulu:build dev --no-interaction
# Login using `admin` user and `admin` password
ddev launch /admin
```

!!!tip
    If you don't want to add an admin user use the `prod` argument instead

    ```bash
    ddev execute bin/adminconsole sulu:build prod
    ```

## Symfony

There are many ways to install Symfony, here are a few of them based on the [Symfony docs](https://symfony.com/doc/current/setup.html).

DDEV automatically updates or creates the `.env.local` file with the database information.

=== "Composer"

    ```bash
    mkdir my-symfony-site && cd my-symfony-site
    ddev config --project-type=symfony --docroot=public
    ddev start
    ddev composer create-project symfony/skeleton
    ddev composer require webapp
    # When it asks if you want to include docker configuration, say "no" with "x"
    ddev launch
    ```

=== "Symfony CLI"

    ```bash
    mkdir my-symfony-site && cd my-symfony-site
    ddev config --project-type=symfony --docroot=public
    ddev start
    ddev exec symfony check:requirements
    ddev exec symfony new temp --version="7.1.*" --webapp
    # 'symfony new' can't install in the current directory right away,
    # so we use 'rsync' to move the installed files one level up
    ddev exec 'rsync -rltgopD temp/ ./ && rm -rf temp'
    ddev launch
    ```

=== "Git Clone"

    ```bash
    git clone <my-symfony-repo> my-symfony-site
    cd my-symfony-site
    ddev config --project-type=symfony --docroot=public
    ddev start
    ddev composer install
    ddev launch
    ```

!!!tip "Want to run Symfony Console (`bin/console`)?"

    ```bash
    ddev console list
    # ddev console doctrine:schema:update --force
    ```

!!!tip "Consuming Messages (Running the Worker)"
    Edit `.ddev/config.yaml` in your project directory and uncomment `post-start` hook to see `messenger:consume` command logs, and run:

    ```bash
    ddev exec symfony server:log
    ```

## TYPO3

=== "Composer"

    ```bash
    PROJECT_NAME=my-typo3-site
    mkdir ${PROJECT_NAME} && cd ${PROJECT_NAME}
    ddev config --project-type=typo3 --docroot=public --php-version=8.3
    ddev start
    ddev composer create-project "typo3/cms-base-distribution"
    ddev exec touch public/FIRST_INSTALL
    ddev launch /typo3/install.php
    ```

=== "Git Clone"

    This example uses a clone of a test repository, `github.com/ddev/test-typo3.git`. 
    Replace that with your git repository.

    ```bash
    PROJECT_GIT_URL=https://github.com/ddev/test-typo3.git
    PROJECT_NAME=my-typo3-site
    mkdir -p ${PROJECT_NAME} && cd ${PROJECT_NAME}
    git clone ${PROJECT_GIT_REPOSITORY} .
    ddev config --project-type=typo3 --docroot=public --php-version=8.3
    ddev start
    ddev composer install
    ddev exec touch public/FIRST_INSTALL
    ddev launch /typo3/install.php
    ```

=== "ddev typo3 setup"

    ```bash
    PROJECT_NAME=my-typo3-site
    mkdir -p ${PROJECT_NAME} && cd ${PROJECT_NAME}
    ddev config --project-type=typo3 --docroot=public --php-version=8.3
    ddev start
    ddev composer create-project typo3/cms-base-distribution 
    ddev exec touch public/FIRST_INSTALL
    
    ddev typo3 setup \
        --admin-user-password="Demo123*" \
        --driver=mysqli \
        --create-site=https://${PROJECT_NAME}.ddev.site \
        --server-type=other \
        --dbname=db \
        --username=db \
        --password=db \
        --port=3306 \
        --host=db \
        --admin-username=admin \
        --admin-email=admin@example.com \
        --project-name="demo TYPO3 site" \
        --force

    ddev launch /typo3/
    # Log in with credentials above, admin/Demo123*
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
    ddev wp core install --url='$DDEV_PRIMARY_URL' --title='My WordPress site' --admin_user=admin --admin_password=admin --admin_email=admin@example.com

    # Launch WordPress admin dashboard in your browser
    ddev launch wp-admin/
    ```

=== "Bedrock"

    [Bedrock](https://roots.io/bedrock/) is a modern, Composer-based installation in WordPress:

    ```bash
    mkdir my-wp-site && cd my-wp-site
    ddev config --project-type=wordpress --docroot=web
    ddev start
    ddev composer create-project roots/bedrock
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

    You can then install the site with WP-CLI and log into the admin interface:

    ```bash
    ddev wp core install --url='$DDEV_PRIMARY_URL' --title='My WordPress site' --admin_user=admin --admin_password=admin --admin_email=admin@example.com
    ddev launch wp-admin/
    ```

    For more details, see [Bedrock installation](https://docs.roots.io/bedrock/master/installation/).

=== "Git Clone"

    To get started using DDEV with an existing WordPress project, clone the project’s repository.

    ```bash
    PROJECT_GIT_URL=https://github.com/ddev/test-wordpress.git
    git clone ${PROJECT_GIT_URL} my-wp-site
    cd my-wp-site
    ddev config --project-type=wordpress
    ddev start
    ddev wp core install --url='$DDEV_PRIMARY_URL' --title='My WordPress site' --admin_user=admin --admin_password=admin --admin_email=admin@example.com
    ddev launch wp-admin/
    ```

    You’ll see a message like:

    > An existing user-managed wp-config.php file has been detected!
    > Project DDEV settings have been written to:
    >
    > /Users/rfay/workspace/bedrock/web/wp-config-ddev.php

    Comment out any database connection settings in your `wp-config.php` file and add the following snippet to your `wp-config.php`, near the bottom of the file and before the include of `wp-settings.php`:

    ```php
    // Include for DDEV-managed settings in wp-config-ddev.php.
    $ddev_settings = __DIR__ . '/wp-config-ddev.php';
    if (is_readable($ddev_settings) && !defined('DB_USER')) {
    require_once($ddev_settings);
    }
    ```

    If you don't care about those settings, or config is managed elsewhere (like in a `.env` file), you can eliminate this message by adding a comment to `wp-config.php`:

    ```php
    // wp-config-ddev.php not needed
    ```

    Now run [`ddev start`](../users/usage/commands.md#start) and continue [Importing a Database](./usage/managing-projects.md#importing-a-database) if you need to.
