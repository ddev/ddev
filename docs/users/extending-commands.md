<h1>Extending ddev Commands</h1>

Certain ddev commands provide hooks to run tasks before or after the main command executes. To automate setup tasks specific to your project, define them in the project's config.yaml file.

To define command tasks in your configuration, specify the desired command hook as a subfield to `hooks`, then provide a list of tasks to run.

_Note: Only simple commands are currently supported, so if you need to handle multiple commands, put them in as separate tasks. Shell pipes, &&, ||, and related bash/shell expressions are not yet supported._

Example:

```
hooks:
  post-start:
    - exec: "simple command expression"
    - rexec: "simple command expression"
    - exec-host: "simple command expression"
  post-import-db:
    - exec-host: "drush uli"
```

## Supported Command Hooks

- `pre-start`: Hooks into "ddev start". Execute tasks before the project environment starts. **Note:** Only `exec-host` tasks can be run successfully for pre-start. See Supported Tasks below for more info.
- `post-start`: Hooks into "ddev start". Execute tasks after the project environment has started
- `pre-import-db`: Hooks into "ddev import-db". Execute tasks before database import
- `post-import-db`: Hooks into "ddev import-db". Execute tasks after database import
- `pre-import-files`: Hooks into "ddev import-files". Execute tasks before files are imported
- `post-import-files`: Hooks into "ddev import-files". Execute tasks after files are imported.

## Supported Tasks

### `exec`: Execute a shell command in the web service container.

Value: string providing the command to run. Commands requiring user interaction are not supported.

Example:

_Use drush to clear the Drupal cache and get a user login link after database import_

```
hooks:
  post-import-db:
    - exec: "drush cc all"
    - exec: "drush uli"
```

Example:

_Use wp-cli to replace the production URL with development URL in the database of a WordPress project_

```
hooks:
  post-import-db:
    - exec: "wp search-replace https://www.myproductionsite.com http://mydevsite.ddev.local"
```

### `rexec`: Execute a shell command in the web service container as root.

Value: string providing the command to run as root. Commands requiring user interaction are not supported.

Example:

_Use apt to install the php-sqlite3 package_

```
hooks:
  post-start:
    - rexec: "apt-get update"
    - rexec: "apt-get install -y php-sqlite3"
```


### `exec-host`: Execute a shell command on the host system.

Value: string providing the command to run. Commands requiring user interaction are not supported.

Example:

_Run "composer install" from your system before starting the project (composer must already be installed on the host workstation)_

```
hooks:
  pre-start:
    - exec-host: "composer install"
```

## WordPress Example

```
hooks:
  post-start:
    # Install WordPress after start
    - exec: "wp config create --dbname=db --dbuser=db --dbpass=db --dbhost=db"
    - exec: "wp core install --url=http://mysite.ddev.local --title=MySite --admin_user=admin --admin_email=admin@mail.test"
  post-import-db:
    # Update the URL of your project throughout your database after import
    - exec: "wp search-replace https://www.myproductionsite.com http://mydevsite.ddev.local"
```

## Drupal 7 Example

```
hooks:
  post-start:
    # Install Drupal after start
    - exec: "drush site-install -y --db-url=db:db@db/db"
    # Generate a one-time login link for the admin account.
    - exec: "drush uli 1"
  post-import-db:
    # Set the project name
    - exec: "drush vset site_name MyDevSite"
    # Enable the environment indicator module
    - exec: "drush en -y environment_indicator"
    # Clear the cache
    - exec: "drush cc all"
```

## Drupal 8 Example

```
hooks:
  pre-start:
    # Install composer dependencies using composer on host system
    - exec-host: "composer install"
  post-start:
    # Install Drupal after start
    - exec: "drush site-install -y --db-url=mysql://db:db@db/db"
    # Generate a one-time login link for the admin account.
    - exec: "drush uli 1"
  post-import-db:
    # Set the site name
    - exec: "drush config-set system.site name MyDevSite"
    # Enable the environment indicator module
    - exec: "drush en -y environment_indicator"
    # Clear the cache
    - exec: "drush cr"
```

## TYPO3 Example

```
hooks:
    post-start:
      - exec: "composer install -d /var/www/html/"
```
