
# Additional Service Configurations & Add-ons

DDEV projects can be extended to provide additional add-ons, including services. You can define these add-ons using docker-compose files in the project’s `.ddev` directory.

Anyone can create their own services with a `.ddev/docker-compose.*.yaml` file, and as of DDEV v1.19.0 a growing number of popular services are supported and tested, and can be installed using the `ddev get` command.

Use `ddev get --list` to see available add-ons. To see all possible add-ons (not necessarily supported or tested), use `ddev get --list --all`.

For example,

```
ddev get --list
┌───────────────────────────┬──────────────────────────────────────────────────┐
│ ADD-ON                    │ DESCRIPTION                                      │
├───────────────────────────┼──────────────────────────────────────────────────┤
│ drud/ddev-elasticsearch   │ Elasticsearch add-on for DDEV*                   │
├───────────────────────────┼──────────────────────────────────────────────────┤
│ drud/ddev-varnish         │ Varnish reverse proxy add-on for DDEV*           │
├───────────────────────────┼──────────────────────────────────────────────────┤
│ drud/ddev-redis           │ redis service for DDEV*                          │
├───────────────────────────┼──────────────────────────────────────────────────┤
│ drud/ddev-beanstalkd      │ beanstalkd for DDEV*                             │
├───────────────────────────┼──────────────────────────────────────────────────┤
│ drud/ddev-redis-commander │ Redis commander for use with DDEV redis service* │
├───────────────────────────┼──────────────────────────────────────────────────┤
│ drud/ddev-mongo           │ mongodb addon for DDEV*                          │
├───────────────────────────┼──────────────────────────────────────────────────┤
│ drud/ddev-drupal9-solr    │ Drupal 9 Apache Solr installation for DDEV*      │
├───────────────────────────┼──────────────────────────────────────────────────┤
│ drud/ddev-pdfreactor      │ PDFreactor service for DDEV*                     │
├───────────────────────────┼──────────────────────────────────────────────────┤
│ drud/ddev-memcached       │ Install memcached as an extra service in DDEV*   │
└───────────────────────────┴──────────────────────────────────────────────────┘
```

!!!tip
    If you need a service not provided here, see [Defining an Additional Service with Docker Compose](custom-compose-files.md).

Some of the officially-supported add-ons:

* [Redis](https://github.com/drud/ddev-redis): `ddev get drud/ddev-redis`.
* [Redis Commander](https://github.com/drud/ddev-redis-commander): `ddev get drud/ddev-redis-commander`.
* [elasticsearch](https://github.com/drud/ddev-elasticsearch): `ddev get drud/ddev-elasticsearch`.
* [Apache Solr for Drupal 9](https://github.com/drud/ddev-drupal9-solr): `ddev get drud/ddev-drupal9-solr`.
* [Memcached](https://github.com/drud/ddev-memcached): `ddev get drud/ddev-memcached`.
* [Varnish](https://github.com/drud/ddev-varnish): `ddev get drud/ddev-varnish`.
* [MongoDB](https://github.com/drud/ddev-mongo): `ddev get drud/ddev-mongo`.
* [PDFreactor](https://github.com/drud/ddev-pdfreactor): `ddev get drud/ddev-pdfreactor`
* [Beanstalkd](https://github.com/drud/ddev-beanstalkd): `ddev get drud/ddev-beanstalkd`.

## Creating an Additional Service for `ddev get`

Anyone can create an add-on for `ddev get`. See [this screencast](https://www.youtube.com/watch?v=fPVGpKGr0f4) and instructions in [`ddev-addon-template`](https://github.com/drud/ddev-addon-template):

1. Click “Use this template” on [`ddev-addon-template`](https://github.com/drud/ddev-addon-template).
2. Create a new repository.
3. Test it and preferably make sure it has valid tests in `tests.bats`.
4. When it’s working and tested, create a release.
5. Add the `ddev-get` label and a good short description to the GitHub repository.
6. When you’re ready for the add-on to become official, open an issue in the [DDEV issue queue](https://github.com/drud/ddev/issues/new) requesting upgrade to official. You’ll be expected to maintain it, and subscribe to all activity and be responsive to questions.

### Sections and Features of ddev-get Add-On install.yaml

The `install.yaml` is a simple YAML file with a few main sections:

* `pre_install_actions`: an array of Bash statements or scripts to be executed before `project_files` are installed. The actions are executed in the context of the target project’s root directory.
* `project_files`: an array of files or directories to be copied from the add-on into the target project’s `.ddev` directory.
* `global_files`: is an array of files or directories to be copied from the add-on into the target system’s global `.ddev` directory (`~/.ddev/`).
* `post_install_actions`: an array of Bash statements or scripts to be executed after `project_files` and `global_files` are installed. The actions are executed in the context of the target project’s root directory.
* `yaml_read_files`: a map of `name: file` of YAML files to be read from the target project’s root directory. The contents of these YAML files may be used as templated actions within `pre_install_actions` and `post_install_actions`.

You can see a simple `install.yaml` in [ddev-addon-template’s install.yaml](https://github.com/drud/ddev-addon-template/blob/main/install.yaml).

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

More exotic template-based replacements can be seen in an advanced test [example](https://github.com/drud/ddev/blob/master/cmd/ddev/cmd/testdata/TestCmdGetComplex/recipe/install.yaml).

## Additional Services in ddev-contrib (MongoDB, elasticsearch, etc)

Commonly-used services will be migrated from the [ddev-contrib](https://github.com/drud/ddev-contrib) repository to individual, tested, supported repositories, but the repository already has a wealth of additional examples and instructions:

* **Headless Chrome**: See [Headless Chrome for Behat Testing](https://github.com/drud/ddev-contrib/blob/master/docker-compose-services/headless-chrome)
* **Old PHP Versions to Run Old Sites**: See [Old PHP Versions](https://github.com/drud/ddev-contrib/blob/master/docker-compose-services/old_php)
* **RabbitMQ**: See [RabbitMQ](https://github.com/drud/ddev-contrib/blob/master/docker-compose-services/rabbitmq)
* **TYPO3 Solr Integration**: See [TYPO3 Solr](https://github.com/drud/ddev-contrib/blob/master/docker-compose-services/typo3-solr)

Your pull requests to integrate other services are welcome at [ddev-contrib](https://github.com/drud/ddev-contrib).
