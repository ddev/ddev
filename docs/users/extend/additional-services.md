<h1> Additional Service Configurations for ddev</h1>

ddev projects can be extended to provide additional services. This is achieved by adding docker-compose files to a project's .ddev directory that defines the added service(s). This page provides configurations for services that are ready to be added to your project with minimal setup.

If you need a service not provided here, see [Defining an additional service with Docker Compose](custom-compose-files.md)

## Apache Solr
This recipe adds an Apache Solr container to a project. It will set up a solr core with the solr configuration you define.

**Installation:**

- Copy [docker-compose.solr.yaml](https://github.com/drud/ddev/tree/master/pkg/servicetest/testdata/services/docker-compose.solr.yaml) to the .ddev folder for your project.
- You can change the Solr version by changing the `image` value in docker-compose.solr.yaml, for example: `image: solr:6.6`. The most obvious official solr image tags are at [hub.docker.com](https://hub.docker.com/_/solr/).
- Create the folder path .ddev/solr/conf.
- Copy the Solr configuration files for your project to `.ddev/solr/conf`. e.g. if using [Drupal Search API Solr](https://www.drupal.org/project/search_api_solr), you would copy the `web/modules/contrib/search_api_solr/solr-conf-templates/6.x/ `contents from the module code base into `.ddev/solr/conf`.
- Ensure that the configuration files are present before running `ddev start`.
- Add the following post-start hook to your config.yaml; this turns on ping for the "dev" core. The curl command can also be done as a one-time configuration from within the web container.
```
hooks:
  post-start:
  - exec: curl --fail -s 'http://solr:8983/solr/dev/admin/ping?action=enable'
```

**Drupal8-specific extra steps:** 
- Enable the Search API Solr Search Defaults module and edit the server settings at `/admin/config/search/search-api/server/default_solr_server/edit`.
- Change the "Solr core" field from the default "d8" to "dev" and under **Advanced Server Configuration** change the _solr.install.dir_ setting to `/opt/solr`.
- Go to the view tab and download the updated config.zip file.
- Stop the project with `ddev stop`.
- Remove the original configuration files from `.ddev/solr/conf` and copy in the updated files extracted from config.zip.
- In order for changes to take effect you must remove the Solr volume by running `docker volume rm ddev-PROJECT-NAME_solrdata` e.g. if your project is called "myproject" then you would run `docker volume rm ddev-myproject_solrdata`.
- Now you can start the project `ddev start`.

**Updating Apache Solr configuration**

- Run `ddev stop` to remove your application's containers (note: if you do not use the [destructive option](cli-usage#removing-projects-from-your-collection-known-to-ddev), the index will be untouched).
- copy the new solr configuration files for your project to .ddev/solr/conf as described in **Installation**, above.
- Run `ddev start` to rebuild and restart the containers.
- An excellent way to automate the updating of the solr config uses a [solr-init.sh](https://github.com/drud/ddev/pull/1645#issuecomment-503722974) script mounted into the solr container's `/docker-entrypoint-initdb.d/solr-init.sh`.

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
