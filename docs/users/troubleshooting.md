<h1>Troubleshooting</h1>

Things might go wrong!

## Webserver ports are already occupied by another webserver

If you get a message from ddev about a port conflict on port 80 or 443, like this:

```
Failed to start yoursite: Unable to listen on required ports, Localhost port 80 is in use
```

it means that you have another webserver listening on port 80 (or 443, or both), and it needs to be stopped so that ddev can access the port. 

Probably the most common reason for this is that Apache is running locally. It can often be stopped gracefully (but temporarily) with:

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
