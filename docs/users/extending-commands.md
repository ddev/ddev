
Most ddev commands provide hooks to run tasks before or after the main command executes. To automate setup tasks specific to your project, define them in the project's config.yaml file.

To define command tasks in your configuration, specify the desired command hook as a subfield to `hooks`, then provide a list of tasks to run.

Example:

```yaml
hooks:
  post-start:
    - exec: "simple command expression"
    - exec: "ls >/dev/null && touch /var/www/html/somefile.txt"
    - exec-host: "simple command expression"
  post-import-db:
    - exec: "drush uli"
```

## Supported Command Hooks

* `pre-start`: Hooks into "ddev start". Execute tasks before the project environment starts. **Note:** Only `exec-host` tasks can be generally run successfully during pre-start. See Supported Tasks below for more info.
* `post-start`: Execute tasks after the project environment has started.
* `pre-import-db` and `post-import-db`: Execute tasks before or after database import.
* `pre-import-files` and `post-import-files`: Execute tasks before or after files are imported
* `pre-composer` and `post-composer`: Execute tasks before or after the `composer` command.
* `pre-stop`, `pre-config`, `post-config`, `pre-exec`, `post-exec`, `pre-pause`, `post-pause`, `pre-pull`, `post-pull`, `pre-push`, `post-push`, `pre-snapshot`, `post-snapshot`, `pre-restore-snapshot`, `post-restore-snapshot`: Execute as the name suggests.
* `post-stop`: Hooks into "ddev stop". Execute tasks after the project environment stopped. **Note:** Only `exec-host` tasks can be generally run successfully during post-stop.

## Supported Tasks

ddev currently supports these tasks:

* `exec` to execute a command in any service/container
* `exec-host` to execute a command on the host
* `composer` to execute a composer command in the web container

### `exec`: Execute a shell command in a container (defaults to web container)

Value: string providing the command to run. Commands requiring user interaction are not supported. You can also add a "service" key to the command, specifying to run it on the db container or any other container you use.

Example: _Use drush to clear the Drupal cache and get a user login link after database import_

```yaml
hooks:
  post-import-db:
    - exec: drush cr
    - exec: drush uli
```

Example: _Use wp-cli to replace the production URL with development URL in the database of a WordPress project_

```yaml
hooks:
  post-import-db:
    - exec: wp search-replace https://www.myproductionsite.com http://mydevsite.ddev.site
```

Example: _Add an extra database before import-db, executing in db container_

```yaml
hooks:
  pre-import-db:
    - exec: mysql -uroot -proot -e "CREATE DATABASE IF NOT EXISTS some_new_database;"
      service: db

```

Example: _Add the common "ll" alias into the web container .bashrc file_

```yaml
hooks:
  post-start:
  - exec: sudo echo alias ll=\"ls -lhA\" >> ~/.bashrc
```

(Note that this could probably be done more efficiently in a .ddev/web-build/Dockerfile as explained in [Customizing Images](extend/customizing-images.md).)

### `exec-host`: Execute a shell command on the host system

Value: string providing the command to run. Commands requiring user interaction are not supported.

Example: Run "composer install" from your system before starting the project (composer must already be installed on the host workstation):

```yaml
hooks:
  pre-start:
    - exec-host: "composer install"
```

### `composer`: Execute a composer command in the web container

Value: string providing the composer command to run.

Example:

```yaml
hooks:
  post-start:
    - composer: config discard-changes true
```

## WordPress Example

```yaml
hooks:
  post-start:
    # Install WordPress after start
    - exec: "wp config create --dbname=db --dbuser=db --dbpass=db --dbhost=db"
    - exec: "wp core install --url=http://mysite.ddev.site --title=MySite --admin_user=admin --admin_email=admin@mail.test"
  post-import-db:
    # Update the URL of your project throughout your database after import
    - exec: "wp search-replace https://www.myproductionsite.com http://mydevsite.ddev.site"
```

## Drupal 7 Example

```yaml
hooks:
  post-start:
    # Install Drupal after start if not installed already
    - exec: "(drush status bootstrap | grep -q Successful) || drush site-install -y --db-url=db:db@db/db"
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

```yaml
hooks:
  pre-start:
    # Install composer dependencies using composer on host system
    - exec-host: "composer install"
  post-start:
    # Install Drupal after start if not installed already
    - exec: "(drush status bootstrap | grep -q Successful) || drush site-install -y --db-url=mysql://db:db@db/db"
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

```yaml
hooks:
    post-start:
      - composer: install
```

## Adding Additional Debian Packages (PHP Modules) Example

```yaml
webimage_extra_packages: ["php-bcmath", "php7.4-tidy"]
dbimage_extra_packages: ["vim"]
```
