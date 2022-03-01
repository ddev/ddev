## Additional Service Configurations for ddev

DDEV-Local projects can be extended to provide additional add-ons, including services. This is achieved by adding docker-compose files to a project's .ddev directory that defines the added add-on(s).

If you need a service not provided here, see [Defining an additional service with Docker Compose](custom-compose-files.md)

Although anyone can create their own services with a `.ddev/docker-compose.*.yaml` file, a growing number of services are supported and tested and can be installed with the `ddev get` command starting with DDEV v1.19.0+.

You can see available supported and tested add-ons with the command `ddev get --list`. To see all possible add-ons (not necessarily supported or tested), use `ddev get --list --all`.

For example,

```bash
$ ddev get --list
┌─────────────────────────┬───────────────────────────────────────────────┐
│ *drud/ddev-beanstalkd   │ beanstalkd for ddev                           │
├─────────────────────────┼───────────────────────────────────────────────┤
│ *drud/ddev-memcached    │ Install memcached as an extra service in ddev │
├─────────────────────────┼───────────────────────────────────────────────┤
│ *drud/ddev-drupal9-solr │ Drupal 9 Apache Solr installation for DDEV    │
└─────────────────────────┴───────────────────────────────────────────────┘
Add-ons marked with '*' are official, maintained ddev add-ons.
```

Here are some of the add-ons that are currently supported:

* [Apache Solr for Drupal 9](https://github.com/drud/ddev-drupal9-solr): `ddev get drud/ddev-drupal9-solr`.
* [Memcached](https://github.com/drud/ddev-memcached): `ddev get drud/ddev-memcached`.
* [Beanstalkd](https://github.com/drud/ddev-beanstalkd): `ddev get drud/ddev-beanstalkd`.

## Additional services in ddev-contrib (MongoDB, Elasticsearch, etc)

Commonly used services will be migrated from the ddev-contrib repository to individual, tested, supported repositories, but
 [ddev-contrib](https://github.com/drud/ddev-contrib) repository has a wealth of additional examples and instructions:

* **ElasticHQ**:See [ElasticHQ](https://github.com/drud/ddev-contrib/blob/master/docker-compose-services/elastichq).
* **Elasticsearch**: See [Elasticsearch](https://github.com/drud/ddev-contrib/blob/master/docker-compose-services/elasticsearch).
* **Headless Chrome**: See [Headless Chrome for Behat Testing](https://github.com/drud/ddev-contrib/blob/master/docker-compose-services/headless-chrome)
* **MongoDB**: See [MongoDB](https://github.com/drud/ddev-contrib/blob/master/docker-compose-services/mongodb).
* **Old PHP Versions to Run Old Sites**: See [Old PHP Versions](https://github.com/drud/ddev-contrib/blob/master/docker-compose-services/old_php)
* **RabbitMQ**: See [RabbitMQ](https://github.com/drud/ddev-contrib/blob/master/docker-compose-services/rabbitmq)
* **Redis Commander**: See [redis commander](https://github.com/drud/ddev-contrib/blob/master/docker-compose-services/redis-commander)
* **TYPO3 Solr Integration**: See [TYPO3 Solr](https://github.com/drud/ddev-contrib/blob/master/docker-compose-services/typo3-solr)

Your PRs to integrate other services are welcome at [ddev-contrib](https://github.com/drud/ddev-contrib).
