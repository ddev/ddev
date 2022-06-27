# Additional Service Configurations and Add-ons for ddev

DDEV-Local projects can be extended to provide additional add-ons, including services. This is achieved by adding docker-compose files to a project's `.ddev` directory that defines the added add-on(s).

If you need a service not provided here, see [Defining an additional service with Docker Compose](custom-compose-files.md)

Although anyone can create their own services with a `.ddev/docker-compose.*.yaml` file, a growing number of services are supported and tested and can be installed with the `ddev get` command starting with DDEV v1.19.0+.

You can see available supported and tested add-ons with the command `ddev get --list`. To see all possible add-ons (not necessarily supported or tested), use `ddev get --list --all`.

For example,

```
ddev get --list
┌───────────────────────────┬──────────────────────────────────────────────────┐
│ ADD-ON                    │ DESCRIPTION                                      │
├───────────────────────────┼──────────────────────────────────────────────────┤
│ drud/ddev-elasticsearch   │ Elasticsearch add-on for DDEV*                   │
├───────────────────────────┼──────────────────────────────────────────────────┤
│ drud/ddev-varnish         │ Varnish reverse proxy add-on for ddev*           │
├───────────────────────────┼──────────────────────────────────────────────────┤
│ drud/ddev-redis           │ redis service for ddev*                          │
├───────────────────────────┼──────────────────────────────────────────────────┤
│ drud/ddev-beanstalkd      │ beanstalkd for ddev*                             │
├───────────────────────────┼──────────────────────────────────────────────────┤
│ drud/ddev-redis-commander │ Redis commander for use with ddev redis service* │
├───────────────────────────┼──────────────────────────────────────────────────┤
│ drud/ddev-mongo           │ mongodb addon for ddev*                          │
├───────────────────────────┼──────────────────────────────────────────────────┤
│ drud/ddev-drupal9-solr    │ Drupal 9 Apache Solr installation for DDEV*      │
├───────────────────────────┼──────────────────────────────────────────────────┤
│ drud/ddev-pdfreactor      │ PDFreactor service for ddev*                     │
├───────────────────────────┼──────────────────────────────────────────────────┤
│ drud/ddev-memcached       │ Install memcached as an extra service in ddev*   │
└───────────────────────────┴──────────────────────────────────────────────────┘
```

Here are some of the add-ons that are officially supported:

* [Redis](https://github.com/drud/ddev-redis): `ddev get drud/ddev-redis`.
* [Redis Commander](https://github.com/drud/ddev-redis-commander): `ddev get drud/ddev-redis-commander`.
* [Elasticsearch](https://github.com/drud/ddev-elasticsearch): `ddev get drud/ddev-elasticsearch`.
* [Apache Solr for Drupal 9](https://github.com/drud/ddev-drupal9-solr): `ddev get drud/ddev-drupal9-solr`.
* [Memcached](https://github.com/drud/ddev-memcached): `ddev get drud/ddev-memcached`.
* [Varnish](https://github.com/drud/ddev-varnish): `ddev get drud/ddev-varnish`.
* [Mongo](https://github.com/drud/ddev-mongo): `ddev get drud/ddev-mongo`.
* [PDFReactor](https://github.com/drud/ddev-pdfreactor): `ddev get drud/ddev-pdfreactor`
* [Beanstalkd](https://github.com/drud/ddev-beanstalkd): `ddev get drud/ddev-beanstalkd`.

## Creating an additional service for `ddev get`

Anyone can create an add-on for `ddev get` (see [screencast](https://www.youtube.com/watch?v=fPVGpKGr0f4) and instructions in [`ddev-addon-template`](https://github.com/drud/ddev-addon-template)):

1. Click "Use this template" on [`ddev-addon-template`](https://github.com/drud/ddev-addon-template).
2. Create a new repository
3. Test it and preferably make sure it has valid tests in `tests.bats`.
4. When it's working and tested, create a release.
5. Add the label `ddev-get` and a good short description to the repository on GitHub.
6. When you're ready for the add-on to become official, open an issue in the [DDEV issue queue](https://github.com/drud/ddev/issues/new) requesting upgrade to official. You'll be expected to maintain it of course, and subscribe to all activity and be responsive to questions.

### Sections and features of ddev-get add-on install.yaml

The install.yaml is a simple yaml file with a few main sections:

* `pre_install_actions` is an array of bash statements or scripts that will be executed before `project_files` are installed. The actions are executed in the context of the target project's root directory.
* `project_files` is an array of files or directories to be copied from the add-on into the target project's .ddev directory.
* `global_files` is an array of files or directories to be copied from the add-on into the target system's global .ddev directory (`~/.ddev/`).
* `post_install_actions` is an array of bash statements or scripts that will be executed after `project_files` and `global_files` are installed. The actions are executed in the context of the target project's root directory.
* `yaml_read_files` is a map of `name: file` of yaml files that will be read from the target project's root directory. The contents of these yaml files may be used as templated actions within `pre_install_actions` and `post_install_actions`.

You can see a simple install.yaml in [ddev-addon-template's install.yaml](https://github.com/drud/ddev-addon-template/blob/main/install.yaml).

### Environment variable replacements

Simple environment variables will be replaced in `install.yaml` as part of filenames. This can include environment variables in the context of where ddev is being run, as well as the standard [environment variables](custom-commands.md#environment-variables-provided) provided to custom host commands, like `DDEV_APPROOT`, `DDEV_DOCROOT`, etc. For example, if a file in `project_files` is listed as `somefile.${DDEV_PROJECT}.txt` with a project named `d10`, the file named `somefile.d10.txt` will be copied from the add-on into the project.

### Template action replacements (advanced)

A number of additional replacements can be made using go template replacement techniques, using the format `{{ .some-gotemplate-action }}`. These are mostly for use of yaml information pulled into `yaml_read_files`. A map of values from each yaml file is placed in a map headed by the name of the yaml file. For example, if a yaml file named `example_yaml.yaml:

```yaml
value1: xxx
```

is referenced using

```yaml
yaml_read_files: 
 example: example_yaml.yaml
```

then `value1` can be used throughout the `install.yaml` as `{{ example.value1 }}` and it will be replaced with the value `xxx`.

More exotic template-based replacements can be seen in advanced test [example](../../../../../cmd/ddev/cmd/testdata/TestCmdGetComplex/recipe/install.yaml).

## Additional services in ddev-contrib (MongoDB, Elasticsearch, etc)

Commonly used services will be migrated from the ddev-contrib repository to individual, tested, supported repositories, but
 [ddev-contrib](https://github.com/drud/ddev-contrib) repository has a wealth of additional examples and instructions:

* **ElasticHQ**:See [ElasticHQ](https://github.com/drud/ddev-contrib/blob/master/docker-compose-services/elastichq).
* **Headless Chrome**: See [Headless Chrome for Behat Testing](https://github.com/drud/ddev-contrib/blob/master/docker-compose-services/headless-chrome)
* **MongoDB**: See [MongoDB](https://github.com/drud/ddev-contrib/blob/master/docker-compose-services/mongodb).
* **Old PHP Versions to Run Old Sites**: See [Old PHP Versions](https://github.com/drud/ddev-contrib/blob/master/docker-compose-services/old_php)
* **RabbitMQ**: See [RabbitMQ](https://github.com/drud/ddev-contrib/blob/master/docker-compose-services/rabbitmq)
* **TYPO3 Solr Integration**: See [TYPO3 Solr](https://github.com/drud/ddev-contrib/blob/master/docker-compose-services/typo3-solr)

Your pull requests to integrate other services are welcome at [ddev-contrib](https://github.com/drud/ddev-contrib).
