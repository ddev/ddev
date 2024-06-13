# Troubleshooting

Things might go wrong! In addition to this page, consider checking [Stack Overflow](https://stackoverflow.com/tags/ddev) and [the DDEV issue queue](https://github.com/ddev/ddev/issues) and [other support options](../support.md), as well as [Docker troubleshooting suggestions](../install/docker-installation.md#testing-and-troubleshooting-your-docker-installation).

## General Troubleshooting Strategies

* Please use the [current stable version of DDEV](https://ddev.readthedocs.io/en/stable/users/install/ddev-upgrade/) and of your Docker provider before going too far or asking for support.
* Start by running [`ddev poweroff`](commands.md#poweroff) to make sure all containers can start fresh.
* Temporarily disable firewalls, VPNs, tunnels, network proxies, and virus checkers while you’re troubleshooting.
* Temporarily disable any proxies you’ve established in Docker’s settings.
* Check to see if you're out of disk space. On macOS, make sure that your Docker provider has adequate disk space allocated. (DDEV will normally warn you about problems in this situation.)
* On macOS, if you have a particular heavyweight project or are encountering `kill` statements in `ddev logs`, increase your memory allocation in your Docker provider. (Most projects are fine with 5-6GB allocated.)
* On macOS your Docker provider limits the amount of disk space available to Docker. Make sure that you increase it if you're seeing disk space problems.
* Please make sure that your project is in a subdirectory of your home directory and has normal ownership and privileges. For example, `ls -ld .` in your project directory should show you as owner of the directory and with write privileges.
* Use [`ddev debug dockercheck`](commands.md#debug-dockercheck) and [`ddev debug test`](commands.md#debug-test) to help sort out Docker problems.
* Make sure you do not have disk space problems on your computer. This can be especially tricky on WSL2, where you need to check both the main Windows disk space and WSL2 disk space as well.
* On macOS, check to make sure Docker Desktop or Colima are not out of disk space. In *Settings* (or *Preferences*) → *Resources* → *Disk image size* there should be ample space left; try not to let usage exceed 80% because the reported number can be unreliable. If it says zero used, something is wrong.
* If you have customizations like PHP overrides, nginx or Apache overrides, MySQL/PostgreSQL overrides, custom services, or `config.yaml` changes, please back them out while troubleshooting. It’s important to have the simplest possible environment while troubleshooting.
* Restart Docker. Consider a Docker factory reset in serious cases, which will destroy any databases you’ve loaded. See [Docker Troubleshooting](../install/docker-installation.md#troubleshooting) for more.
* Try the simplest possible DDEV project (like [`ddev debug test`](commands.md#debug-test) does):

    ```bash
    ddev poweroff
    mkdir ~/tmp/testddev
    cd ~/tmp/testddev
    ddev config --auto
    printf "<?php\nphpinfo();\n" > index.php
    ddev start
    ```

    If that starts up fine, there may be an issue specifically with the project you’re trying to start.

!!!tip "Using DDEV with Other Development Environments"

    DDEV uses your system’s port 80 and 443 by default when projects are running. If you’re using another local development environment (like Lando or Docksal or a native setup), you can either stop the other environment or configure DDEV to use different ports. See [troubleshooting](troubleshooting.md#unable-listen) for more detailed problem-solving. It’s easiest to stop the other environment when you want to use DDEV, and stop DDEV when you want to use the other environment.

### Debug Environment Variables

Two environment variables meant for DDEV development may also be useful for broader troubleshooting: `DDEV_DEBUG` and `DDEV_VERBOSE`. When enabled, they’ll output more information when DDEV is executing a command. `DDEV_VERBOSE` can be particularly helpful debugging Dockerfile problems because it outputs complete information about the Dockerfile build stage within the `ddev start` command.

You can set either one in your current session by running `export DDEV_DEBUG=true` and `export DDEV_VERBOSE=true`.

<a name="unable-listen"></a>

## Web Server Ports Already Occupied

DDEV may notify you about port conflicts with this message about port 80 or 443:

> Failed to start yoursite: Unable to listen on required ports, localhost port 80 is in use

DDEV sometimes also has this error message that will alert you to port conflicts:

> ERROR: for ddev-router Cannot start service ddev-router: Ports are not available: listen tcp 127.0.0.1:XX: bind: An attempt was made to access a socket in a way forbidden by its access permissions.

or

> Error response from daemon: Ports are not available: exposing port TCP 127.0.0.1:443 -> 0.0.0.0:0: listen tcp 127.0.0.1:443: bind: Only one usage of each socket address (protocol/network address/port) is normally permitted.

This means there’s another process or web server listening on the named port(s) and DDEV cannot access the port. The most common conflicts are on ports 80 and 443.

In some cases, the conflict could be over Mailpit’s port 8025 or 8026.

To resolve this conflict, choose one of these methods:

1. Stop all Docker containers that might be using the port by running `ddev poweroff && docker rm -f $(docker ps -aq)`, then restart Docker.
2. If you’re using another local development environment that uses these ports (MAMP, WAMP, Lando, etc.), consider stopping it.
3. Fix port conflicts by configuring DDEV globally to use different ports.
4. Fix port conflicts by stopping the competing application.

### Method 1: Stop the conflicting application

Consider `lando poweroff` for Lando, or `fin system stop` for Docksal, or stop MAMP using GUI interface or [`stop.sh`](https://stackoverflow.com/a/17750194/215713).

### Method 2: Fix port conflicts by configuring your project to use different ports

To configure a project to use non-conflicting ports, remove router port configuration from the project and set it globally to different values. This will work for most people:

```
ddev config --router-http-port="" --router-https-port=""
ddev config global --router-http-port=8080 --router-https-port=8443
ddev start
```

This changes the project’s HTTP URL to `http://yoursite.ddev.site:8080` and the HTTPS URL to `https://yoursite.ddev.site:8443`.

If the conflict is over port 8025 or 8026, it’s probably clashing with Mailpit’s default port:

```
ddev config --mailpit-http-port="" --mailpit-https-port=""
ddev config global --mailpit-http-port=8301 --mailpit-https-port=8302
```

### Method 3: Fix port conflicts by stopping the competing application

Alternatively, stop the other application.

Probably the most common conflicting application is Apache running locally. It can often be stopped gracefully (but temporarily) with:

```
sudo apachectl stop
```

**Common tools that use port 80 and port 443:**

Here are some of the other common processes that could be using ports 80/443 and methods to stop them.

* macOS content filtering: Under "Screen Time" → "Choose Screen Time content and privacy settings", turn off "Content and Privacy" and then reboot. This has been a common issue with macOS Sonoma.
* macOS or Linux Homebrew: Look for active processes by running `brew services` and temporarily running `brew services stop` individually to see if it has any impact on the conflict.
* MAMP (macOS): Stop MAMP.
* Apache: Temporarily stop with `sudo apachectl stop`, permanent stop depends on your environment.
* nginx (macOS Homebrew): `sudo brew services stop nginx` or `sudo launchctl stop homebrew.mxcl.nginx`.
* nginx (Ubuntu): `sudo service nginx stop`.
* Apache (many environments, often named “httpd”): `sudo apachectl stop` or on Ubuntu `sudo service apache2 stop`.
* VPNKit (macOS): You likely have a Docker container bound to port 80. Do you have containers up for Lando or another Docker-based development environment? If so, stop the other environment.
* Lando: If you’ve previously used Lando, try running `lando poweroff`.
* IIS on Windows (can affect WSL2). You’ll have to disable it in the Windows settings.

To dig deeper, you can use a number of tools to find out what process is listening.

On macOS and Linux, try the `lsof` tool on ports 80 or 443 or whatever port you’re having trouble with:

```
$ sudo lsof -i :443 -sTCP:LISTEN
COMMAND  PID     USER   FD   TYPE DEVICE SIZE/OFF NODE NAME
nginx   1608 www-data   46u  IPv4  13913      0t0  TCP *:http (LISTEN)
nginx   5234     root   46u  IPv4  13913      0t0  TCP *:http (LISTEN)
```

You can also use the `netstat -anv -p tcp` command to examine processes running on specific ports:

```
sudo netstat -anv -p tcp | egrep 'Proto|(\*\.(80|443))'
Proto Recv-Q Send-Q  Local Address          Foreign Address        (state)      rhiwat  shiwat    pid   epid state  options           gencnt    flags   flags1 usscnt rtncnt fltrs
tcp4       0      0  *.80                   *.*                    LISTEN       131072  131072  10521      0 00100 00000006 000000000000965d 00000000 00000900      1      0 000001
tcp4       0      0  *.443                  *.*                    LISTEN       131072  131072  10521      0 00100 00000006 000000000000965c 00000000 00000900      1      0 000001```
```

The `pid` column shows the process ID of the process listening on the port. In this case, it’s `10521`. You can use `ps` to find out what process that is:

```
$ ps -p 10521
$ ps -fp 10521
UID   PID  PPID   C STIME   TTY           TIME CMD
501 10521     1   0 12:35PM ??         0:05.16 /Applications/OrbStack.app/Contents/MacOS/../Frameworks/OrbStack Helper.app/Contents/MacOS/OrbStack Helper vmgr -build-id 339077377 -handoff
```

On Windows CMD, use [sysinternals tcpview](https://docs.microsoft.com/en-us/sysinternals/downloads/tcpview) or try using `netstat` and `tasklist` to find the process ID:

```
> netstat -aon | findstr ":80.*LISTENING"
  TCP    127.0.0.1:80           0.0.0.0:0              LISTENING       5760
  TCP    127.0.0.1:8025         0.0.0.0:0              LISTENING       5760
  TCP    127.0.0.1:8036         0.0.0.0:0              LISTENING       5760

> tasklist | findstr "5760"
com.docker.backend.exe        5760 Services                   0      9,536 K
```

The resulting output displays which command is running and its PID. Choose the appropriate method to stop the other server.

You may also be able to find what’s using a port using `curl`. On Linux, macOS, or in Git Bash on Windows, `curl -I localhost` or `curl -I -k https://localhost:443`. The result may give you a hint about which application is at fault.

We welcome your [suggestions](https://github.com/ddev/ddev/issues/new) based on other issues you’ve run into and your troubleshooting technique.

### Debugging Port Issues on WSL2

On WSL2 it’s harder to debug this because the port may be occupied either on the traditional Windows side, or within your WSL2 distro. This means you may have to debug it in both places, perhaps using both the Windows techniques shown above and the Linux techniques shown above. The ports are shared between Windows and WSL2, so they can be broken on either side.

## Database Container Fails to Start

Use `ddev logs -s db` to see what’s wrong.

The most common cause of the database container not coming up is changing the database type or version in the project configuration, so the database server daemon is unable to start using an existing configuration for a different type or version.

To solve this:

* Change the configuration in `.ddev/config.yaml` back to the original configuration.
* Export the database with [`ddev export-db`](commands.md#export-db).
* Delete the project with [`ddev delete`](commands.md#delete), or stop the project and remove the database volume using `docker volume rm <project>-mariadb` or `docker volume rm <project>-postgres`.
* Update `.ddev/config.yaml` to use the new [database type or version](../extend/database-types.md).
* Start the project and import the database from your export.

## “web service unhealthy” or “web service starting” or Exited

Use `ddev logs` to see what’s wrong.

The most common cause of the web container being unhealthy is a user-defined `.ddev/nginx-full/nginx-site.conf` or `.ddev/apache/apache-site.conf`. Please rename these to `<xxx_site.conf>` during testing. To figure out what’s wrong with it after you’ve identified that as the problem, use `ddev logs` and review the error.

Changes to `.ddev/nginx-site.conf` and `.ddev/apache/apache-site.conf` take effect only when you do a `ddev restart` or the equivalent.

## No Input File Specified (404) or Forbidden (403)

If you get a 404 with “No input file specified” (nginx) or a 403 with “Forbidden” (Apache) when you visit your project, it usually means that no `index.php` or `index.html` is being found in the docroot. This can result from:

* Misconfigured docroot: If the docroot isn’t where the web server thinks it is, then the web server won’t find `index.php`. Look at your `.ddev/config.yaml` to verify it has a docroot containing `index.php`. It should be a relative path.
* Missing `index.php`: There may not be an `index.php` or `index.html` in your project.

## `ddev start` Fails and Logs Contain "failed (28: No space left on device)" - Docker File Space

If `ddev start` fails, it’s most often because the `web` or `db` container fails to start. In this case, the error message from `ddev start` says something like “Failed to start <project>: db container failed: log=, err=container exited, please use 'ddev logs -s db' to find out why it failed”. You can`ddev logs -s db` to find out what happened.

If you see any variant of “no space left on device” in the logs when using Docker Desktop, it means you have to increase or clean up Docker’s file space. Increase the “Disk image size” setting under “Resources” in Docker’s Preferences:

![Docker disk space](../../images/docker-disk-image-size.png)

If you see “no space left on device” on Linux, it most likely means your filesystem is full.

## `ddev start` Fails with "container failed to become ready"

A container fails to become ready when its health check is failing. This can happen to any of the containers, and you can usually find the issue with a `docker inspect` command.

!!!tip
    You may need to install [jq](https://stedolan.github.io/jq/download/) for these examples (`brew install jq`), or remove the `| jq` from the command and read the raw JSON output.

For the `web` container:

```
docker inspect --format "{{json .State.Health }}" ddev-<projectname>-web | jq
```

For `ddev-router`:

```
docker inspect --format "{{json .State.Health }}" ddev-router
```

For `ddev-ssh-agent`:

```
docker inspect --format "{{json .State.Health }}" ddev-ssh-agent
```

Don’t forget to check logs using `ddev logs` for the `web` container, and `ddev logs -s db` for the `db` container!

For `ddev-router` and `ddev-ssh-agent`: `docker logs ddev-router` and `docker logs ddev-ssh-agent`.

Run [`ddev debug router-nginx-config`](commands.md#debug-router-nginx-config) to print the nginx configuration of the currently running `ddev-router`.

## `ddev start` Fails with "Failed to start [project name]: No such container: ddev-router"

Deleting the images and re-pulling them generally solves this problem.

Try running the following commands from the host machine:

```
ddev poweroff
docker rm -f $(docker ps -aq)
docker rmi -f $(docker images -q)
```

You should then be able to start your DDEV machine.

## `ddev --version` shows an old version

If you have installed or upgraded DDEV to the latest version, but when you check the actual version with `ddev --version`, it shows an older version, please refer to [Why do I have an old DDEV?](./faq.md#why-do-i-have-an-old-ddev)

## Trouble Building Dockerfiles

The additional `.ddev/web-build/Dockerfile` capability in DDEV is wonderful, but it can be hard to figure out what to put in there.

The best approach for any significant Dockerfile is to `ddev ssh` and `sudo -s` and then one at a time, do the things that you plan to do with a `RUN` command in the Dockerfile.

For example, if your Dockerfile were

```dockerfile
RUN npm install --global forever
```

You could test it with `ddev ssh`, `sudo -s`, and then `npm install --global forever`.

The error messages you get will be more informative than messages that come when the Dockerfile is processed.

You can also see the output from the full Docker build using either

```
ddev debug refresh
```

or

```
~/.ddev/bin/docker-compose -f .ddev/.ddev-docker-compose-full.yaml --progress=plain build --no-cache
```

### Docker build fails `apt-get update`, perhaps "SSL certificate problem: self-signed certificate"

The Docker build environment (where all projects have a little bit happening) is very sensitive to problems with `apt-get update` or with TLS certificate authentication. If you ware seeing problems with `apt-get update` failing, some of these strategies may help:

* **WSL2**: On WSL2 it's a known issue that the WSL2 environment time can get out of sync with the real time. This is an [ongoing problem](https://github.com/microsoft/WSL/issues/10006) with WSL2, and can be fixed with various workarounds. One good workaround is to install `ntpdate` and `sudo ntpdate pool.ntp.org` to sync the time. The time in WSL2 can get out of sync due to laptop sleeping or other causes. A reboot also fixes it.
* **VPN**: If you are on a packet-inspection VPN, it often causes problems with validation of certificates on internet sites. In that situation you'll need to get the CA updates required and install them with a custom Dockerfile, as described on [Stack Overflow](https://stackoverflow.com/questions/71595327/corporate-network-ddev-composer-create-results-in-ssl-certificate-error/71595428#71595428).
* **Other Docker Build**: The Dockerfile build environment is different from the host-side build and different from what you get with `ddev ssh`. If you're having trouble with it it may be caused by name resolution or IP connectivity problems, most often caused by a firewall or VPN. Turn off your firewall temporarily and VPN. A good debugging technique would be to do a simple `.ddev/web-build/Dockerfile` that does `RUN curl -I https://www.google.com` and then use `ddev debug refresh` to see the result. If it gets a 200 result, then your name resolution and internet connectivity are working in the Docker build environment.

## DDEV Starts but Browser Can’t Access URL

You may see one of these messages in your browser:

* `403` Forbidden
* *[url] server IP address could not be found*
* *We can’t connect to the server at [url]*

If you get the `403 Forbidden` it's almost always because your [docroot is set wrong](faq.md#why-do-i-get-a-403-or-404-on-my-project-after-ddev-launch). You should have something like `docroot: web` or `docroot: ""` or `docroot: docroot` with the relative path to the directory where your `index.php` lives in the project.

**Name resolution**: Most people use `*.ddev.site` URLs, which work great most of the time but require internet access.

`*.ddev.site` is a wildcard DNS entry that always returns the IP address 127.0.0.1 (localhost). If you’re not connected to the internet, however, or if various other name resolution issues fail, this name resolution won’t work.

While DDEV can create a web server and a Docker network infrastructure for a project, it doesn’t have control of your computer’s name resolution, so its backup technique to make a hostname resolvable by the browser is to add an entry to the hosts file (`/etc/hosts` on Linux and macOS, `C:\Windows\system32\drivers\etc\hosts` on traditional Windows).

* If you’re not connected to the internet, your browser will not be able to look up `*.ddev.site` hostnames. DDEV works fine offline, but for your browser to look up names they’ll have to be resolved in a different way.
* DDEV assumes that hostnames can be resolved within 3 seconds. That assumption is not valid on all networks or computers, so you can increase the amount of time it waits for resolution. Increasing to 5 seconds, for example: `ddev config global --internet-detection-timeout-ms=5000`.
* If DDEV detects that it can’t look up one of the hostnames assigned to your project for that or other reasons, it will try to add that to the hosts file on your computer, which requires administrative privileges (sudo or Windows UAC).
    * This technique may not work on Windows WSL2, see below.

### DNS Rebinding Prohibited (Mostly on Fritzbox Routers)

You may see one of several messages:

* *Cannot resolve*
* *unknown host*
* *No address associated with hostname*

Some DNS servers prevent the use of DNS records that resolve to `localhost` (127.0.0.1) because in uncontrolled environments this may be used as a form of attack called [DNS Rebinding](https://en.wikipedia.org/wiki/DNS_rebinding). Since `*.ddev.site` resolves to 127.0.0.1, they may refuse to resolve, and your browser may be unable to look up a hostname, and give you messages like “<url> server IP address could not be found” or “We can’t connect to the server at <url>”.

You verify this is your problem by running `ping -c 1 dkkd.ddev.site`. If you get “No address associated with hostname” or something of that type, your computer is unable to look up `*.ddev.site`.

In this case, you can take any one of the following approaches:

1. Reconfigure your router to allow DNS Rebinding. Many Fritzbox routers have added default DNS Rebinding disallowal, and they can be reconfigured to allow it. See [issue](https://github.com/ddev/ddev/issues/2409#issuecomment-686718237). If you have the local dnsmasq DNS server it may also be configured to disallow DNS rebinding, but it’s a simple change to a configuration directive to allow it.
2. Most computers can use most relaxed DNS resolution if they are not on corporate intranets that have non-internet DNS. So for example, the computer can be set to use 8.8.8.8 (Google) or 1.1.1.1 (Cloudflare) for DNS name resolution, see [this article](https://www.hellotech.com/guide/for/how-to-change-dns-server-windows-mac).
3. If you have control of the router, you can usually change its DHCP settings to choose a public, relaxed DNS server as in #2.
4. You can live with DDEV trying to edit the `/etc/hosts` file, which it only has to do when a new name is added to a project.

An extensive discussion of this class of problem is on [ddev.com](https://ddev.com/blog/ddev-name-resolution-wildcards).

## Windows WSL2 Network Issues

* Some recent WSL2 versions have had very slow or completely failed network access inside the container or during the Docker build process. A `wsl --shutdown` or a reboot seems to clear these up.
* If you’re using a browser on Windows, accessing a project in WSL2, you can end up with confusing results when your project is listening on a port inside WSL2 while a Windows process is listening on that same port. The way to sort this out is to stop your project inside WSL2, verify that nothing is listening on the port there, and then study the port on the Windows side by visiting it with a browser or using other tools as described above.

## Limitations on Symbolic Links (symlinks)

Symbolic links are widely used but have specific limitations in many environments beyond DDEV. Here are some of the ways those may affect you:

* **Crossing mount boundaries**: Symlinks may not generally cross between network mounts. In other words, if you have a relative symlink in the root of your project directory on the host that points to `../somefile.txt`, that symlink will not be valid inside the container where `../` is a completely different filesystem (and is typically not mounted).
* **Symlinks to absolute paths**: If you have an absolute symlink to something like `/Users/xxx/somefile.txt` on the host, it will not be resolvable inside the container because `/Users` is not mounted there. Some tools, especially on Magento 2, may create symlinks to rooted paths, with targets like `/var/www/html/path/to/something`. These basically can’t make it to the host and may create errors.
* **Windows restrictions on symlinks**: Inside the Docker container on Windows, you may not be able to create a symlink that goes outside the container.
* **Mutagen restrictions on Windows symlinks**: On macOS and Linux (including WSL2) the default `.ddev/mutagen/mutagen.yml` chooses the `posix-raw` type of symlink handling. (See [Mutagen docs](https://mutagen.io/documentation/synchronization/symbolic-links)). This basically means that any symlink created will try to sync, regardless of whether it’s valid in the other environment. However, Mutagen does not support posix-raw on traditional Windows, so DDEV uses the `portable` symlink mode. So on Windows with Mutagen, symlinks have to be strictly limited to relative links that are inside the Mutagen section of the project.

### Delete and Re-Download Docker Images

In a few unusual cases, the actual downloaded Docker images can somehow get corrupted. Deleting the images will force them to be re-downloaded or rebuilt. This does no harm, as everything is rebuilt, but running `ddev start` will take longer while it downloads needed resources:

```bash
ddev poweroff
docker rm -f $(docker ps -aq) # Stop any other random containers that may be running
docker rmi -f $(docker images -q) # You might have to repeat this to get rid of all images
```
