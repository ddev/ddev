## Additional Service Configurations for ddev

DDEV-Local projects can be extended to provide additional services. This is achieved by adding docker-compose files to a project's .ddev directory that defines the added service(s). This page provides configurations for services that are ready to be added to your project with minimal setup.

If you need a service not provided here, see [Defining an additional service with Docker Compose](custom-compose-files.md)

Although anyone can create their own services with a `docker-compose.*.yaml` file, a growing number of services are supported and tested and can be installed with the `ddev service get` command starting with DDEV v1.19.0-alpha3.

* [Apache Solr for Drupal 9](https://github.com/drud/ddev-drupal9-solr): `ddev service get drud/ddev-drupal9-solr`.
* [Memcached](https://github.com/drud/ddev-memcached): `ddev service get drud/ddev-memcached`.

### Beanstalk (Work Queue)

This recipe adds a [Beanstalk](https://beanstalkd.github.io/) container to a project.

#### Beanstalk Installation

* Copy [docker-compose.beanstalk.yaml](https://github.com/drud/ddev/tree/master/pkg/servicetest/testdata/TestServices/docker-compose.beanstalkd.yaml) to the .ddev folder for your project.
* Run `ddev start`.

#### Interacting with the Beanstalk Queue

* The Beanstalk instance will listen on TCP port 11300 (the beanstalkd default).
* Configure your application to access Beanstalk on the host:port `beanstalk:11300`.

## Additional services in ddev-contrib (MongoDB, PostgresSQL, etc)

The [ddev-contrib](https://github.com/drud/ddev-contrib) repository has a wealth of additional examples and instructions:

* **ElasticHQ**:See [ElasticHQ](https://github.com/drud/ddev-contrib/blob/master/docker-compose-services/elastichq).
* **Elasticsearch**: See [Elasticsearch](https://github.com/drud/ddev-contrib/blob/master/docker-compose-services/elasticsearch).
* **Headless Chrome**: See [Headless Chrome for Behat Testing](https://github.com/drud/ddev-contrib/blob/master/docker-compose-services/headless-chrome)
* **MongoDB**: See [MongoDB](https://github.com/drud/ddev-contrib/blob/master/docker-compose-services/mongodb).
* **Old PHP Versions to Run Old Sites**: See [Old PHP Versions](https://github.com/drud/ddev-contrib/blob/master/docker-compose-services/old_php)
* **PostgresSQL**: See [PostgresSQL](https://github.com/drud/ddev-contrib/blob/master/docker-compose-services/postgres).
* **RabbitMQ**: See [RabbitMQ](https://github.com/drud/ddev-contrib/blob/master/docker-compose-services/rabbitmq)
* **Redis**: See [redis](https://github.com/drud/ddev-contrib/blob/master/docker-compose-services/redis).
* **Redis Commander**: See [redis commander](https://github.com/drud/ddev-contrib/blob/master/docker-compose-services/redis-commander)
* **TYPO3 Solr Integration**: See [TYPO3 Solr](https://github.com/drud/ddev-contrib/blob/master/docker-compose-services/typo3-solr)

Your PRs to integrate other services are welcome at [ddev-contrib](https://github.com/drud/ddev-contrib).
