<h1>Troubleshooting</h1>

Things might go wrong!

<a name="unable-listen"></a>
## Webserver ports are already occupied by another webserver

If you get a message from ddev about a port conflict, like this:

```
Failed to start yoursite: Unable to listen on required ports, localhost port 80 is in use,
```

it means that you have another webserver listening on the named port(s), and it needs to be stopped so that ddev can access the port. 

You have two choices: 

1. You can configure your project to use different ports
2. You can stop the competing application.

### Configuring your project to use non-conflicting ports

To configure your project to use non-conflicting ports, edit the project's .ddev/config.yaml to add entries like `router_http_port: 8000` and `router_https_port: 8443` depending on your needs, then use `ddev start` again. For example, if you had a port conflict with a local apache http on port 80, you could add

```
router_http_port: 8000
```

to the config.yaml, and `ddev start`, and the project's http URL will change to http://yoursite.ddev.local:8000.


### Fixing port conflicts by stopping the other application

If you choose to do so you can also just stop the other application.

Probably the most common conflicting application is Apache running locally. It can often be stopped gracefully (but temporarily) with:

```
sudo apachectl stop
```

**Common tools that use port 80:**

There are many processes that could be using port 80. Here are some of the common ones and how to stop them:

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

As you see, the command that's running is listed, and its pid. You then need to use the appropriate technique to stop the other server. 


We welcome your [suggestions](https://github.com/drud/ddev/issues/new) based on other issues you've run into and your troubleshooting technique.

<a name="container-restarts"></a>
## DDEV-Local reports container restarts and does not arrive at "ready"

### Restarts of the database container

We've seen cases where this is caused by old databases that are not compatible with the current version of MariaDB that DDEV-Local is using. See [issue](https://github.com/drud/ddev/issues/615) for more information. The simple fix is to 

Note: Your project database will be destroyed by this procedure.

1. `ddev remove --remove-data`
2. rm -r .ddev (removes the config.yaml and docker-compose.yml, do this only if you haven't modified those)
3. `ddev start` 
4. `ddev import-db` if you have a db to import

### Restarts of the web container

The most common cause of the web container restarting is a user-defined .ddev/nginx-site.conf - Please rename it to nginx-site.conf.bak during testing. To figure out what's wrong with it after you've identified that as the problem, `ddev ssh` and look at /var/log/nginx/error.log or use `ddev logs` and review the error.

Changes to .ddev/nginx-site.conf take effect only when you do a `ddev rm` followed by `ddev start`.
