<h1> Additional Service Configurations for ddev</h1>

ddev projects can be extended to provide additional services. This is achieved by adding docker-compose files to a project's .ddev directory that defines the added service(s). This page provides configurations for services that are ready to be added to your project with minimal setup.

If you need a service not provided here, see [Defining an additional service with Docker Compose](custom-compose-files.md)

## Apache Solr
This recipe adds an Apache Solr container to a project. It will set up a solr core named "dev" with the solr configuration you define.

**Installation:**

- Copy [docker-compose.solr.yaml](https://github.com/drud/ddev/tree/master/pkg/servicetest/testdata/services/docker-compose.solr.yaml) to the .ddev folder for your project.
- The recommended Solr version is: `image: solr:8`, from [hub.docker.com](https://hub.docker.com/_/solr/).
- Create the folder path .ddev/solr/conf.
- Copy/extract the Solr configuration files for your project into `.ddev/solr/conf`. 
- Ensure that the configuration files are present before running `ddev start`.

**Drupal8-specific extra steps:**

- `ddev start`
- Enable the Search API Solr Search Defaults module 
- Add a solr server at `https://<projectname>>.ddev.site/en/admin/config/search/search-api/add-server`.
  - Use the "standard" Solr connector
  - Use the "http" protocol
  - The "solr host" should be "solr" **NOT the default "localhost"**
  - The "solr core" should be named "dev" unless you customize the docker-compose.solr.yaml
  - Under "Advanced server configuration" set the "solr.install.dir" to `/opt/solr`
- Download the config.zip provided on /admin/config/search/search-api/server/dev
- Unzip the config.zip into .ddev/solr/conf. For example, `cd .ddev/solr/conf && unzip ~/Downloads/solr_8.x-config.zip`
- In order for changes to take effect you must stop the project, remove the Solr volume, and start it again.  So run `docker volume rm ddev-<projectname>_solrdata` if your project is called "myproject" then you would run `ddev stop && docker volume rm ddev-myproject_solrdata && ddev restart`. (If you have installed solr-configupdate.sh as described below, then you need only `ddev restart`)

**Updating Apache Solr configuration on an existing Solr core**

The default [solr-precreate script](https://github.com/docker-solr/docker-solr/blob/master/scripts/solr-precreate) provided in [docker-solr](https://github.com/docker-solr/docker-solr) and used in the `entrypoint` in docker-compose.solr.yaml does not have the capability to update core configuration after the core has been created. It just copies mounted config into the core, where it would otherwise live forever. However, a simple optional script executed on startup can re-copy config into place. Here's the technique:

- Copy [solr-configupdate.sh](https://github.com/drud/ddev/tree/master/pkg/servicetest/testdata/services/solr-configupdate.sh) to .ddev/solr. This simple script is mounted into the container and updates config from .ddev/solr/conf on `ddev restart`: `cd .ddev/solr && rm -rf solr-configupdate.sh && curl -O https://github.com/drud/ddev/tree/master/pkg/servicetest/testdata/services/solr-configupdate.sh && chmod +x solr-configupdate.sh`
- Make sure solr-configupdate.sh is executable: `chmod +x .ddev/solr/configupdate.sh`
- You can now copy/edit/update the solr configuration files for your project in .ddev/solr/conf and when you `ddev restart` the solr configuration will be live.

**Interacting with Apache Solr**

- The Solr admin interface will be accessible at: `http://<projectname>.ddev.site:8983/solr/` For example, if the project is named "_myproject_" the hostname will be: `http://myproject.ddev.site:8983/solr/`.
- To access the Solr container from the web container use: `http://solr:8983/solr/`
- A Solr core is automatically created with the name "dev"; it can be accessed (from inside the web container) at the URL: `http://solr:8983/solr/dev` or from the host at `http://<projectname>.ddev.site:8983/solr/#/~cores/dev`.

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

## Additional services in ddev-contrib (MongoDB, Blackfire, PostgresSQL, etc)

The [ddev-contrib](https://github.com/drud/ddev-contrib) repository has a wealth of additional examples and instructions:

* **MongoDB**: See [MongoDB](https://github.com/drud/ddev-contrib/blob/master/docker-compose-services/mongodb).
* **Blackfire.io for performance testing and profiling**: See [Blackfire.io](https://github.com/drud/ddev-contrib/blob/master/docker-compose-services/blackfire).
* **PostgresSQL**: See [PostgresSQL](https://github.com/drud/ddev-contrib/blob/master/docker-compose-services/postgres).
* **Elasticsearch**: See [Elasticsearch](https://github.com/drud/ddev-contrib/blob/master/docker-compose-services/elasticsearch).
* **Oracle MySQL**: See [MySQL](https://github.com/drud/ddev-contrib/blob/master/docker-compose-services/mysql).
