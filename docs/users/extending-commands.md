<h1>Extending ddev Commands<h1>

Certain ddev commands provide hooks to run tasks before or after the main command executes. These tasks can be defined in the config.yml for your site, and allow for you to automate setup tasks specific to your site. To define command tasks in your configuration, specify the desired command hook as a subfield to `extend-commands`, then provide a list of tasks to run.

Example:

```
extend-commands:
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

- `exec`: Execute a shell command in the web service container.

Value: string providing the command to run. Commands requiring user interaction are not supported.

Example:

_Use drush to clear the drupal cache after database import_

```
extend-commands:
  post-import-db:
    - exec: "drush cc all"
```

- `exec-host`: Execute a shell command on your system.

Value: string providing the command to run. Commands requiring user interaction are not supported.

Example:

_Run a git pull before starting the site_

```
extend-commands:
  pre-start:
    - exec: "git pull origin master"
```

- `import-db`: Import the database of an existing site to the local environment.

Values:
- `src`: Required. Provide the path to a sql dump in .sql or tar/tar.gz/tgz/zip format
- `extract-path`: Optional. If provided asset is an archive, provide the path to extract within the archive.

Example:

_Import a database after environment starts_

```
extend-commands:
  post-start:
    - import-db:
        src: "~/Downloads/site-archive.tar.gz"
        extract-path: "data.sql"
```

- `import-files`: Import the uploaded files directory of an existing site to the default public
upload directory of your application.

Values:
- `src`: Required. Provide the path to a sql dump in .sql or tar/tar.gz/tgz/zip format
- `extract-path`: Optional. If provided asset is an archive, provide the path to extract within the archive.

Example:

_Import files after importing database_

```
extend-commands:
  post-import-db:
    - import-files:
        src: "~/Downloads/site-archive.tar.gz"
        extract-path: "docroot/sites/default/files"
```

## Full Example

The following example would import database and files from a full site archive, and clear the cache for a drupal site when "ddev start" is run.

```
extend-commands:
  post-start:
    - import-db:
        src: "~/Downloads/site-archive.tar.gz"
        extract-path: "data.sql"
    - import-files:
        src: "~/Downloads/site-archive.tar.gz"
        extract-path: "docroot/sites/default/files"
    - exec: "drush cc all"
```
