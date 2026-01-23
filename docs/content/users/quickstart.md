# CMS Quickstarts

DDEV is [ready to go](./project.md) with generic project types for PHP frameworks, and more specific project types for working with popular platforms and CMSes. To learn more about how to manage projects in DDEV visit [Managing Projects](../users/usage/managing-projects.md).

Before proceeding, make sure your installation of DDEV is up to date. In a new and empty project folder, using your favorite shell, run the following commands:

## Backdrop

You can start a new [Backdrop](https://backdropcms.org) project or configure an existing one.

=== "New projects"

    Create the project directory and configure DDEV:

    ```bash
    mkdir my-backdrop-site && cd my-backdrop-site
    ddev config --project-type=backdrop
    # Add the official Bee CLI add-on
    ddev add-on get backdrop-ops/ddev-backdrop-bee
    ```

    Start DDEV (this may take a minute):

    ```bash
    ddev start
    ```

    Download Backdrop core and create admin user:

    ```bash
    # Download Backdrop core
    ddev bee download-core
    # Create admin user
    ddev bee si --username=admin --password=Password123 --db-name=db --db-user=db --db-pass=db --db-host=db --auto
    ```

    Launch the site:

    ```bash
    # Login using `admin` user and `Password123` password
    ddev launch
    ```

    ??? tip "Prefer to run as a script?"
        To run the whole setup as a script, examine and run this script:

        ```bash
        cat > setup-backdrop.sh << 'EOF'
        #!/usr/bin/env bash
        set -euo pipefail
        mkdir my-backdrop-site && cd my-backdrop-site
        ddev config --project-type=backdrop
        ddev add-on get backdrop-ops/ddev-backdrop-bee
        ddev start -y
        ddev bee download-core
        ddev bee si --username=admin --password=Password123 --db-name=db --db-user=db --db-pass=db --db-host=db --auto
        ddev launch
        EOF
        chmod +x setup-backdrop.sh
        ./setup-backdrop.sh
        ```

=== "Existing projects"

    You can start using DDEV with an existing project, too—but make sure you have a database backup handy!

    Create project directory and clone your repository:

    ```bash
    mkdir my-backdrop-site && cd my-backdrop-site
    git clone https://github.com/ddev/test-backdrop.git .
    ```

    Configure DDEV:

    ```bash
    ddev config --project-type=backdrop
    ```

    Add the official Bee CLI add-on:

    ```bash
    ddev add-on get backdrop-ops/ddev-backdrop-bee
    ```

    Start DDEV (this may take a minute):

    ```bash
    ddev start
    ```

    Import database and files backups:

    ```bash
    ddev import-db --file=/path/to/db.sql.gz
    ddev import-files --source=/path/to/files.tar.gz
    ddev bee cc all
    ```

    Launch the site:

    ```bash
    ddev launch
    ```

    ??? tip "Prefer to run as a script?"
        To run the whole setup as a script, examine and run this script:

        ```bash
        cat > setup-backdrop-existing.sh << 'EOF'
        #!/usr/bin/env bash
        set -euo pipefail
        mkdir my-backdrop-site && cd my-backdrop-site
        git clone https://github.com/ddev/test-backdrop.git .
        ddev config --project-type=backdrop
        ddev add-on get backdrop-ops/ddev-backdrop-bee
        ddev start -y
        ddev import-db --file=/path/to/db.sql.gz
        ddev import-files --source=/path/to/files.tar.gz
        ddev bee cc all
        ddev launch
        EOF
        chmod +x setup-backdrop-existing.sh
        ./setup-backdrop-existing.sh
        ```

## CakePHP

You can start a new [CakePHP](https://cakephp.org) project or configure an existing one.

The CakePHP project type can be used with any CakePHP project >= 3.x, but it has been fully tested with CakePHP 5.x. DDEV automatically creates the `.env` file with the database information, email transport configuration and a random salt. If `.env` file already exists, `.env.ddev` will be created, so you can take any variable and put it into your `.env` file.

Please note that you will need to change the PHP version to 7.4 to be able to work with CakePHP 3.x.

=== "Composer"

    Create the project directory and configure DDEV:

    ```bash
    mkdir my-cakephp-site && cd my-cakephp-site
    ddev config --project-type=cakephp --docroot=webroot
    ```

    Start DDEV (this may take a minute):

    ```bash
    ddev start
    ```

    Install CakePHP via Composer:

    ```bash
    ddev composer create-project --prefer-dist --no-interaction cakephp/app:~5.0
    ```

    Launch the site:

    ```bash
    ddev launch
    ```

    ??? tip "Prefer to run as a script?"
        To run the whole setup as a script, examine and run this script:

        ```bash
        cat > setup-cakephp.sh << 'EOF'
        #!/usr/bin/env bash
        set -euo pipefail
        mkdir my-cakephp-site && cd my-cakephp-site
        ddev config --project-type=cakephp --docroot=webroot
        ddev start -y
        ddev composer create-project --prefer-dist --no-interaction cakephp/app:~5.0
        ddev launch
        EOF
        chmod +x setup-cakephp.sh
        ./setup-cakephp.sh
        ```

=== "Git Clone"

    Clone the repository and configure DDEV:

    ```bash
    git clone <my-cakephp-repo> my-cakephp-site
    cd my-cakephp-site
    ddev config --project-type=cakephp --docroot=webroot
    ```

    Start DDEV (this may take a minute):

    ```bash
    ddev start
    ```

    Install dependencies and launch:

    ```bash
    ddev composer install
    ddev cake
    ddev launch
    ```

    ??? tip "Prefer to run as a script?"
        To run the whole setup as a script, examine and run this script:

        ```bash
        cat > setup-cakephp-git.sh << 'EOF'
        #!/usr/bin/env bash
        set -euo pipefail
        git clone <my-cakephp-repo> my-cakephp-site
        cd my-cakephp-site
        ddev config --project-type=cakephp --docroot=webroot
        ddev start -y
        ddev composer install
        ddev cake
        ddev launch
        EOF
        chmod +x setup-cakephp-git.sh
        ./setup-cakephp-git.sh
        ```

## CiviCRM (Standalone)

[CiviCRM Standalone](https://civicrm.org/blog/ufundo/next-steps-civicrm-standalone) allows running [CiviCRM](https://civicrm.org/) without a CMS. Visit [Install CiviCRM (Standalone)](https://docs.civicrm.org/installation/en/latest/standalone) for more installation details.

Create the project directory and configure DDEV:

```bash
mkdir my-civicrm-site && cd my-civicrm-site
ddev config --project-type=php --composer-root=core --upload-dirs=public/media
```

Start DDEV (this may take a minute):

```bash
ddev start
```

Download and extract CiviCRM:

```bash
ddev exec "curl -LsS https://download.civicrm.org/latest/civicrm-STABLE-standalone.tar.gz -o /tmp/civicrm-standalone.tar.gz"
ddev exec "tar --strip-components=1 -xzf /tmp/civicrm-standalone.tar.gz"
ddev composer require civicrm/cli-tools --no-scripts
```

Install CiviCRM (or use `ddev launch` to install manually):

```bash
# You can install CiviCRM manually in your browser using `ddev launch`
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
```

Launch the site:

```bash
ddev launch
```

??? tip "Prefer to run as a script?"
    To run the whole setup as a script, examine and run this script:

    ```bash
    cat > setup-civicrm.sh << 'EOF'
    #!/usr/bin/env bash
    set -euo pipefail
    mkdir my-civicrm-site && cd my-civicrm-site
    ddev config --project-type=php --composer-root=core --upload-dirs=public/media
    ddev start -y
    ddev exec "curl -LsS https://download.civicrm.org/latest/civicrm-STABLE-standalone.tar.gz -o /tmp/civicrm-standalone.tar.gz"
    ddev exec "tar --strip-components=1 -xzf /tmp/civicrm-standalone.tar.gz"
    ddev composer require civicrm/cli-tools --no-scripts
    ddev exec cv core:install \
        --cms-base-url='$DDEV_PRIMARY_URL' \
        --db=mysql://db:db@db/db \
        -m loadGenerated=1 \
        -m extras.adminUser=admin \
        -m extras.adminPass=admin \
        -m extras.adminEmail=admin@example.com
    ddev launch
    EOF
    chmod +x setup-civicrm.sh
    ./setup-civicrm.sh
    ```

## CodeIgniter

Use a new or existing Composer project, or clone a Git repository.

DDEV automatically updates or creates the `.env` file with the database information.

=== "Composer"

    Create the project directory and configure DDEV:

    ```bash
    mkdir my-ci4-site && cd my-ci4-site
    ddev config --project-type=codeigniter --docroot=public
    ```

    Start DDEV (this may take a minute):

    ```bash
    ddev start
    ```

    Install CodeIgniter via Composer:

    ```bash
    ddev composer create-project codeigniter4/appstarter
    ```

    Launch the site:

    ```bash
    ddev launch
    ```

    ??? tip "Prefer to run as a script?"
        To run the whole setup as a script, examine and run this script:

        ```bash
        cat > setup-codeigniter.sh << 'EOF'
        #!/usr/bin/env bash
        set -euo pipefail
        mkdir my-ci4-site && cd my-ci4-site
        ddev config --project-type=codeigniter --docroot=public
        ddev start -y
        ddev composer create-project codeigniter4/appstarter
        ddev launch
        EOF
        chmod +x setup-codeigniter.sh
        ./setup-codeigniter.sh
        ```

## Contao

Further information on the DDEV procedure can also be found in the [Contao documentation](https://docs.contao.org/manual/en/guides/local-installation/ddev/).

=== "Composer"

    Create the project directory and configure DDEV:

    ```bash
    mkdir my-contao-site && cd my-contao-site
    ddev config --project-type=php --docroot=public --webserver-type=apache-fpm --php-version=8.2
    ```

    Install Contao via Composer (this may take a minute):

    ```bash
    ddev composer create-project contao/managed-edition:5.3
    ```

    Configure database and mailer settings:

    ```bash
    ddev dotenv set .env.local --database-url=mysql://db:db@db:3306/db --mailer-dsn=smtp://localhost:1025
    ```

    Create the database:

    ```bash
    ddev exec contao-console contao:migrate --no-interaction
    ```

    Create backend user:

    ```bash
    ddev exec contao-console contao:user:create --username=admin --name=Administrator --email=admin@example.com --language=en --password=Password123 --admin
    ```

    Launch the administration area:

    ```bash
    ddev launch contao
    ```

    ??? tip "Prefer to run as a script?"
        To run the whole setup as a script, examine and run this script:

        ```bash
        cat > setup-contao.sh << 'EOF'
        #!/usr/bin/env bash
        set -euo pipefail
        mkdir my-contao-site && cd my-contao-site
        ddev config --project-type=php --docroot=public --webserver-type=apache-fpm --php-version=8.2
        ddev composer create-project contao/managed-edition:5.3
        ddev dotenv set .env.local --database-url=mysql://db:db@db:3306/db --mailer-dsn=smtp://localhost:1025
        ddev exec contao-console contao:migrate --no-interaction
        ddev exec contao-console contao:user:create --username=admin --name=Administrator --email=admin@example.com --language=en --password=Password123 --admin
        ddev launch contao
        EOF
        chmod +x setup-contao.sh
        ./setup-contao.sh
        ```

=== "Contao Manager"

    Like most PHP projects, Contao could be installed and updated with Composer. The [Contao Manager](https://docs.contao.org/manual/en/installation/contao-manager/) is a tool that provides a graphical user interface to manage a Contao installation.

    Create the project directory and configure DDEV:

    ```bash
    mkdir my-contao-site && cd my-contao-site
    ddev config --project-type=php --docroot=public --webserver-type=apache-fpm --php-version=8.2
    ```

    Configure database and mailer settings:

    ```bash
    ddev dotenv set .env.local --database-url=mysql://db:db@db:3306/db --mailer-dsn=smtp://localhost:1025
    ```

    Start DDEV (this may take a minute):

    ```bash
    ddev start
    ```

    Download the Contao Manager:

    ```bash
    ddev exec "wget -O public/contao-manager.phar.php https://download.contao.org/contao-manager/stable/contao-manager.phar"
    ```

    Launch the Contao Manager and follow the setup wizard:

    ```bash
    ddev launch contao-manager.phar.php
    ```

    ??? tip "Prefer to run as a script?"
        To run the whole setup as a script, examine and run this script:

        ```bash
        cat > setup-contao-manager.sh << 'EOF'
        #!/usr/bin/env bash
        set -euo pipefail
        mkdir my-contao-site && cd my-contao-site
        ddev config --project-type=php --docroot=public --webserver-type=apache-fpm --php-version=8.2
        ddev dotenv set .env.local --database-url=mysql://db:db@db:3306/db --mailer-dsn=smtp://localhost:1025
        ddev start -y
        ddev exec "wget -O public/contao-manager.phar.php https://download.contao.org/contao-manager/stable/contao-manager.phar"
        ddev launch contao-manager.phar.php
        EOF
        chmod +x setup-contao-manager.sh
        ./setup-contao-manager.sh
        ```

=== "Demo Website"

    The [Contao demo website](https://demo.contao.org/) is maintained for the currently supported Contao versions and can be [optionally installed](https://github.com/contao/contao-demo).
    Via the Contao Manager you can select this option during the first installation.

## Craft CMS

Start a new [Craft CMS](https://craftcms.com) project or retrofit an existing one.

DDEV injects a number of special environment variables into the container (via `.ddev/.env.web`) that [automatically configure](https://craftcms.com/docs/5.x/configure.html#environment-overrides) Craft’s database connection and the project’s primary site URL. You may opt out of this behavior with the [`disable_settings_management`](./configuration/config.md#disable_settings_management) setting.

!!!tip "Compatibility with Craft CMS 3"
    The `craftcms` project works best with configuration features that became available in Craft CMS 4.x. If you are using Craft CMS 3.x or earlier, you may want to use the `php` project type and explicitly define [database connection details](./usage/database-management.md#database-backends-and-defaults) via [Craft’s `db.php`](https://craftcms.com/docs/3.x/config/db-settings.html).

=== "New projects"

    Create the project directory and configure DDEV:

    ```bash
    mkdir my-craft-site && cd my-craft-site
    ddev config --project-type=craftcms --docroot=web
    ```

    Start DDEV (this may take a minute):

    ```bash
    ddev start
    ```

    Scaffold a new project with Composer:

    ```bash
    ddev composer create-project craftcms/craft
    ```

    Craft's setup wizard will start automatically!

    [Third-party starter projects](https://craftcms.com/knowledge-base/using-the-starter-project#community-starter-projects) can be substituted for `craftcms/craft` when running `ddev composer create-project`.

    ??? tip "Prefer to run as a script?"
        To run the whole setup as a script, examine and run this script:

        ```bash
        cat > setup-craft.sh << 'EOF'
        #!/usr/bin/env bash
        set -euo pipefail
        mkdir my-craft-site && cd my-craft-site
        ddev config --project-type=craftcms --docroot=web
        ddev start -y
        ddev composer create-project craftcms/craft
        EOF
        chmod +x setup-craft.sh
        ./setup-craft.sh
        ```

=== "Existing projects"

    You can start using DDEV with an existing Craft project, too. All you need is the codebase and a database backup!

    Clone the repository (or navigate to a local project directory):

    ```bash
    git clone https://github.com/example/example-site my-craft-site
    cd my-craft-site
    ```

    Configure DDEV:

    ```bash
    ddev config --project-type=craftcms --docroot=web
    ```

    Start DDEV and install Composer packages:

    ```bash
    ddev start
    ddev composer install
    ```

    Import database backup and launch:

    ```bash
    ddev craft db/restore /path/to/db.sql.gz
    ddev launch
    ```

    Craft CMS projects use MySQL 8.0, by default. You can override this setting (and the PHP version) during setup with [`config` command flags](./usage/commands.md#config) or after setup via the [configuration files](./configuration/config.md).

    ??? tip "Prefer to run as a script?"
        To run the whole setup as a script, examine and run this script:

        ```bash
        cat > setup-craft-existing.sh << 'EOF'
        #!/usr/bin/env bash
        set -euo pipefail
        git clone https://github.com/example/example-site my-craft-site
        cd my-craft-site
        ddev config --project-type=craftcms --docroot=web
        ddev start -y
        ddev composer install
        ddev craft db/restore /path/to/db.sql.gz
        ddev launch
        EOF
        chmod +x setup-craft-existing.sh
        ./setup-craft-existing.sh
        ```

### Running Craft in a Subdirectory

Set [`composer_root`](./configuration/config.md#composer_root) to the subdirectory where Craft is installed. For example, `ddev config --composer-root=app`.

!!!tip "Installing Craft"
    Read more about installing Craft in the [official documentation](https://craftcms.com/docs).

## Drupal

=== "Drupal 11"

    Create the project directory and configure DDEV:

    ```bash
    mkdir my-drupal-site && cd my-drupal-site
    ddev config --project-type=drupal11 --docroot=web
    ```

    Start DDEV (this may take a minute):

    ```bash
    ddev start
    ```

    Install Drupal via Composer:

    ```bash
    ddev composer create-project "drupal/recommended-project:^11"
    ddev composer require drush/drush
    ```

    Run Drupal installation and launch:

    ```bash
    ddev drush site:install --account-name=admin --account-pass=admin -y
    ddev launch
    # or automatically log in with:
    ddev launch $(ddev drush uli)
    ```

    Read more about: [Drupal Core](https://new.drupal.org/about/overview/technical) & [Documentation](https://www.drupal.org/docs)

    ??? tip "Prefer to run as a script?"
        To run the whole setup as a script, examine and run this script:

        ```bash
        cat > setup-drupal11.sh << 'EOF'
        #!/usr/bin/env bash
        set -euo pipefail
        mkdir my-drupal-site && cd my-drupal-site
        ddev config --project-type=drupal11 --docroot=web
        ddev start -y
        ddev composer create-project "drupal/recommended-project:^11"
        ddev composer require drush/drush
        ddev drush site:install --account-name=admin --account-pass=admin -y
        ddev launch
        EOF
        chmod +x setup-drupal11.sh
        ./setup-drupal11.sh
        ```

=== "Drupal CMS"

    Create the project directory and configure DDEV:

    ```bash
    mkdir my-drupal-site && cd my-drupal-site
    ddev config --project-type=drupal11 --docroot=web
    ```

    Start DDEV (this may take a minute):

    ```bash
    ddev start
    ```

    Install Drupal CMS via Composer:

    ```bash
    ddev composer create-project drupal/cms
    ```

    Launch the site:

    ```bash
    ddev launch
    ```

    Read more about: [Drupal CMS](https://new.drupal.org/drupal-cms) & [Documentation](https://new.drupal.org/docs/drupal-cms)

    ??? tip "Prefer to run as a script?"
        To run the whole setup as a script, examine and run this script:

        ```bash
        cat > setup-drupal-cms.sh << 'EOF'
        #!/usr/bin/env bash
        set -euo pipefail
        mkdir my-drupal-site && cd my-drupal-site
        ddev config --project-type=drupal11 --docroot=web
        ddev start -y
        ddev composer create-project drupal/cms
        ddev launch
        EOF
        chmod +x setup-drupal-cms.sh
        ./setup-drupal-cms.sh
        ```

=== "Drupal 10"

    Create the project directory and configure DDEV:

    ```bash
    mkdir my-drupal-site && cd my-drupal-site
    ddev config --project-type=drupal10 --docroot=web
    ```

    Start DDEV (this may take a minute):

    ```bash
    ddev start
    ```

    Install Drupal via Composer:

    ```bash
    ddev composer create-project "drupal/recommended-project:^10"
    ddev composer require drush/drush
    ```

    Run Drupal installation and launch:

    ```bash
    ddev drush site:install --account-name=admin --account-pass=admin -y
    ddev launch
    # or automatically log in with:
    ddev launch $(ddev drush uli)
    ```

    Read more about: [Drupal Core](https://new.drupal.org/about/overview/technical) & [Documentation](https://www.drupal.org/docs)

    ??? tip "Prefer to run as a script?"
        To run the whole setup as a script, examine and run this script:

        ```bash
        cat > setup-drupal10.sh << 'EOF'
        #!/usr/bin/env bash
        set -euo pipefail
        mkdir my-drupal-site && cd my-drupal-site
        ddev config --project-type=drupal10 --docroot=web
        ddev start -y
        ddev composer create-project "drupal/recommended-project:^10"
        ddev composer require drush/drush
        ddev drush site:install --account-name=admin --account-pass=admin -y
        ddev launch
        EOF
        chmod +x setup-drupal10.sh
        ./setup-drupal10.sh
        ```

=== "Drupal 6/7"

    Clone your Drupal repository:

    ```bash
    git clone https://github.com/example/my-drupal-site
    cd my-drupal-site
    ```

    Configure DDEV (follow prompts):

    ```bash
    ddev config
    ```

    Start DDEV and launch:

    ```bash
    ddev start
    ddev launch /install.php
    ```

    Drupal 7 doesn't know how to redirect from the front page to `/install.php` if the database is not set up but the settings files *are* set up, so launching with `/install.php` gets you started with an installation. You can also run `drush site-install`, then `ddev exec drush site-install --yes`.

    See [Importing a Database](./usage/managing-projects.md#importing-a-database).

    ??? tip "Prefer to run as a script?"
        To run the whole setup as a script, examine and run this script:

        ```bash
        cat > setup-drupal67.sh << 'EOF'
        #!/usr/bin/env bash
        set -euo pipefail
        git clone https://github.com/example/my-drupal-site
        cd my-drupal-site
        ddev config
        ddev start -y
        ddev launch /install.php
        EOF
        chmod +x setup-drupal67.sh
        ./setup-drupal67.sh
        ```

=== "Git Clone"

    Clone your Drupal repository:

    ```bash
    PROJECT_GIT_URL=https://github.com/ddev/test-drupal11.git
    git clone ${PROJECT_GIT_URL} my-drupal-site
    cd my-drupal-site
    ```

    Configure and start DDEV:

    ```bash
    ddev config --project-type=drupal11 --docroot=web
    ddev start
    ```

    Install dependencies and set up Drupal:

    ```bash
    ddev composer install
    ddev drush site:install --account-name=admin --account-pass=admin -y
    ```

    Launch the site:

    ```bash
    ddev launch
    ```

    Read more about: [Drupal Core](https://new.drupal.org/about/overview/technical) & [Documentation](https://www.drupal.org/docs)

    ??? tip "Prefer to run as a script?"
        To run the whole setup as a script, examine and run this script:

        ```bash
        cat > setup-drupal-git.sh << 'EOF'
        #!/usr/bin/env bash
        set -euo pipefail
        PROJECT_GIT_URL=https://github.com/ddev/test-drupal11.git
        git clone ${PROJECT_GIT_URL} my-drupal-site
        cd my-drupal-site
        ddev config --project-type=drupal11 --docroot=web
        ddev start -y
        ddev composer install
        ddev drush site:install --account-name=admin --account-pass=admin -y
        ddev launch
        EOF
        chmod +x setup-drupal-git.sh
        ./setup-drupal-git.sh
        ```

## ExpressionEngine

=== "ExpressionEngine ZIP File Download"

    Create the project directory:

    ```bash
    mkdir my-ee-site && cd my-ee-site
    ```

    Download and extract the latest ExpressionEngine release:

    ```bash
    DOWNLOAD_URL=$(curl -sL https://api.github.com/repos/ExpressionEngine/ExpressionEngine/releases/latest | docker run -i --rm ddev/ddev-utilities jq -r '.assets | map(select(.name | test("^ExpressionEngine.*\\.zip$")))[0].browser_download_url')
    curl -o ee.zip -L "${DOWNLOAD_URL}"
    unzip ee.zip && rm -f ee.zip
    ```

    Configure and start DDEV:

    ```bash
    ddev config --database=mysql:8.0
    ddev start
    ```

    Launch the installation wizard:

    ```bash
    ddev launch /admin.php
    ```

    When the installation wizard prompts for database settings, enter `db` for the _DB Server Address_, _DB Name_, _DB Username_, and _DB Password_.

    Visit your site.

    ??? tip "Prefer to run as a script?"
        To run the whole setup as a script, examine and run this script:

        ```bash
        cat > setup-ee.sh << 'EOF'
        #!/usr/bin/env bash
        set -euo pipefail
        mkdir my-ee-site && cd my-ee-site
        DOWNLOAD_URL=$(curl -sL https://api.github.com/repos/ExpressionEngine/ExpressionEngine/releases/latest | docker run -i --rm ddev/ddev-utilities jq -r '.assets | map(select(.name | test("^ExpressionEngine.*\\.zip$")))[0].browser_download_url')
        curl -o ee.zip -L "${DOWNLOAD_URL}"
        unzip ee.zip && rm -f ee.zip
        ddev config --database=mysql:8.0
        ddev start -y
        ddev launch /admin.php
        EOF
        chmod +x setup-ee.sh
        ./setup-ee.sh
        ```

=== "ExpressionEngine Git Checkout"

    Follow these steps based on the [ExpressionEngine Git Repository README.md](https://github.com/ExpressionEngine/ExpressionEngine#how-to-install):

    Create the project directory and clone the repository:

    ```bash
    mkdir my-ee-site && cd my-ee-site
    git clone https://github.com/ExpressionEngine/ExpressionEngine .
    ```

    Configure and start DDEV:

    ```bash
    ddev config --database=mysql:8.0
    ddev start
    ```

    Install Composer dependencies and prepare installation:

    ```bash
    ddev composer install
    touch system/user/config/config.php
    echo "EE_INSTALL_MODE=TRUE" >.env.php
    ```

    Launch the installation wizard:

    ```bash
    ddev launch /admin.php
    ```

    When the installation wizard prompts for database settings, enter `db` for the _DB Server Address_, _DB Name_, _DB Username_, and _DB Password_.

    ??? tip "Prefer to run as a script?"
        To run the whole setup as a script, examine and run this script:

        ```bash
        cat > setup-ee-git.sh << 'EOF'
        #!/usr/bin/env bash
        set -euo pipefail
        mkdir my-ee-site && cd my-ee-site
        git clone https://github.com/ExpressionEngine/ExpressionEngine .
        ddev config --database=mysql:8.0
        ddev start -y
        ddev composer install
        touch system/user/config/config.php
        echo "EE_INSTALL_MODE=TRUE" >.env.php
        ddev launch /admin.php
        EOF
        chmod +x setup-ee-git.sh
        ./setup-ee-git.sh
        ```

## Generic

The [`webserver_type: generic`](./configuration/config.md#webserver_type) allows you to define your own web server process(es) and exposed ports for projects that don't use the standard `nginx-fpm` or `apache-fpm` configurations.

!!!tip "Looking for more advanced generic web server examples?"
    Check out the [Node.js](#nodejs) and [Wagtail](#wagtail-python-generic) examples below.

    See also the [ddev-frankenphp](https://github.com/ddev/ddev-frankenphp) add-on, which uses the `generic` webserver under the hood.

=== "PHP's built-in web server"

    This trivial example demonstrates running PHP's built-in web server inside DDEV's web container. The `ddev-webserver` container will not start the default `nginx` or `php-fpm` daemons—the PHP built-in server will handle all requests. You probably wouldn't find this useful compared to the normal `nginx-fpm` or `apache-fpm` configurations, but it's offered here as an example of how the `generic` webserver type works.

    Create the project directory and configure DDEV:

    ```bash
    export GENERIC_SITENAME=my-generic-site
    mkdir ${GENERIC_SITENAME} && cd ${GENERIC_SITENAME}
    ddev config --project-type=php
    ```

    Create a sample PHP info page:

    ```bash
    echo "<?php phpinfo(); ?>" > index.php
    ```

    Configure the web server to run PHP's built-in server:

    ```bash
    cat <<'EOF' > .ddev/config.php-server.yaml
    webserver_type: generic
    web_extra_daemons:
        - name: "php-server"
          command: "php -S 0.0.0.0:8000 -t \"${DDEV_DOCROOT:-.}\""
          directory: /var/www/html
    web_extra_exposed_ports:
        - name: "php-server"
          container_port: 8000
          http_port: 80
          https_port: 443
    EOF
    ```

    Start DDEV (this may take a minute):

    ```bash
    ddev start
    ```

    Launch the site:

    ```bash
    ddev launch
    ```

    ??? tip "Prefer to run as a script?"
        To run the whole setup as a script, examine and run this script:

        ```bash
        cat > setup-generic.sh << 'EOF'
        #!/usr/bin/env bash
        set -euo pipefail
        export GENERIC_SITENAME=my-generic-site
        mkdir ${GENERIC_SITENAME} && cd ${GENERIC_SITENAME}
        ddev config --project-type=php
        echo "<?php phpinfo(); ?>" > index.php
        cat <<'INNEREOF' > .ddev/config.php-server.yaml
        webserver_type: generic
        web_extra_daemons:
            - name: "php-server"
              command: "php -S 0.0.0.0:8000 -t \"${DDEV_DOCROOT:-.}\""
              directory: /var/www/html
        web_extra_exposed_ports:
            - name: "php-server"
              container_port: 8000
              http_port: 80
              https_port: 443
        INNEREOF
        ddev start -y
        ddev launch
        EOF
        chmod +x setup-generic.sh
        ./setup-generic.sh
        ```

## Grav

=== "Composer"

    Create the project directory and configure DDEV:

    ```bash
    mkdir my-grav-site && cd my-grav-site
    ddev config --php-version=8.3 --omit-containers=db
    ```

    Start DDEV (this may take a minute):

    ```bash
    ddev start
    ```

    Install Grav via Composer:

    ```bash
    ddev composer create-project getgrav/grav
    ```

    Install the admin plugin and launch:

    ```bash
    ddev exec gpm install admin -y
    ddev launch
    ```

    ??? tip "Prefer to run as a script?"
        To run the whole setup as a script, examine and run this script:

        ```bash
        cat > setup-grav.sh << 'EOF'
        #!/usr/bin/env bash
        set -euo pipefail
        mkdir my-grav-site && cd my-grav-site
        ddev config --php-version=8.3 --omit-containers=db
        ddev start -y
        ddev composer create-project getgrav/grav
        ddev exec gpm install admin -y
        ddev launch
        EOF
        chmod +x setup-grav.sh
        ./setup-grav.sh
        ```

=== "Git Clone"

    Create the project directory and clone Grav:

    ```bash
    mkdir my-grav-site && cd my-grav-site
    git clone -b master https://github.com/getgrav/grav.git .
    ```

    Configure DDEV:

    ```bash
    ddev config --php-version=8.3 --omit-containers=db
    ```

    Start DDEV (this may take a minute):

    ```bash
    ddev start
    ```

    Install dependencies and initialize Grav:

    ```bash
    ddev composer install
    ddev exec grav install
    ```

    Install the admin plugin and launch:

    ```bash
    ddev exec gpm install admin -y
    ddev launch
    ```

    ??? tip "Prefer to run as a script?"
        To run the whole setup as a script, examine and run this script:

        ```bash
        cat > setup-grav-git.sh << 'EOF'
        #!/usr/bin/env bash
        set -euo pipefail
        mkdir my-grav-site && cd my-grav-site
        git clone -b master https://github.com/getgrav/grav.git .
        ddev config --php-version=8.3 --omit-containers=db
        ddev start -y
        ddev composer install
        ddev exec grav install
        ddev exec gpm install admin -y
        ddev launch
        EOF
        chmod +x setup-grav-git.sh
        ./setup-grav-git.sh
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

Create the project directory and configure DDEV:

```bash
mkdir my-ibexa-site && cd my-ibexa-site
ddev config --project-type=php --docroot=public --web-environment-add DATABASE_URL=mysql://db:db@db:3306/db
```

Start DDEV (this may take a minute):

```bash
ddev start
```

Install Ibexa via Composer:

```bash
ddev composer create-project ibexa/oss-skeleton
```

Run Ibexa installation:

```bash
ddev exec console ibexa:install --no-interaction
```

Launch the admin interface:

```bash
ddev launch /admin/login
```

In the web browser, log into your account using `admin` and `publish`.

Visit [Ibexa documentation](https://doc.ibexa.co/en/latest/getting_started/install_with_ddev/) for more cases.

??? tip "Prefer to run as a script?"
    To run the whole setup as a script, examine and run this script:

    ```bash
    cat > setup-ibexa.sh << 'EOF'
    #!/usr/bin/env bash
    set -euo pipefail
    mkdir my-ibexa-site && cd my-ibexa-site
    ddev config --project-type=php --docroot=public --web-environment-add DATABASE_URL=mysql://db:db@db:3306/db
    ddev start -y
    ddev composer create-project ibexa/oss-skeleton
    ddev exec console ibexa:install
    ddev exec console ibexa:graphql:generate-schema
    ddev launch /admin/login
    EOF
    chmod +x setup-ibexa.sh
    ./setup-ibexa.sh
    ```

## Joomla

Create the project directory and download Joomla:

```bash
mkdir my-joomla-site && cd my-joomla-site
# Download the latest version of Joomla! and unzip it.
# This can be manually downloaded from https://downloads.joomla.org/ or done using curl as here.
DOWNLOAD_URL=$(curl -sL https://api.github.com/repos/joomla/joomla-cms/releases/latest | docker run -i --rm ddev/ddev-utilities jq -r '.assets | map(select(.name | test("^Joomla.*Stable-Full_Package\\.zip$")))[0].browser_download_url')
curl -o joomla.zip -L "${DOWNLOAD_URL}"
unzip joomla.zip && rm -f joomla.zip
```

Configure DDEV:

```bash
ddev config --project-type=php --webserver-type=apache-fpm --upload-dirs=images
```

Start DDEV (this may take a minute):

```bash
ddev start
```

Install Joomla and launch:

```bash
ddev php installation/joomla.php install --site-name="My Joomla Site" --admin-user="Administrator" --admin-username=admin --admin-password=AdminAdmin1! --admin-email=admin@example.com --db-type=mysql --db-encryption=0 --db-host=db --db-user=db --db-pass="db" --db-name=db --db-prefix=ddev_ --public-folder=""
ddev launch /administrator
```

??? tip "Prefer to run as a script?"
    To run the whole setup as a script, examine and run this script:

    ```bash
    cat > setup-joomla.sh << 'EOF'
    #!/usr/bin/env bash
    set -euo pipefail
    mkdir my-joomla-site && cd my-joomla-site
    DOWNLOAD_URL=$(curl -sL https://api.github.com/repos/joomla/joomla-cms/releases/latest | docker run -i --rm ddev/ddev-utilities jq -r '.assets | map(select(.name | test("^Joomla.*Stable-Full_Package\\.zip$")))[0].browser_download_url')
    curl -o joomla.zip -L "${DOWNLOAD_URL}"
    unzip joomla.zip && rm -f joomla.zip
    ddev config --project-type=php --webserver-type=apache-fpm --upload-dirs=images
    ddev start -y
    ddev php installation/joomla.php install --site-name="My Joomla Site" --admin-user="Administrator" --admin-username=admin --admin-password=AdminAdmin1! --admin-email=admin@example.com --db-type=mysql --db-encryption=0 --db-host=db --db-user=db --db-pass="db" --db-name=db --db-prefix=ddev_ --public-folder=""
    ddev launch /administrator
    EOF
    chmod +x setup-joomla.sh
    ./setup-joomla.sh
    ```

## Kirby CMS

Start a new [Kirby CMS](https://getkirby.com) project or use an existing one.

=== "New projects"

    Create a new Kirby CMS project from the official [Starterkit](https://github.com/getkirby/starterkit) using DDEV's [`composer create-project` command](../users/usage/commands.md#composer):

    Create the project directory and configure DDEV:

    ```bash
    mkdir my-kirby-site && cd my-kirby-site
    ddev config --omit-containers=db --webserver-type=apache-fpm
    ```

    Start DDEV (this may take a minute):

    ```bash
    ddev start
    ```

    Install the Kirby Starterkit:

    ```bash
    ddev composer create-project getkirby/starterkit
    ```

    Launch the site:

    ```bash
    ddev launch
    ```

    ??? tip "Prefer to run as a script?"
        To run the whole setup as a script, examine and run this script:

        ```bash
        cat > setup-kirby.sh << 'EOF'
        #!/usr/bin/env bash
        set -euo pipefail
        mkdir my-kirby-site && cd my-kirby-site
        ddev config --omit-containers=db --webserver-type=apache-fpm
        ddev start -y
        ddev composer create-project getkirby/starterkit
        ddev launch
        EOF
        chmod +x setup-kirby.sh
        ./setup-kirby.sh
        ```

=== "Existing projects"

    You can start using DDEV with an existing project as well:

    Navigate to your existing project directory:

    ```bash
    cd my-kirby-site
    ```

    Configure DDEV:

    ```bash
    ddev config --omit-containers=db --webserver-type=apache-fpm
    ```

    Start DDEV (this may take a minute):

    ```bash
    ddev start
    ```

    Launch the site:

    ```bash
    ddev launch
    ```

    ??? tip "Prefer to run as a script?"
        To run the whole setup as a script, examine and run this script:

        ```bash
        cat > setup-kirby-existing.sh << 'EOF'
        #!/usr/bin/env bash
        set -euo pipefail
        cd my-kirby-site
        ddev config --omit-containers=db --webserver-type=apache-fpm
        ddev start -y
        ddev launch
        EOF
        chmod +x setup-kirby-existing.sh
        ./setup-kirby-existing.sh
        ```

!!!tip "Installing Kirby"
    Read more about developing your Kirby project with DDEV in our [extensive DDEV guide](https://getkirby.com/docs/cookbook/setup/ddev).

## Laravel

Use a new or existing Composer project, or clone a Git repository.

The Laravel project type can be used for [StarterKits](https://laravel.com/docs/starter-kits), [Laravel Livewire](https://livewire.laravel.com/) and others, as it is used with basic Laravel. DDEV automatically updates or creates the `.env` file with the database information.

=== "Composer"

    Laravel defaults to SQLite, but we use MariaDB to better mimic a production environment:

    Create the project directory and configure DDEV:

    ```bash
    mkdir my-laravel-site && cd my-laravel-site
    ddev config --project-type=laravel --docroot=public
    ```

    Start DDEV (this may take a minute):

    ```bash
    ddev start
    ```

    Install Laravel via Composer:

    ```bash
    ddev composer create-project "laravel/laravel:^12"
    ```

    Launch the site:

    ```bash
    ddev launch
    ```

    ??? tip "Prefer to run as a script?"
        To run the whole setup as a script, examine and run this script:

        ```bash
        cat > setup-laravel.sh << 'EOF'
        #!/usr/bin/env bash
        set -euo pipefail
        mkdir my-laravel-site && cd my-laravel-site
        ddev config --project-type=laravel --docroot=public
        ddev start -y
        ddev composer create-project "laravel/laravel:^12"
        ddev launch
        EOF
        chmod +x setup-laravel.sh
        ./setup-laravel.sh
        ```

=== "Composer (SQLite)"

    To use the SQLite configuration provided by Laravel:

    Create the project directory and configure DDEV:

    ```bash
    mkdir my-laravel-site && cd my-laravel-site
    ddev config --project-type=laravel --docroot=public --omit-containers=db --disable-settings-management=true
    ```

    Start DDEV (this may take a minute):

    ```bash
    ddev start
    ```

    Install Laravel via Composer:

    ```bash
    ddev composer create-project "laravel/laravel:^12"
    ```

    Launch the site:

    ```bash
    ddev launch
    ```

    ??? tip "Prefer to run as a script?"
        To run the whole setup as a script, examine and run this script:

        ```bash
        cat > setup-laravel-sqlite.sh << 'EOF'
        #!/usr/bin/env bash
        set -euo pipefail
        mkdir my-laravel-site && cd my-laravel-site
        ddev config --project-type=laravel --docroot=public --omit-containers=db --disable-settings-management=true
        ddev start -y
        ddev composer create-project "laravel/laravel:^12"
        ddev launch
        EOF
        chmod +x setup-laravel-sqlite.sh
        ./setup-laravel-sqlite.sh
        ```

    To switch an existing Laravel project to SQLite:

    Configure for SQLite and restart:

    ```bash
    ddev config --project-type=laravel --docroot=public --omit-containers=db --disable-settings-management=true
    ddev restart
    ```

    Run post-install scripts:

    ```bash
    ddev composer run-script post-root-package-install
    ddev dotenv set .env --db-connection=sqlite
    ddev composer run-script post-create-project-cmd
    ```

    Launch the site:

    ```bash
    ddev launch
    ```

    ??? tip "Prefer to run as a script?"
        To run the whole setup as a script, examine and run this script:

        ```bash
        cat > switch-laravel-sqlite.sh << 'EOF'
        #!/usr/bin/env bash
        set -euo pipefail
        ddev config --project-type=laravel --docroot=public --omit-containers=db --disable-settings-management=true
        ddev restart
        ddev composer run-script post-root-package-install
        ddev dotenv set .env --db-connection=sqlite
        ddev composer run-script post-create-project-cmd
        ddev launch
        EOF
        chmod +x switch-laravel-sqlite.sh
        ./switch-laravel-sqlite.sh
        ```

=== "Laravel Installer"

    Create the project directory and configure DDEV:

    ```bash
    mkdir my-laravel-site && cd my-laravel-site
    ddev config --project-type=laravel --docroot=public
    # For SQLite instead, use:
    # ddev config --project-type=laravel --docroot=public --omit-containers=db --disable-settings-management=true
    ```

    Create Dockerfile to add Laravel installer:

    ```bash
    cat <<'DOCKERFILEEND' >.ddev/web-build/Dockerfile.laravel
    ARG COMPOSER_HOME=/usr/local/composer
    RUN composer global require laravel/installer
    RUN ln -s $COMPOSER_HOME/vendor/bin/laravel /usr/local/bin/laravel
    DOCKERFILEEND
    ```

    Start DDEV (this may take a minute):

    ```bash
    ddev start
    ```

    Run Laravel installer (follow prompts and select starter kit):

    ```bash
    ddev exec laravel new temp --database=sqlite
    # SQLite is used here as other database types would fail due to
    # the .env file not being ready, which DDEV will fix on 'ddev restart'
    ```

    Move files and clean up:

    ```bash
    ddev exec 'rsync -rltgopD temp/ ./ && rm -rf temp'
    rm -f .ddev/web-build/Dockerfile.laravel .env
    ```

    Restart and finalize:

    ```bash
    ddev restart
    ddev composer run-script post-root-package-install
    ddev composer run-script post-create-project-cmd
    ```

    Launch the site:

    ```bash
    ddev launch
    ```

    ??? tip "Prefer to run as a script?"
        To run the whole setup as a script, examine and run this script:

        ```bash
        cat > setup-laravel-installer.sh << 'EOF'
        #!/usr/bin/env bash
        set -euo pipefail
        mkdir my-laravel-site && cd my-laravel-site
        ddev config --project-type=laravel --docroot=public

        cat <<'INNEREOF' >.ddev/web-build/Dockerfile.laravel
        ARG COMPOSER_HOME=/usr/local/composer
        RUN composer global require laravel/installer
        RUN ln -s $COMPOSER_HOME/vendor/bin/laravel /usr/local/bin/laravel
        INNEREOF

        ddev start -y
        ddev exec laravel new temp --database=sqlite
        ddev exec 'rsync -rltgopD temp/ ./ && rm -rf temp'
        rm -f .ddev/web-build/Dockerfile.laravel .env
        ddev restart
        ddev composer run-script post-root-package-install
        ddev composer run-script post-create-project-cmd
        ddev launch
        EOF
        chmod +x setup-laravel-installer.sh
        ./setup-laravel-installer.sh
        ```

=== "Git Clone"

    Clone your Laravel repository:

    ```bash
    git clone <my-laravel-repo> my-laravel-site
    cd my-laravel-site
    ```

    Configure and start DDEV:

    ```bash
    ddev config --project-type=laravel --docroot=public
    ddev start
    ```

    Install dependencies and run post-install scripts:

    ```bash
    ddev composer install
    ddev composer run-script post-root-package-install
    ddev composer run-script post-create-project-cmd
    ```

    Launch the site:

    ```bash
    ddev launch
    ```

    ??? tip "Prefer to run as a script?"
        To run the whole setup as a script, examine and run this script:

        ```bash
        cat > setup-laravel-git.sh << 'EOF'
        #!/usr/bin/env bash
        set -euo pipefail
        git clone <my-laravel-repo> my-laravel-site
        cd my-laravel-site
        ddev config --project-type=laravel --docroot=public
        ddev start -y
        ddev composer install
        ddev composer run-script post-root-package-install
        ddev composer run-script post-create-project-cmd
        ddev launch
        EOF
        chmod +x setup-laravel-git.sh
        ./setup-laravel-git.sh
        ```

!!!tip "Add Vite support?"
    Since Laravel v9.19, Vite is included as the default [asset bundler](https://laravel.com/docs/vite). See the [Vite Integration](usage/vite.md#laravel) documentation for complete setup instructions.

## Magento 2

Normal details of a Composer build for Magento 2 are on the [Magento 2 site](https://experienceleague.adobe.com/docs/commerce-operations/installation-guide/composer.html). You must have a public and private key to install from Magento’s repository. When prompted for “username” and “password” in `composer create-project`, it’s asking for your public key as "username" and private key as "password".

!!!tip "Store Adobe/Magento Composer credentials in the global DDEV config"
    If you have Composer installed on your workstation and have an `auth.json` you can reuse the `auth.json` by making a symlink. See [In-Container Home Directory and Shell Configuration](extend/in-container-configuration.md):

    ```bash
    mkdir -p $HOME/.ddev/homeadditions/.composer && ln -s ~/.composer/auth.json $HOME/.ddev/homeadditions/.composer/auth.json
    ```

    Alternately, you can install the Adobe/Magento Composer credentials in your global `$HOME/.ddev/homeadditions/.composer/auth.json` and never have to enter them again (see below):

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
        docker_command+=("$image" "bash" "-c" "composer create-project --repository https://repo.magento.com/ magento/project-community-edition --no-install")
        # Execute the command to store credentials
        "${docker_command[@]}"
        ```

Create the project directory and configure DDEV:

```bash
export MAGENTO_HOSTNAME=my-magento2-site
mkdir ${MAGENTO_HOSTNAME} && cd ${MAGENTO_HOSTNAME}
ddev config --project-type=magento2 --docroot=pub --upload-dirs=media --disable-settings-management
ddev add-on get ddev/ddev-opensearch
```

Start DDEV (this may take a minute):

```bash
ddev start
```

Install Magento via Composer:

```bash
ddev composer create-project --repository https://repo.magento.com/ magento/project-community-edition
rm -f app/etc/env.php
```

Run Magento setup:

```bash
ddev magento setup:install --base-url="https://${MAGENTO_HOSTNAME}.ddev.site/" \
    --cleanup-database --db-host=db --db-name=db --db-user=db --db-password=db \
    --opensearch-host=opensearch --search-engine=opensearch --opensearch-port=9200 \
    --admin-firstname=Magento --admin-lastname=User --admin-email=user@example.com \
    --admin-user=admin --admin-password=Password123 --language=en_US
```

Configure Magento and launch:

```bash
ddev magento deploy:mode:set developer
ddev magento module:disable Magento_TwoFactorAuth Magento_AdminAdobeImsTwoFactorAuth
ddev config --disable-settings-management=false
# Change the backend frontname URL to /admin_ddev
ddev magento setup:config:set --backend-frontname="admin_ddev" --no-interaction
# Login using `admin` user and `Password123` password
ddev launch /admin_ddev
```

Change the admin name and related information as needed.

??? tip "Prefer to run as a script?"
    To run the whole setup as a script, examine and run this script:

    ```bash
    cat > setup-magento.sh << 'EOF'
    #!/usr/bin/env bash
    set -euo pipefail
    export MAGENTO_HOSTNAME=my-magento2-site
    mkdir ${MAGENTO_HOSTNAME} && cd ${MAGENTO_HOSTNAME}
    ddev config --project-type=magento2 --docroot=pub --upload-dirs=media --disable-settings-management
    ddev add-on get ddev/ddev-opensearch
    ddev start -y
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
    ddev magento setup:config:set --backend-frontname="admin_ddev" --no-interaction
    ddev launch /admin_ddev
    EOF
    chmod +x setup-magento.sh
    ./setup-magento.sh
    ```

The admin login URL is specified by `frontName` in `app/etc/env.php`.

You may want to add the [Magento 2 Sample Data](https://experienceleague.adobe.com/docs/commerce-operations/installation-guide/next-steps/sample-data/composer-packages.html) with:

```
ddev magento sampledata:deploy
ddev magento setup:upgrade
```

## Moodle

=== "Composer"

    Create the project directory and configure DDEV:

    ```bash
    mkdir my-moodle-site && cd my-moodle-site
    ddev config --docroot=public --webserver-type=apache-fpm
    ```

    Start DDEV (this may take a minute):

    ```bash
    ddev start
    ```

    Install Moodle via Composer:

    ```bash
    ddev composer create-project moodle/moodle
    ```

    Run Moodle installation:

    ```bash
    ddev exec 'php admin/cli/install.php --non-interactive --agree-license --wwwroot=$DDEV_PRIMARY_URL --dbtype=mariadb --dbhost=db --dbname=db --dbuser=db --dbpass=db --fullname="DDEV Moodle Demo" --shortname=Demo --adminpass=password'
    ```

    Launch Moodle:

    ```bash
    ddev launch /login
    ```

    In the web browser, log into your account using `admin` and `password`.

    Visit the [Moodle Admin Quick Guide](https://docs.moodle.org/400/en/Admin_quick_guide) for more information.

    !!!tip
        Moodle relies on a periodic cron job—don't forget to set that up! See [ddev/ddev-cron](https://github.com/ddev/ddev-cron).

    ??? tip "Prefer to run as a script?"
        To run the whole setup as a script, examine and run this script:

        ```bash
        cat > setup-moodle.sh << 'EOF'
        #!/usr/bin/env bash
        set -euo pipefail
        mkdir my-moodle-site && cd my-moodle-site
        ddev config --docroot=public --webserver-type=apache-fpm
        ddev start -y
        ddev composer create-project moodle/moodle
        ddev exec 'php admin/cli/install.php --non-interactive --agree-license --wwwroot=$DDEV_PRIMARY_URL --dbtype=mariadb --dbhost=db --dbname=db --dbuser=db --dbpass=db --fullname="DDEV Moodle Demo" --shortname=Demo --adminpass=password'
        ddev launch /login
        EOF
        chmod +x setup-moodle.sh
        ./setup-moodle.sh
        ```

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

    SvelteKit requires just a bit of configuration to make it run. There are many ways to make any Node.js site work, these are just examples. The `svelte.config.js` and `vite.config.js` used above can be adapted in many ways. For more comprehensive Vite configuration options, see the [Vite Integration](usage/vite.md) documentation.

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

    Create the project directory and configure DDEV:

    ```bash
    mkdir my-openmage-site && cd my-openmage-site
    ddev config --project-type=magento --docroot=public_test --php-version=8.1 --web-environment-add=MAGE_IS_DEVELOPER_MODE=1
    ```

    Start DDEV (this may take a minute):

    ```bash
    ddev start
    ```

    Initialize and configure Composer:

    ```bash
    ddev composer init --name "openmage/composer-test" --description "OpenMage starter project" --type "project" -l "OSL-3.0" -s "dev" -q
    ddev composer config extra.magento-root-dir "public_test"
    ddev composer config extra.enable-patching true
    ddev composer config extra.magento-core-package-type "magento-source"
    ddev composer config allow-plugins.cweagans/composer-patches true
    ddev composer config allow-plugins.magento-hackathon/magento-composer-installer true
    ddev composer config allow-plugins.aydin-hassan/magento-core-composer-installer true
    ddev composer config allow-plugins.openmage/composer-plugin true
    ddev composer require --no-update "aydin-hassan/magento-core-composer-installer":"^2.1.0" "openmage/magento-lts":"^20.13"
    ```

    Download the OpenMage install command and install dependencies:

    ```bash
    ddev exec wget -O .ddev/commands/web/openmage-install https://raw.githubusercontent.com/OpenMage/magento-lts/refs/heads/main/.ddev/commands/web/openmage-install
    ddev composer install
    ```

    Run OpenMage silent installation with sample data:

    ```bash
    ddev openmage-install -q
    ```

    Launch the admin interface (login with `admin` and `veryl0ngpassw0rd`):

    ```bash
    ddev launch /admin
    ```

    !!!note "Make sure that `docroot` is set correctly"
        DDEV config `--docroot` has to match Composer config `extra.magento-root-dir`.

    ??? tip "Prefer to run as a script?"
        To run the whole setup as a script, examine and run this script:

        ```bash
        cat > setup-openmage.sh << 'EOF'
        #!/usr/bin/env bash
        set -euo pipefail
        mkdir my-openmage-site && cd my-openmage-site
        ddev config --project-type=magento --docroot=public_test --php-version=8.1 --web-environment-add=MAGE_IS_DEVELOPER_MODE=1
        ddev start -y
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
        ddev openmage-install -q
        ddev launch /admin
        EOF
        chmod +x setup-openmage.sh
        ./setup-openmage.sh
        ```

=== "Git Clone (for contributors)"

    Create the project directory and clone the repository:

    ```bash
    mkdir my-openmage-site && cd my-openmage-site
    git clone https://github.com/OpenMage/magento-lts .
    ```

    Configure and start DDEV:

    ```bash
    ddev config --project-type=magento --php-version=8.1 --web-environment-add=MAGE_IS_DEVELOPER_MODE=1
    ddev start
    ```

    Install Composer dependencies:

    ```bash
    ddev composer install
    ```

    Run OpenMage silent installation with sample data:

    ```bash
    ddev openmage-install -q
    ```

    Launch the admin interface (login with `admin` and `veryl0ngpassw0rd`):

    ```bash
    ddev launch /admin
    ```

    Note that OpenMage itself provides several custom DDEV commands, including `openmage-install`, `openmage-admin`, `phpmd`, `rector`, `phpcbf`, `phpstan`, `vendor-patches`, and `php-cs-fixer`.

    ??? tip "Prefer to run as a script?"
        To run the whole setup as a script, examine and run this script:

        ```bash
        cat > setup-openmage-git.sh << 'EOF'
        #!/usr/bin/env bash
        set -euo pipefail
        mkdir my-openmage-site && cd my-openmage-site
        git clone https://github.com/OpenMage/magento-lts .
        ddev config --project-type=magento --php-version=8.1 --web-environment-add=MAGE_IS_DEVELOPER_MODE=1
        ddev start -y
        ddev composer install
        ddev openmage-install -q
        ddev launch /admin
        EOF
        chmod +x setup-openmage-git.sh
        ./setup-openmage-git.sh
        ```

## Pimcore

=== "Composer"

    Using the [Pimcore skeleton](https://github.com/pimcore/skeleton) repository:

    Create the project directory and configure DDEV:

    ``` bash
    mkdir my-pimcore-site && cd my-pimcore-site
    ddev config --project-type=php --docroot=public --webimage-extra-packages='php${DDEV_PHP_VERSION}-amqp'
    ```

    Start DDEV (this may take a minute):

    ```bash
    ddev start
    ```

    Install Pimcore via Composer:

    ```bash
    ddev composer create-project pimcore/skeleton
    ```

    Run Pimcore installation:

    ```bash
    ddev exec pimcore-install \
        --mysql-username=db \
        --mysql-password=db \
        --mysql-host-socket=db \
        --mysql-database=db \
        --admin-password=admin \
        --admin-username=admin
    ```

    Create consumer daemon configuration:

    ```bash
    echo "web_extra_daemons:
      - name: consumer
        command: 'while true; do /var/www/html/bin/console messenger:consume pimcore_core pimcore_maintenance pimcore_scheduled_tasks pimcore_image_optimize pimcore_asset_update --memory-limit=250M --time-limit=3600; done'
        directory: /var/www/html" >.ddev/config.pimcore.yaml
    ```

    Restart and launch:

    ```bash
    ddev restart
    ddev launch /admin
    ```

    ??? tip "Prefer to run as a script?"
        To run the whole setup as a script, examine and run this script:

        ```bash
        cat > setup-pimcore.sh << 'EOF'
        #!/usr/bin/env bash
        set -euo pipefail
        mkdir my-pimcore-site && cd my-pimcore-site
        ddev config --project-type=php --docroot=public --webimage-extra-packages='php${DDEV_PHP_VERSION}-amqp'
        ddev start -y
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
        EOF
        chmod +x setup-pimcore.sh
        ./setup-pimcore.sh
        ```

## ProcessWire

To get started with [ProcessWire](https://processwire.com/), create a new directory and use the ZIP file download, composer, or Git checkout to build. These instructions are adapted from [ProcessWire Install Documentation](https://processwire.com/docs/start/install/new/#installing-processwire).

=== "ZIP File"

    Create the project directory:

    ```bash
    mkdir my-processwire-site && cd my-processwire-site
    ```

    Download and extract ProcessWire:

    ```bash
    curl -LJOf https://github.com/processwire/processwire/archive/master.zip
    unzip processwire-master.zip && rm -f processwire-master.zip && mv processwire-master/* . && mv processwire-master/.* . 2>/dev/null && rm -rf processwire-master
    ```

    Configure and start DDEV:

    ```bash
    ddev config --project-type=php --webserver-type=apache-fpm
    ddev start
    ```

    Launch ProcessWire:

    ```bash
    ddev launch
    ```

    ??? tip "Prefer to run as a script?"
        To run the whole setup as a script, examine and run this script:

        ```bash
        cat > setup-processwire-zip.sh << 'EOF'
        #!/usr/bin/env bash
        set -euo pipefail
        mkdir my-processwire-site && cd my-processwire-site
        curl -LJOf https://github.com/processwire/processwire/archive/master.zip
        unzip processwire-master.zip && rm -f processwire-master.zip && mv processwire-master/* . && mv processwire-master/.* . 2>/dev/null && rm -rf processwire-master
        ddev config --project-type=php --webserver-type=apache-fpm
        ddev start -y
        ddev launch
        EOF
        chmod +x setup-processwire-zip.sh
        ./setup-processwire-zip.sh
        ```

=== "Composer"

    Create the project directory and configure DDEV:

    ```bash
    mkdir my-processwire-site && cd my-processwire-site
    ddev config --project-type=php --webserver-type=apache-fpm
    ```

    Start DDEV (this may take a minute):

    ```bash
    ddev start
    ```

    Install ProcessWire via Composer:

    ```bash
    ddev composer create-project "processwire/processwire:^3"
    ```

    Launch ProcessWire:

    ```bash
    ddev launch
    ```

    ??? tip "Prefer to run as a script?"
        To run the whole setup as a script, examine and run this script:

        ```bash
        cat > setup-processwire.sh << 'EOF'
        #!/usr/bin/env bash
        set -euo pipefail
        mkdir my-processwire-site && cd my-processwire-site
        ddev config --project-type=php --webserver-type=apache-fpm
        ddev start -y
        ddev composer create-project "processwire/processwire:^3"
        ddev launch
        EOF
        chmod +x setup-processwire.sh
        ./setup-processwire.sh
        ```

=== "Git"

    Create the project directory:

    ```bash
    mkdir my-processwire-site && cd my-processwire-site
    ```

    Clone ProcessWire (main branch for stable release):

    ```bash
    git clone https://github.com/processwire/processwire.git .
    # For latest features, use dev branch instead:
    # git clone -b dev https://github.com/processwire/processwire.git .
    ```

    Configure and start DDEV:

    ```bash
    ddev config --webserver-type=apache-fpm
    ddev start
    ```

    Launch ProcessWire:

    ```bash
    ddev launch
    ```

    ??? tip "Prefer to run as a script?"
        To run the whole setup as a script, examine and run this script:

        ```bash
        cat > setup-processwire-git.sh << 'EOF'
        #!/usr/bin/env bash
        set -euo pipefail
        mkdir my-processwire-site && cd my-processwire-site
        git clone https://github.com/processwire/processwire.git .
        ddev config --webserver-type=apache-fpm
        ddev start -y
        ddev launch
        EOF
        chmod +x setup-processwire-git.sh
        ./setup-processwire-git.sh
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

```bash
ddev config --upload-dirs=sites/assets/files && ddev restart
```

If you have any questions there is lots of help in the [DDEV thread in the ProcessWire forum](https://processwire.com/talk/topic/27433-using-ddev-for-local-processwire-development-tips-tricks/).

## Shopware

=== "Composer"

    Though you can set up a Shopware 6 environment many ways, we recommend the following technique. DDEV creates a `.env.local` file for you by default; if you already have one DDEV adds necessary information to it. When `ddev composer create-project` asks if you want to include Docker configuration, answer `x`, as this approach does not use their Docker configuration.

    Create the project directory and configure DDEV:

    ```bash
    mkdir my-shopware-site && cd my-shopware-site
    ddev config --project-type=shopware6 --docroot=public
    ```

    Start DDEV (this may take a minute):

    ```bash
    ddev start
    ```

    Install Shopware via Composer:

    ```bash
    ddev composer create-project shopware/production
    # If it asks `Do you want to include Docker configuration from recipes?`
    # answer `x`, as we're using DDEV for this rather than its recipes.
    ```

    Run Shopware installation and launch:

    ```bash
    ddev exec console system:install --basic-setup
    ddev launch /admin
    # Default username and password are `admin` and `shopware`
    ```

    Log into the admin site (`/admin`) using the web browser. The default credentials are username `admin` and password `shopware`. You can use the web UI to install sample data or accomplish many other tasks.

    For more advanced tasks like adding elasticsearch, building and watching storefront and administration, see [susi.dev](https://susi.dev/ddev-shopware-6).

    ??? tip "Prefer to run as a script?"
        To run the whole setup as a script, examine and run this script:

        ```bash
        cat > setup-shopware.sh << 'EOF'
        #!/usr/bin/env bash
        set -euo pipefail
        mkdir my-shopware-site && cd my-shopware-site
        ddev config --project-type=shopware6 --docroot=public
        ddev start -y
        ddev composer create-project shopware/production
        ddev exec console system:install --basic-setup
        ddev launch /admin
        EOF
        chmod +x setup-shopware.sh
        ./setup-shopware.sh
        ```

## Silverstripe CMS

Use a new or existing Composer project, or clone a Git repository.

=== "Composer"

    Create the project directory and configure DDEV:

    ```bash
    mkdir my-silverstripe-site && cd my-silverstripe-site
    ddev config --project-type=silverstripe --docroot=public
    ```

    Start DDEV (this may take a minute):

    ```bash
    ddev start
    ```

    Install Silverstripe via Composer:

    ```bash
    ddev composer create-project --prefer-dist silverstripe/installer
    ddev sake dev/build flush=all
    ```

    Launch the admin area:

    ```bash
    ddev launch /admin
    ```

    ??? tip "Prefer to run as a script?"
        To run the whole setup as a script, examine and run this script:

        ```bash
        cat > setup-silverstripe.sh << 'EOF'
        #!/usr/bin/env bash
        set -euo pipefail
        mkdir my-silverstripe-site && cd my-silverstripe-site
        ddev config --project-type=silverstripe --docroot=public
        ddev start -y
        ddev composer create-project --prefer-dist silverstripe/installer
        ddev sake dev/build flush=all
        ddev launch /admin
        EOF
        chmod +x setup-silverstripe.sh
        ./setup-silverstripe.sh
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

    Create the project directory and configure DDEV:

    ```bash
    mkdir my-statamic-site && cd my-statamic-site
    ddev config --project-type=laravel --docroot=public
    ```

    Install Statamic via Composer:

    ```bash
    ddev composer create-project --prefer-dist statamic/statamic
    ```

    Create admin user and launch:

    ```bash
    ddev php please make:user admin@example.com --password=admin1234 --super --no-interaction
    ddev launch /cp
    ```

    ??? tip "Prefer to run as a script?"
        To run the whole setup as a script, examine and run this script:

        ```bash
        cat > setup-statamic.sh << 'EOF'
        #!/usr/bin/env bash
        set -euo pipefail
        mkdir my-statamic-site && cd my-statamic-site
        ddev config --project-type=laravel --docroot=public
        ddev composer create-project --prefer-dist statamic/statamic
        ddev php please make:user admin@example.com --password=admin1234 --super --no-interaction
        ddev launch /cp
        EOF
        chmod +x setup-statamic.sh
        ./setup-statamic.sh
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

Create the project directory and configure DDEV:

```bash
mkdir my-sulu-site && cd my-sulu-site
ddev config --project-type=php --docroot=public --upload-dirs=uploads --database=mysql:8.0
```

Start DDEV (this may take a minute):

```bash
ddev start
```

Install Sulu via Composer:

```bash
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

Configure webspace settings:

```bash
export SULU_PROJECT_NAME="My Sulu Site"
export SULU_PROJECT_KEY="my-sulu-site"
export SULU_PROJECT_CONFIG_FILE="config/webspaces/my-sulu-site.xml"
ddev exec "mv config/webspaces/website.xml ${SULU_PROJECT_CONFIG_FILE}"
ddev exec "sed -i -e 's|<name>.*</name>|<name>${SULU_PROJECT_NAME}</name>|g' -e 's|<key>.*</key>|<key>${SULU_PROJECT_KEY}</key>|g' ${SULU_PROJECT_CONFIG_FILE}"
```

!!!warning "Caution"
    Changing the `<key>` for a webspace later on causes problems. It is recommended to decide on the value for the key before the database is build in the next step.

Build the database with the `dev` argument (adds user `admin` with password `admin`):

```bash
ddev dotenv set .env.local --app-env=dev --database-url="mysql://db:db@db:3306/db?serverVersion=8.0&charset=utf8mb4"
ddev exec bin/adminconsole sulu:build dev --no-interaction
```

Launch Sulu (login with `admin` and `admin`):

```bash
ddev launch /admin
```

!!!tip
    If you don't want to add an admin user use the `prod` argument instead

    ```bash
    ddev execute bin/adminconsole sulu:build prod
    ```

??? tip "Prefer to run as a script?"
    To run the whole setup as a script, examine and run this script:

    ```bash
    cat > setup-sulu.sh << 'EOF'
    #!/usr/bin/env bash
    set -euo pipefail
    mkdir my-sulu-site && cd my-sulu-site
    ddev config --project-type=php --docroot=public --upload-dirs=uploads --database=mysql:8.0
    ddev start -y
    ddev composer create-project sulu/skeleton
    export SULU_PROJECT_NAME="My Sulu Site"
    export SULU_PROJECT_KEY="my-sulu-site"
    export SULU_PROJECT_CONFIG_FILE="config/webspaces/my-sulu-site.xml"
    ddev exec "mv config/webspaces/website.xml ${SULU_PROJECT_CONFIG_FILE}"
    ddev exec "sed -i -e 's|<name>.*</name>|<name>${SULU_PROJECT_NAME}</name>|g' -e 's|<key>.*</key>|<key>${SULU_PROJECT_KEY}</key>|g' ${SULU_PROJECT_CONFIG_FILE}"
    ddev dotenv set .env.local --app-env=dev --database-url="mysql://db:db@db:3306/db?serverVersion=8.0&charset=utf8mb4"
    ddev exec bin/adminconsole sulu:build dev --no-interaction
    ddev launch /admin
    EOF
    chmod +x setup-sulu.sh
    ./setup-sulu.sh
    ```

## Symfony

There are many ways to install Symfony, here are a few of them based on the [Symfony docs](https://symfony.com/doc/current/setup.html).

DDEV automatically updates or creates the `.env.local` file with the database information.

=== "Composer"

    Create the project directory and configure DDEV:

    ```bash
    mkdir my-symfony-site && cd my-symfony-site
    ddev config --project-type=symfony --docroot=public
    ```

    Start DDEV (this may take a minute):

    ```bash
    ddev start
    ```

    Install Symfony via Composer:

    ```bash
    ddev composer create-project symfony/skeleton
    ddev composer require webapp
    # When it asks if you want to include docker configuration, say "no" with "x"
    ```

    Launch the site:

    ```bash
    ddev launch
    ```

    ??? tip "Prefer to run as a script?"
        To run the whole setup as a script, examine and run this script:

        ```bash
        cat > setup-symfony.sh << 'EOF'
        #!/usr/bin/env bash
        set -euo pipefail
        mkdir my-symfony-site && cd my-symfony-site
        ddev config --project-type=symfony --docroot=public
        ddev start -y
        ddev composer create-project symfony/skeleton
        ddev composer require webapp
        ddev launch
        EOF
        chmod +x setup-symfony.sh
        ./setup-symfony.sh
        ```

=== "Symfony CLI"

    ```bash
    mkdir my-symfony-site && cd my-symfony-site
    ddev config --project-type=symfony --docroot=public
    ddev start
    ddev exec symfony check:requirements
    ddev exec symfony new temp --webapp
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

    Create the project directory and configure DDEV:

    ```bash
    PROJECT_NAME=my-typo3-site
    mkdir ${PROJECT_NAME} && cd ${PROJECT_NAME}
    ddev config --project-type=typo3 --docroot=public
    ```

    Start DDEV (this may take a minute):

    ```bash
    ddev start
    ```

    Install TYPO3 via Composer:

    ```bash
    ddev composer create-project "typo3/cms-base-distribution:^14"
    ```

    Run the TYPO3 setup:

    ```bash
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
        --project-name="My TYPO3 site" \
        --force
    ```

    Launch the site:

    ```bash
    ddev launch /typo3/
    ```

    ??? tip "Prefer to run as a script?"
        To run the whole setup as a script, examine and run this script:

        ```bash
        cat > setup-typo3.sh << 'EOF'
        #!/usr/bin/env bash
        set -euo pipefail
        PROJECT_NAME=my-typo3-site
        mkdir -p ${PROJECT_NAME} && cd ${PROJECT_NAME}
        ddev config --project-type=typo3 --docroot=public
        ddev start -y
        ddev composer create-project "typo3/cms-base-distribution:^14"
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
            --project-name="My TYPO3 site" \
            --force
        ddev launch /typo3/
        EOF
        chmod +x setup-typo3.sh
        ./setup-typo3.sh
        ```

=== "Git Clone"

    This example uses a clone of a test repository, `github.com/ddev/test-typo3.git`. 
    Replace that with your git repository.

    ```bash
    PROJECT_GIT_URL=https://github.com/ddev/test-typo3.git
    PROJECT_NAME=my-typo3-site
    mkdir -p ${PROJECT_NAME} && cd ${PROJECT_NAME}
    git clone ${PROJECT_GIT_REPOSITORY} .
    ddev config --project-type=typo3 --docroot=public
    ddev start
    ddev composer install
    ddev exec touch public/FIRST_INSTALL
    ddev launch /typo3/install.php
    ```

## Wagtail (Python, Generic)

[Wagtail](https://wagtail.org/) is a popular, open-source content management system built on the Django web framework. This quickstart demonstrates how to set up a new Wagtail project using DDEV with Python and a virtual environment.

Create the project directory and configure DDEV:

```bash
export WAGTAIL_SITENAME=my-wagtail-site
mkdir ${WAGTAIL_SITENAME} && cd ${WAGTAIL_SITENAME}
ddev config --project-type=generic --webserver-type=generic \
    --webimage-extra-packages=python3-pip,python3-venv \
    --web-environment-add=DJANGO_SETTINGS_MODULE=mysite.settings.dev \
    --omit-containers=db
```

Configure the container to automatically activate the Python virtual environment:

```bash
cat <<'DOCKERFILEEND' >.ddev/web-build/Dockerfile.python-venv
RUN for file in /etc/bash.bashrc /etc/bash.nointeractive.bashrc; do \
        echo '[ -s "$DDEV_APPROOT/env/bin/activate" ] && source "$DDEV_APPROOT/env/bin/activate"' >> "$file"; \
    done
DOCKERFILEEND
```

Start DDEV (this may take a minute):

```bash
ddev start
```

Create a Python virtual environment and install Wagtail:

```bash
ddev exec python -m venv env
ddev exec pip install wagtail gunicorn
```

Initialize the Wagtail project:

```bash
ddev exec wagtail start mysite .
ddev exec pip install -r requirements.txt
```

Configure Django to detect HTTPS behind the [Traefik](./extend/traefik-router.md) router:

```bash
ddev exec "echo \"SECURE_PROXY_SSL_HEADER = ('HTTP_X_FORWARDED_PROTO', 'https')\" >> mysite/settings/dev.py"
```

Run database migrations and create a superuser:

```bash
ddev exec python manage.py migrate --noinput
ddev exec "DJANGO_SUPERUSER_PASSWORD=admin python manage.py createsuperuser --username=admin --email=admin@example.com --noinput"
```

Configure DDEV to run the Wagtail development server:

```bash
cat <<'EOF' > .ddev/config.wagtail.yaml
web_extra_daemons:
    - name: "wagtail"
      command: "gunicorn mysite.wsgi:application -b 0.0.0.0:8000"
      directory: /var/www/html
web_extra_exposed_ports:
    - name: "wagtail"
      container_port: 8000
      http_port: 80
      https_port: 443
EOF
```

Restart DDEV to apply the configuration:

```bash
ddev restart
```

Launch the Wagtail admin interface (login with `admin` and `admin`):

```bash
ddev launch /admin
```

??? tip "Prefer to run as a script?"
    To run the whole setup as a script, examine and run this script:

    ```bash
    cat > setup-wagtail.sh << 'EOF'
    #!/usr/bin/env bash
    set -euo pipefail
    export WAGTAIL_SITENAME=my-wagtail-site
    mkdir ${WAGTAIL_SITENAME} && cd ${WAGTAIL_SITENAME}
    ddev config --project-type=generic --webserver-type=generic \
        --webimage-extra-packages=python3-pip,python3-venv \
        --web-environment-add=DJANGO_SETTINGS_MODULE=mysite.settings.dev \
        --omit-containers=db
    cat <<'INNEREOF' >.ddev/web-build/Dockerfile.python-venv
    RUN for file in /etc/bash.bashrc /etc/bash.nointeractive.bashrc; do \
            echo '[ -s "$DDEV_APPROOT/env/bin/activate" ] && source "$DDEV_APPROOT/env/bin/activate"' >> "$file"; \
        done
    INNEREOF
    ddev start -y
    ddev exec python -m venv env
    ddev exec pip install wagtail gunicorn
    ddev exec wagtail start mysite .
    ddev exec pip install -r requirements.txt
    ddev exec "echo \"SECURE_PROXY_SSL_HEADER = ('HTTP_X_FORWARDED_PROTO', 'https')\" >> mysite/settings/dev.py"
    ddev exec python manage.py migrate --noinput
    ddev exec "DJANGO_SUPERUSER_PASSWORD=admin python manage.py createsuperuser --username=admin --email=admin@example.com --noinput"
    cat <<'INNEREOF' > .ddev/config.wagtail.yaml
    web_extra_daemons:
        - name: "wagtail"
          command: "gunicorn mysite.wsgi:application -b 0.0.0.0:8000"
          directory: /var/www/html
    web_extra_exposed_ports:
        - name: "wagtail"
          container_port: 8000
          http_port: 80
          https_port: 443
    INNEREOF
    ddev restart
    ddev launch /admin
    EOF
    chmod +x setup-wagtail.sh
    ./setup-wagtail.sh
    ```

## WordPress

There are several easy ways to use DDEV with WordPress:

=== "WP-CLI"

    DDEV has built-in support for [WP-CLI](https://wp-cli.org/), the command-line interface for WordPress.

    Create the project directory and configure DDEV:

    ```bash
    mkdir my-wp-site && cd my-wp-site
    # Create a new DDEV project inside the newly-created folder
    # (Primary URL automatically set to `https://<folder>.ddev.site`)
    ddev config --project-type=wordpress
    ```

    Start DDEV (this may take a minute):

    ```bash
    ddev start
    ```

    Download WordPress:

    ```bash
    ddev wp core download
    ```

    Install WordPress (or use `ddev launch` to install via browser):

    ```bash
    # You can launch in browser to finish installation:
    # ddev launch
    # OR use the following installation command
    # (we need to use single quotes to get the primary site URL from `.ddev/config.yaml` as variable)
    ddev wp core install --url='$DDEV_PRIMARY_URL' --title='My WordPress site' --admin_user=admin --admin_password=admin --admin_email=admin@example.com
    ```

    Launch WordPress admin dashboard:

    ```bash
    ddev launch wp-admin/
    ```

    ??? tip "Prefer to run as a script?"
        To run the whole setup as a script, examine and run this script:

        ```bash
        cat > setup-wordpress.sh << 'EOF'
        #!/usr/bin/env bash
        set -euo pipefail
        mkdir my-wp-site && cd my-wp-site
        ddev config --project-type=wordpress
        ddev start -y
        ddev wp core download
        ddev wp core install --url='$DDEV_PRIMARY_URL' --title='My WordPress site' --admin_user=admin --admin_password=admin --admin_email=admin@example.com
        ddev launch wp-admin/
        EOF
        chmod +x setup-wordpress.sh
        ./setup-wordpress.sh
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
