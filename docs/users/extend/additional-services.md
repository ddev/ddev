<h1> Additional Service Configurations for ddev</h1>

ddev projects can be extended to provide additional services. This is achieved by adding docker-compose files to a project's .ddev directory that defines the added service(s). This page provides configurations for services that are ready to be added to your project with minimal setup.

If you need a service not provided here, see [Defining an additional service with Docker Compose](custom-compose-files.md)

## Apache Solr
This recipe adds an Apache Solr container to a project. It will setup a solr core with the solr configuration you define.

**Installation:**

- Copy [docker-compose.solr.yaml](https://github.com/drud/ddev/tree/master/pkg/servicetest/testdata/services/docker-compose.solr.yaml) to the .ddev folder for your project.
- You can change the Solr version by changing the `image` value in docker-compose.solr.yaml, for example: `image: solr:6.6`. The most obvious official solr image tags are at [hub.docker.com](https://hub.docker.com/_/solr/).
- Create the folder path .ddev/solr/conf.
- Copy the solr configuration files for your project to .ddev/solr/conf. _e.g., using Drupal Search API Solr, you would copy the solr-conf/5.x/ contents from the module code base into .ddev/solr/conf._
- Ensure that the configuration files are present before running `ddev start`.

**Interacting with Apache Solr**

- The Solr admin interface will be accessible at: `http://<projectname>.ddev.site:8983/solr/` For example, if the project is named "myproject" the hostname will be: `http://myproject.ddev.site:8983/solr/`
- To access the Solr container from the web container use: `http://solr:8983/solr/`
- A Solr core is automatically created with the name "dev"; it can be accessed at the URL: http://solr:8983/solr/dev

## Memcached
This recipe adds a Memcached 1.5 container to a project. The default configuration allocates 128 MB of RAM for the Memcached instance; to change that or other command line arguments, edit the `command` array within the docker-compose file.

**Installation:**

- Copy [docker-compose.memcached.yaml](https://github.com/drud/ddev/tree/master/pkg/servicetest/testdata/services/docker-compose.memcached.yaml) to the .ddev folder for your project.
- Run `ddev start`.

**Interacting with Memcached**

- The Memcached instance will listen on TCP port 11211 (the Memcached default).
- Configure your application to access Memcached on the host:port `memcached:11211`.
- To reach the Memcached admin interface, run `ddev ssh` to connect to the web container, then use `nc` or `telnet` to connect to the Memcached container on port 11211, i.e. `nc memcached 11211`. You can then run commands such as `stats` to see usage information.

## Beanstalk (Work Queue)
This recipe adds a [Beanstalk](https://beanstalkd.github.io/) container to a project.

**Installation:**

- Copy [docker-compose.beanstalk.yaml](https://github.com/drud/ddev/tree/master/pkg/servicetest/testdata/services/docker-compose.beanstalkd.yaml) to the .ddev folder for your project.
- Run `ddev start`.

**Interacting with the Beanstalk Queue**

- The Beanstalk instance will listen on TCP port 11300 (the beanstalkd default).
- Configure your application to access Beanstalk on the host:port `beanstalk:11300`.
