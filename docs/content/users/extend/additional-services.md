---
search:
  boost: 2
---

# Additional Service Configurations & Add-ons

DDEV projects can be extended to provide additional add-ons, including services. You can define these add-ons using `docker-compose.*.yaml` files in the project’s `.ddev` directory.

Anyone can create their own services with a `.ddev/docker-compose.*.yaml` file, and a growing number of popular services are supported and tested, and can be installed using the [`ddev add-on get`](../usage/commands.md#add-on-get) command.

Use `ddev add-on list` to see available add-ons. To see all possible add-ons (not necessarily supported or tested), use `ddev add-on list --all`.

For example,

```
→  ddev add-on list
┌──────────────────────────────────────┬────────────────────────────────────────────────────┐
│ ADD-ON                               │ DESCRIPTION                                        │
├──────────────────────────────────────┼────────────────────────────────────────────────────┤
│ ddev/ddev-adminer                    │ Adminer service for DDEV*                          │
├──────────────────────────────────────┼────────────────────────────────────────────────────┤
│ ddev/ddev-beanstalkd                 │ Beanstalkd for DDEV*                               │
├──────────────────────────────────────┼────────────────────────────────────────────────────┤
│ ddev/ddev-browsersync                │ Auto-refresh HTTPS page on changes with DDEV*      │
├──────────────────────────────────────┼────────────────────────────────────────────────────┤
│ ddev/ddev-cron                       │ Schedule commands to execute within DDEV*          │
├──────────────────────────────────────┼────────────────────────────────────────────────────┤
│ ddev/ddev-drupal-contrib             │ DDEV integration for developing Drupal contrib     │
│                                      │ projects*                                          │
├──────────────────────────────────────┼────────────────────────────────────────────────────┤
│ ddev/ddev-drupal-solr                │ Drupal Apache Solr installation for DDEV (please   │
│                                      │ consider ddev/ddev-solr first)*                    │
├──────────────────────────────────────┼────────────────────────────────────────────────────┤
│ ddev/ddev-elasticsearch              │ Elasticsearch add-on for DDEV*                     │
├──────────────────────────────────────┼────────────────────────────────────────────────────┤
│ ddev/ddev-ioncube                    │ IonCube loaders for DDEV*                          │
├──────────────────────────────────────┼────────────────────────────────────────────────────┤
│ ddev/ddev-memcached                  │ Install Memcached as an extra service in DDEV*     │
├──────────────────────────────────────┼────────────────────────────────────────────────────┤
│ ddev/ddev-minio                      │ MinIO addon for DDEV*                              │
├──────────────────────────────────────┼────────────────────────────────────────────────────┤
│ ddev/ddev-mongo                      │ MongoDB add-on for DDEV*                           │
├──────────────────────────────────────┼────────────────────────────────────────────────────┤
│ ddev/ddev-opensearch                 │ OpenSearch add-on for DDEV*                        │
├──────────────────────────────────────┼────────────────────────────────────────────────────┤
│ ddev/ddev-pdfreactor                 │ PDFreactor service for DDEV*                       │
├──────────────────────────────────────┼────────────────────────────────────────────────────┤
│ ddev/ddev-phpmyadmin                 │ phpMyAdmin Add-on For DDEV*                        │
├──────────────────────────────────────┼────────────────────────────────────────────────────┤
│ ddev/ddev-platformsh                 │ Add integration with Platform.sh hosting service*  │
├──────────────────────────────────────┼────────────────────────────────────────────────────┤
│ ddev/ddev-proxy-support              │ Support HTTP/HTTPS proxies with DDEV*              │
├──────────────────────────────────────┼────────────────────────────────────────────────────┤
│ ddev/ddev-redis                      │ Redis service for DDEV*                            │
├──────────────────────────────────────┼────────────────────────────────────────────────────┤
│ ddev/ddev-redis-7                    │ Redis 7 service for DDEV*                          │
├──────────────────────────────────────┼────────────────────────────────────────────────────┤
│ ddev/ddev-redis-commander            │ Redis Commander for use with DDEV Redis service*   │
├──────────────────────────────────────┼────────────────────────────────────────────────────┤
│ ddev/ddev-selenium-standalone-chrome │ A DDEV service for running standalone Chrome*      │
├──────────────────────────────────────┼────────────────────────────────────────────────────┤
│ ddev/ddev-solr                       │ Solr service for DDEV*                             │
├──────────────────────────────────────┼────────────────────────────────────────────────────┤
│ ddev/ddev-sqlsrv                     │ MS SQL server add-on for DDEV*                     │
├──────────────────────────────────────┼────────────────────────────────────────────────────┤
│ ddev/ddev-typo3-solr                 │ Use Apache Solr (standalone) in your DDEV project  │
│                                      │ *                                                  │
├──────────────────────────────────────┼────────────────────────────────────────────────────┤
│ ddev/ddev-varnish                    │ Varnish reverse proxy add-on for DDEV*             │
├──────────────────────────────────────┼────────────────────────────────────────────────────┤
│ ddev/ddev-xhgui                      │ XHGui service for a DDEV project*                  │
└──────────────────────────────────────┴────────────────────────────────────────────────────┘
25 repositories found. Add-ons marked with '*' are officially maintained DDEV add-ons.
```

!!!tip
    If you need a service not provided here, see [Defining an Additional Service with Docker Compose](custom-compose-files.md).

Officially-supported add-ons:

* [Adminer](https://github.com/ddev/ddev-adminer): `ddev add-on get ddev/ddev-adminer`.
* [Apache Solr for Drupal](https://github.com/ddev/ddev-drupal-solr): `ddev add-on get ddev/ddev-drupal-solr`.
* [Beanstalkd](https://github.com/ddev/ddev-beanstalkd): `ddev add-on get ddev/ddev-beanstalkd`.
* [Browsersync](https://github.com/ddev/ddev-browsersync): `ddev add-on get ddev/ddev-browsersync`.
* [cron](https://github.com/ddev/ddev-cron): `ddev add-on get ddev/ddev-cron`.
* [Elasticsearch](https://github.com/ddev/ddev-elasticsearch): `ddev add-on get ddev/ddev-elasticsearch`.
* [Memcached](https://github.com/ddev/ddev-memcached): `ddev add-on get ddev/ddev-memcached`.
* [MongoDB](https://github.com/ddev/ddev-mongo): `ddev add-on get ddev/ddev-mongo`.
* [PDFreactor](https://github.com/ddev/ddev-pdfreactor): `ddev add-on get ddev/ddev-pdfreactor`
* [Platform.sh](https://github.com/ddev/ddev-platformsh): `ddev add-on get ddev/ddev-platformsh`
* [Proxy Support](https://github.com/ddev/ddev-proxy-support): `ddev add-on get ddev/ddev-proxy-support`.
* [Redis Commander](https://github.com/ddev/ddev-redis-commander): `ddev add-on get ddev/ddev-redis-commander`.
* [Redis](https://github.com/ddev/ddev-redis): `ddev add-on get ddev/ddev-redis`.
* [Selenium Standalone Chrome](https://github.com/ddev/ddev-selenium-standalone-chrome): `ddev add-on get ddev/ddev-selenium-standalone-chrome`.
* [Varnish](https://github.com/ddev/ddev-varnish): `ddev add-on get ddev/ddev-varnish`.

## Managing Installed Add-Ons

Installed add-ons are versioned and can be viewed by running `ddev add-on list --installed`.

You can update an add-on by running `ddev add-on get <addonname>`, or remove it by running `ddev add-on remove <addonname>`.

## Adding Custom Configuration to an Add-on

Sometimes it's necessary to add custom configuration to an add-on. For example, in [`ddev-redis`](https://github.com/ddev/ddev-redis) the `image` is set to `image: redis:${REDIS_TAG:-6-bullseye}` (it is important to have the default value for `$REDIS_TAG`). If you wanted to change this to use `7-bookworm` instead, you would have two choices:

1. With DDEV v1.23.5+, the override can be done using a special `.ddev/.env.*` file, where `*` can be anything, in this case `.ddev/.env.redis`:

    ```bash
    ddev dotenv set .ddev/.env.redis --redis-tag 7-bookworm
    ddev restart
    ```

    This will set the variable `REDIS_TAG="7-bookworm"`, which is written to `.ddev/.env.redis` and then used on `ddev start`.

    Variables in the `.ddev/.env.*` files are passed to Docker service containers with the same name (e.g. `redis`). If this is not desired or the service does not exist, use a different filename, for example, `.ddev/.env.redis-build`, so that it is only expanded during the parsing of `.ddev/docker-compose.*.yaml` files.

    !!!tip "Give the add-on variables a unique name"
        The variable name should be unique enough not to conflict with other variables from different `.ddev/.env.*` files, for example, if the add-on name is `foo` and you want to have a variable `BAR=TEST`, include the add-on name in the variable name, i.e. `--foo-bar TEST`, which will be converted to `FOO_BAR="TEST"` in the `.ddev/.env.foo` file.

2. Or add a `.ddev/docker-compose.redis_extra.yaml` with the contents:

    ```yaml
    services:
      redis:
        image: redis:7-bookworm
    ```

Using the options here allow you to update to future versions of `ddev-redis` without losing your configuration, and without the upgrade being blocked because you removed the `#ddev-generated` line from the upstream `docker-compose.redis.yaml`.

!!!note
    Changing the image version to a different version will not necessarily work for any add-on or additional service. In this case, adjusting the Redis version often works.

## Creating an Additional Service for `ddev add-on`

Anyone can create an add-on for `ddev add-on`. See [this screencast](https://www.youtube.com/watch?v=fPVGpKGr0f4) and instructions in [`ddev-addon-template`](https://github.com/ddev/ddev-addon-template):

1. Click “Use this template” on [`ddev-addon-template`](https://github.com/ddev/ddev-addon-template).
2. Create a new repository.
3. Test it and preferably make sure it has valid tests in `tests.bats`.
4. When it’s working and tested, create a release.
5. Add the `ddev-get` label and a good short description to the GitHub repository.
6. When you’re ready for the add-on to become official, open an issue in the [DDEV issue queue](https://github.com/ddev/ddev/issues/new) requesting upgrade to official. You’ll be expected to maintain it, and subscribe to all activity and be responsive to questions.

### Sections and Features of ddev-get Add-On `install.yaml`

The `install.yaml` is a simple YAML file with a few main sections:

* `name`: The add-on name. This will be used in the various `ddev add-on` commands. If the repository name is `ddev-solr`, for example, a good `name` will be `solr`.
* `pre_install_actions`: an array of Bash statements or scripts to be executed before `project_files` are installed. The actions are executed in the context of the target project’s root directory. With DDEV v1.23.5+, variables from the `.ddev/.env.name` file, where `name` is the name of the add-on, are available in this context.
* `project_files`: an array of files or directories to be copied from the add-on into the target project’s `.ddev` directory.
* `global_files`: is an array of files or directories to be copied from the add-on into the target system’s global `.ddev` directory (`~/.ddev/`).
* `ddev_version_constraint` (available with DDEV v1.23.4+): a [version constraint](https://github.com/Masterminds/semver#checking-version-constraints) for DDEV that will be validated against the running DDEV executable and prevent add-on from being installed if it doesn't validate. For example:

    ```yaml
    ddev_version_constraint: '>= v1.23.4'
    ```

* `dependencies`: an array of add-ons that this add-on depends on.
* `post_install_actions`: an array of Bash statements or scripts to be executed after `project_files` and `global_files` are installed. The actions are executed in the context of the target project’s `.ddev` directory. With DDEV v1.23.5+, variables from the `.ddev/.env.name` file, where `name` is the name of the add-on, are available in this context.
* `removal_actions`: an array of Bash statements or scripts to be executed when the add-on is being removed with `ddev add-on remove`. The actions are executed in the context of the target project’s `.ddev` directory.
* `yaml_read_files`: a map of `name: file` of YAML files to be read from the target project’s root directory. The contents of these YAML files may be used as templated actions within `pre_install_actions` and `post_install_actions`.

In any stanza of `pre_install_actions` and `post_install_actions` you can:

* Use `#ddev-description:<some description of what stanza is doing>` to instruct DDEV to output a description of the action it's taking.

You can see a simple `install.yaml` in [`ddev-addon-template`’s `install.yaml`](https://github.com/ddev/ddev-addon-template/blob/main/install.yaml).

### Environment Variable Replacements

Simple environment variables will be replaced in `install.yaml` as part of filenames. This can include environment variables in the context where DDEV run, as well as the standard [environment variables](custom-commands.md#environment-variables-provided) provided to custom host commands, like `DDEV_APPROOT`, `DDEV_DOCROOT`, etc. For example, if a file in `project_files` is listed as `somefile.${DDEV_PROJECT}.txt` with a project named `d10`, the file named `somefile.d10.txt` will be copied from the add-on into the project.

### Template Action Replacements (Advanced)

A number of additional replacements can be made using Go template replacement techniques, using the format `{{ .some-gotemplate-action }}`. These are mostly for use of YAML information pulled into `yaml_read_files`. A map of values from each YAML file is placed in a map headed by the name of the YAML file. For example, if a YAML file named `example_yaml.yaml`:

```yaml
value1: xxx
```

is referenced using

```yaml
yaml_read_files:
  example: example_yaml.yaml
```

then `value1` can be used throughout the `install.yaml` as `{{ example.value1 }}` and it will be replaced with the value `xxx`.

More exotic template-based replacements can be seen in an advanced test [example](https://github.com/ddev/ddev/blob/main/cmd/ddev/cmd/testdata/TestCmdAddonComplex/recipe/install.yaml).

Go templating resources:

* [Official Go template docs](https://pkg.go.dev/text/template)
* [Lots of intro to Golang templates](https://www.google.com/search?q=golang+templates+intro&oq=golang+templates+intro&aqs=chrome..69i57j0i546l4.3161j0j4&sourceid=chrome&ie=UTF-8)
* [masterminds/sprig](http://masterminds.github.io/sprig/) extra functions.

## Additional services in ddev-contrib

Most commonly-used services have been migrated from the [ddev-contrib](https://github.com/ddev/ddev-contrib) repository to individual, tested, supported add-on repositories, but the repository still has a wealth of additional examples and instructions, but it's mostly obsolete.

* **Old PHP Versions to Run Old Sites**: See [Old PHP Versions](https://github.com/ddev/ddev-contrib/blob/master/docker-compose-services/old_php)
* **RabbitMQ**: See [RabbitMQ](https://github.com/ddev/ddev-contrib/blob/master/docker-compose-services/rabbitmq)

While we welcome requests to integrate other services at [ddev-contrib](https://github.com/ddev/ddev-contrib), we encourage creating a supported add-on that’s more beneficial to the community.
