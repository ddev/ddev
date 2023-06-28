# Hooks

Most DDEV commands provide hooks to run tasks before or after the main command executes. To automate setup tasks specific to your project, define them in the project’s `config.yaml` file.

To define command tasks in your configuration, specify the desired command hook as a subfield to `hooks`, then a list of tasks to run:

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

* `pre-start`: Hooks into [`ddev start`](../usage/commands.md#start). Execute tasks before the project environment starts.

    !!!tip
        Only `exec-host` tasks can run during `pre-start` because the containers are not yet running. See [Supported Tasks](#supported-tasks) below.

* `post-start`: Execute tasks after the project environment has started.
* `pre-import-db` and `post-import-db`: Execute tasks before or after database import.
* `pre-import-files` and `post-import-files`: Execute tasks before or after files are imported.
* `pre-composer` and `post-composer`: Execute tasks before or after the `composer` command.
* `pre-stop`, `pre-config`, `post-config`, `pre-exec`, `post-exec`, `pre-pull`, `post-pull`, `pre-push`, `post-push`, `pre-snapshot`, `post-snapshot`, `pre-restore-snapshot`, `post-restore-snapshot`: Execute as the name suggests.
* `post-stop`: Hooks into [`ddev stop`](../usage/commands.md#stop). Execute tasks after the project environment stopped.

    !!!tip
        Only `exec-host` tasks can run during `post-stop`. See [Supported Tasks](#supported-tasks) below.

## Supported Tasks

DDEV currently supports these tasks:

* `exec` to execute a command in any service/container.
* `exec-host` to execute a command on the host.
* `composer` to execute a Composer command in the web container.

### `exec`: Execute a shell command in a container (defaults to web container)

Value: string providing the command to run. Commands requiring user interaction are not supported. You can also add a “service” key to the command, specifying to run it on the `db` container or any other container you use.

Example: _Use Drush to clear the Drupal cache and get a user login link after database import_.

```yaml
hooks:
  post-import-db:
    - exec: drush cr
    - exec: drush uli
```

Example: _Use wp-cli to replace the production URL with development URL in a WordPress project’s database_.

```yaml
hooks:
  post-import-db:
    - exec: wp search-replace https://www.myproductionsite.com http://mydevsite.ddev.site
```

Example: _Add an extra database before `import-db`, executing in `db` container_.

```yaml
hooks:
  pre-import-db:
    - exec: mysql -uroot -proot -e "CREATE DATABASE IF NOT EXISTS some_new_database;"
      service: db

```

Example: _Add the common `ll` alias into the `web` container’s `.bashrc` file_.

```yaml
hooks:
  post-start:
  - exec: sudo echo alias ll=\"ls -lhA\" >> ~/.bashrc
```

!!!tip
    This could be done more efficiently via `.ddev/web-build/Dockerfile` as explained in [Customizing Images](../extend/customizing-images.md).

Advanced usages may require running commands directly with explicit arguments. This approach is useful when Bash interpretation is not required (no environment variables, no redirection, etc.).

```yaml
hooks:
  post-start:
  - exec:
    exec_raw: [ls, -lR, /var/www/html]
```

### `exec-host`: Execute a shell command on the host system

Value: string providing the command to run. Commands requiring user interaction are not supported.

```yaml
hooks:
  pre-start:
    - exec-host: "command to run"
```

### `composer`: Execute a Composer command in the web container

Value: string providing the Composer command to run.

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
    # Generate a one-time login link for the admin account
    - exec: "drush uli"
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
  post-start:
    # Install Composer dependencies from the web container
    - composer: install
    # Install Drupal after start if not installed already
    - exec: "(drush status bootstrap | grep -q Successful) || drush site-install -y --db-url=mysql://db:db@db/db"
    # Generate a one-time login link for the admin account
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
