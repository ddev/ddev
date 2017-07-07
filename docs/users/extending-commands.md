<h1>Extending ddev Commands</h1>

Certain ddev commands provide hooks to run tasks before or after the main command executes. These tasks can be defined in the config.yaml for your site, and allow for you to automate setup tasks specific to your site. To define command tasks in your configuration, specify the desired command hook as a subfield to `hooks`, then provide a list of tasks to run.

Example:

```
hooks:
  $command-hook:
    - task: value
```

## Supported Command Hooks

- `pre-start`: Hooks into "ddev start". Execute tasks before the site environment starts. **Note:** Only `exec-host` tasks can be run successfully for pre-start. See Supported Tasks below for more info.
- `post-start`: Hooks into "ddev start". Execute tasks after the site environment has started
- `pre-import-db`: Hooks into "ddev import-db". Execute tasks before database import
- `post-import-db`: Hooks into "ddev import-db". Execute tasks after database import
- `pre-import-files`: Hooks into "ddev import-files". Execute tasks before files are imported
- `post-import-files`: Hooks into "ddev import-files". Execute tasks after files are imported.

## Supported Tasks

### `exec`: Execute a shell command in the web service container.

Value: string providing the command to run. Commands requiring user interaction are not supported.

Example:

_Use drush to clear the Drupal cache after database import_

```
hooks:
  post-import-db:
    - exec: "drush cc all"
```

Example:

_Use wp-cli to replace the production URL with development URL in the database of a WordPress site_

```
hooks:
  post-import-db:
    - exec: "wp search-replace https://www.myproductionsite.com http://mydevsite.ddev.local"
```

### `exec-host`: Execute a shell command on your system.

Value: string providing the command to run. Commands requiring user interaction are not supported.

Example:

_Run "composer install" from your system before starting the site (composer would nee to be installed on your system)_

```
hooks:
  pre-start:
    - exec: "composer install"
```

## WordPress Example

```
hooks:
  post-start:
    # Install WordPress after start
    - exec: "wp config create --dbname=db --dbuser=db --dbpass=db --dbhost=db"
    - exec: "wp core install --url=http://mysite.ddev.local --title=MySite --admin_user=admin --admin_email=admin@mail.test"
  post-import-db:
    # Update the URL of your site throughout your database after import
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
    # Set the site name
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
