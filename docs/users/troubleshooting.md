<h1>Troubleshooting</h1>

Things might go wrong! Besides the suggestions on this page don't forget about [Stack Overflow](https://stackoverflow.com/tags/ddev) and [the ddev issue queue](https://github.com/drud/ddev/issues) and [other support options](https://ddev.readthedocs.io/en/stable/#support). And see [Docker troubleshooting suggstions](./docker_installation.md#troubleshooting).

<a name="unable-listen"></a>
## Webserver ports are already occupied by another webserver

ddev notifies you about port conflicts with this message:

```
Failed to start yoursite: Unable to listen on required ports, localhost port 80 is in use,
```

This means there is another webserver listening on the named port(s) and ddev cannot access the port.

To resolve this conflict, choose one of two methods:

1. Fix port conflicts by configuring your project to use different ports.
2. Fix port conflicts by stopping the competing application.

### Method 1: Fix port conflicts by configuring your project to use different ports

To configure a project to use non-conflicting ports, edit the project's .ddev/config.yaml to add entries like `router_http_port: 8000` and `router_https_port: 8443` depending on your needs. Then use `ddev start` again.

For example, if there was a port conflict with a local apache http on port 80 add the following to the to the config.yaml file.

```
router_http_port: 8000
```

Then run `ddev start`. This changes the project's http URL to http://yoursite.ddev.local:8000.


### Method 2: Fix port conflicts by stopping the competing application

Alternatively, stop the other application.

Probably the most common conflicting application is Apache running locally. It can often be stopped gracefully (but temporarily) with:

```
sudo apachectl stop
```

**Common tools that use port 80:**

Here are some of the other common processes that could be using port 80 and methods to stop them.

* MAMP (macOS): [Stop MAMP](http://documentation.mamp.info/en/MAMP-Mac/Preferences/Start-Stop/)
* Apache: Temporarily stop with `sudo apachectl stop`, permanent stop depends on your environment.
* nginx (macOS Homebrew): `sudo brew services stop nginx`
or `sudo launchctl stop homebrew.mxcl.nginx`
* nginx (Ubuntu): `sudo service nginx stop`
* apache (often named "httpd") (many environments): `sudo apachectl stop` or on Ubuntu `sudo service apache2 stop`
* vpnkit (macOS): You likely have a docker container bound to port 80, do you have containers up for Kalabox or another docker-based development environment? If so, stop the other environment.
* Kalabox: If you have previously used Kalabox try running `kbox poweroff`

To dig deeper, you can use a number of tools to find out what process is listening. On macOS and Linux, try the lsof tool:

```
$ sudo lsof -i :80 -sTCP:LISTEN
COMMAND  PID     USER   FD   TYPE DEVICE SIZE/OFF NODE NAME
nginx   1608 www-data   46u  IPv4  13913      0t0  TCP *:http (LISTEN)
nginx   5234     root   46u  IPv4  13913      0t0  TCP *:http (LISTEN)
```

The resulting output displays which command is running and its pid. Choose the appropriate method to stop the other server.

We welcome your [suggestions](https://github.com/drud/ddev/issues/new) based on other issues you've run into and your troubleshooting technique.

<a name="container-restarts"></a>
## DDEV-Local reports container restarts and does not arrive at "ready"

## Restarts of the database container

The most common cause of the database container not coming up is a damaged database, so the mariadb server daemon is unable to start. This is typically caused by an unexpected docker event like system shutdown or docker exit which doesn't give the db container time to clean up and close connections. See [issue](https://github.com/drud/ddev/issues/748). In general, the easiest fix is to destroy and reload the database from either a database dump or a ddev snapshot. Otherwise, that issue has more ambitious approaches that may be taken if you have neither. But the easiest approach is this, which *will destroy and then reload your project database*:

1. `ddev stop --remove-data --omit-snapshot`
2. mv .ddev .ddev.bak (renames the directory with config.yaml and docker-compose.yml and any custom nginx/php/mariadb config you may have added. Renaming it means .)
3. `ddev config`
4. `ddev start` 
5. `ddev import-db` or `ddev restore-snapshot <snapshot-name>` if you have a db to import or a snapshot to restore.

Another approach to destroying the database is to destroy the docker volume where it is =stored with `docker volume rm <projectname>-mariadb`

## "web service unhealthy" or "web service starting" or exited 

The most common cause of the web container being unhealthy is a user-defined .ddev/nginx-site.conf or .ddev/apache/apache-site.conf - Please rename these to <xxx_site.conf> during testing. To figure out what's wrong with it after you've identified that as the problem, use `ddev logs` and review the error.

Changes to .ddev/nginx-site.conf and .ddev/apache/apache-site.conf take effect only when you do a `ddev restart` or the equivalent.

## No input file specified (404) or Forbidden (403)

If you get a 404 with "No input file specified" (nginx) or a 403 with "Forbidden" (apache) when you visit your project it may mean that no index.php or index.html is being found in the docroot. This can result from:

* Missing index.php: There may not be an index.php or index.html in your project.
* Misconfigured docroot: If the docroot isn't where the webserver thinks it is, then the webserver won't find the index.php. Look at your .ddev/config.yaml to verify it has a docroot that will lead to the index.php. It should be a relative path from the project root to the directory where the index.php is.
* Docker not mounting your code: If you `ddev ssh` and `ls` and there's nothing there, Docker may not be mounting your code. See [docker installation](./docker_installation.md) for testing docker install. (Is Docker, the drive or directory where your project is must be shared. In Docker Toolbox it *must* be a subdirectory of your home directory unless you [make special accomodations for Docker Toolbox](http://support.divio.com/local-development/docker/how-to-use-a-directory-outside-cusers-with-docker-toolbox-on-windowsdocker-for-windows)).

<a name="old-snapshot"></a>
## Can't restore snapshot from a MariaDB 10.1 database (before ddev v1.3)

Database snapshots from MariaDB 10.1 (normally from before ddev v1.3) cannot be restored into a MariaDB 10.2 environment. If you need to restore a 10.1 snapshot, here's how to do it. 

1. Back up any existing database you have running with `ddev export-db` or something like `ddev snapshot --name=before_reverting_to_10.1`
2. `ddev stop --remove-data` will completely remove the existing (10.2) database.
3. `ddev config --mariadb-version=10.1`
4. `ddev start` to start with MariaDB 10.1
5. Use `ddev restore-snapshot` to restore the snapshot by name
6. If you want to go upgrade your restored database to MariaDB 10.2, you can 
  * `ddev config --mariadb-version=10.2`
  * `ddev restart`
 

## Windows-Specific Issues
<a name="windows-hosts-file-limited">
### Windows Hosts File limited to 10 hosts per IP address line

On Windows only, there is a limit to the number of hosts that can be placed in one line. But since all ddev hosts are typically on the same IP address (typically 127.0.0.1, localhost), they can really add up. As soon as you have more than 10 entries there, your browser won't be able to resolve the addresses beyond the 10th entry.

There are two workarounds for this problem:

1. Use `ddev stop --all` and `sudo ddev hostname --remove-inactive` to prune the number of hosts on that hosts-file line. When you start a project, the hostname(s) associated with that project will be added back again.
2. Manually edit the hosts file (typically `C:\Windows\System32\drivers\etc\hosts`) and put some of your hosts on a separate line in the file. 


## More Support

[Support options](https://ddev.readthedocs.io/en/stable/#support) has a variety of options.
