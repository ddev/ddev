# Step Debugging with Xdebug

Every DDEV project is automatically configured with Xdebug so that popular IDEs can do step debugging of PHP code.

Xdebug is a server-side tool, and it’s installed automatically in the container so you don’t have to install or configure it on your workstation. Xdebug is disabled by default for performance reasons, so you’ll need to enable it and configure your IDE before can start debugging.

All IDEs basically work the same, listening on a port and reacting when they’re contacted there. IDEs other than those listed here work fine, if they listen on Xdebug’s default port 9003.

!!!tip
    This was port 9000 through DDEV v1.18, changed in v1.19+ to port 9003.

**Key facts:**

* Enable Xdebug by running `ddev xdebug` or `ddev xdebug on` from your project directory.  
It will remain enabled until you start or restart the project.
* Disable Xdebug for better performance when not debugging with `ddev xdebug off`.
* `ddev xdebug status` will show Xdebug’s current status.
* The IDE’s debug server port must be set to Xdebug’s default 9003 (port 9000 before v1.19), which may already be the case in popular IDEs. If you have a port conflict, see [Using Xdebug on a Port Other than the Default 9003](#using-xdebug-on-a-port-other-than-the-default-9003) below.

For more background on Xdebug, see [Xdebug documentation](https://xdebug.org/docs/remote). The intention here is that one won’t have to understand Xdebug to do debugging.

## IDE Setup

* [PhpStorm](#phpstorm-debugging-setup)
* [Visual Studio Code (VS Code)](#visual-studio-code-vs-code-debugging-setup)

<a name="phpstorm"></a>

### PhpStorm Debugging Setup

[PhpStorm](https://www.jetbrains.com/phpstorm/download) is a leading PHP development IDE with extensive built-in debugging support. It provides two ways to do debugging: “zero-configuration” debugging which requires very little effort in the PhpStorm IDE, and a “run configuration” you can set up just like Netbeans or Eclipse.

!!!tip "If you’re using PhpStorm inside WSL2 or Linux"
    Make PhpStorm listen for Xdebug using IPv4 by visiting Help → Edit Custom VM Options, and adding an additional line: `-Djava.net.preferIPv4Stack=true`. The Linux version of PhpStorm seems to otherwise default to only using IPv6.

#### PhpStorm Zero-Configuration Debugging

PhpStorm [zero-configuration debugging](https://confluence.jetbrains.com/display/PhpStorm/Zero-configuration+Web+Application+Debugging+with+Xdebug+and+PhpStorm) will automatically detect a connection and offer to create a “server” that maps your workstation files to the container. This means you only have to:

1. Toggle the “Start Listening for PHP Debug Connections” button:  
    ![Start listening for debug connections button](../images/phpstorm_listen_for_debug_connections.png)
2. Set a breakpoint.
3. Visit a page that should stop in the breakpoint you set.
4. PhpStorm will ask you what mapping to use, so it can figure out how to map the path inside the container to the path on your workstation. The first time you do this with a project, PhpStorm will pop up a “server” dialog box to set the mapping. The default will work, but it’s best to click the checkbox to map the whole project directory to `/var/www/html`.

When using this zero-configuration option:

1. Use the latest DDEV version.
2. Under *Run* → *Edit Configurations*, check that there are no servers already defined. PhpStorm will create a new server for you if no others exist. You can delete all servers and have PhpStorm create a new one, or you can create/edit an existing server as discussed below.

#### PhpStorm "Run/Debug Configuration" Debugging

PhpStorm [run/debug configurations](https://www.jetbrains.com/help/phpstorm/creating-and-editing-run-debug-configurations.html) require more setup but may be easier and more flexible for some people.

1. Under the *Run* menu select *Edit configurations*.
2. Click the *+* in the upper left and choose *PHP Web Application* to create a configuration.  
Give it a reasonable name.
3. Create a “server” for the project. Make sure *Name* is exactly the same as your host (e.g. `my-site.ddev.site`):
    ![PhpStorm server creation](../images/phpstorm_config_server_config.png)
4. Add file mappings for the files on the server.  
Click on the local repo path and add `/var/www/html` as the *Absolute path on the server* and your repository root as the path on the host.
5. Set an appropriate breakpoint.
6. Start debugging by clicking the “debug” button, which will launch a page in your browser:
    ![PhpStorm debug start](../images/phpstorm_config_debug_button.png)

#### PhpStorm and Command-Line Debugging

If you need to debug command-line PHP processes, especially code that’s outside the docroot, the environment variable `PHP_IDE_CONFIG` is already set inside the web container, so you don’t have to do much more.

However, if you’ve not yet used PhpStorm with Xdebug for a regular web request, do that to automatically create the PhpStorm “server” with the same name as your primary URL (*Languages and Frameworks* → *PHP* → *Servers*). The key job of the server is to map filesystem locations on the workstation (your computer) to filesystem locations on the remote server—in this case the `ddev-webserver` container. Often, PhpStorm automatically sets up a mapping that doesn’t include the entire project, meaning things like the `vendor` directory may not be mapped. You can fix this by mapping the top-level directory of your project to `/var/www/html` in the container:
![PhpStorm mapping](../images/PHPStormServerMapping.png)

### Visual Studio Code (VS Code) Debugging Setup

1. Install the [PHP Debug](https://marketplace.visualstudio.com/items?itemName=felixfbecker.php-debug) extension.
2. Update the project’s `.vscode/launch.json` to add the “Listen for Xdebug” configuration from [this config snippet](../snippets/launch.json). For more on customizing `launch.json`, see the [VS Code docs](https://code.visualstudio.com/docs/editor/debugging#_launch-configurations).
3. Set a breakpoint in your `index.php`. If it isn’t solid red, restart.
4. In the menu, choose *Run* → *Start Debugging*. You may have to select “Listen for Xdebug” by the green arrowhead at the top left. The bottom pane of VS Code should now be orange (live) and should say “Listen for Xdebug”.
5. Enable Xdebug with `ddev xdebug on`.
6. In a browser, visit your project and confirm you hit the breakpoint.

!!!tip "If you’re using VS Code on Windows with WSL2"
    VS Code should suggest two extensions if you have WSL2 enabled along with a PHP project: “PHP Debug” and “Remote - WSL”. You’ll need to enable both of these extensions in your distro (e.g. Ubuntu).

## Using Xdebug on a Port Other than the Default 9003

By default, DDEV is set up to contact the default port, port 9003 on your IDE. However, if you have something else listening on that port or your IDE does not yet default to 9003, you’ll need to change the port. (PhpStorm and VS Code have switch to supporting 9003 instead of 9000 for some time now.)

* To override the port, add an override file in the project’s `.ddev/php directory`. For example, a file `.ddev/php/xdebug_client_port.ini` would change to use the legacy port 9000:

    ```ini
    [PHP]
    xdebug.client_port=9000
    ```

* Then change your IDE’s configuration to listen on the new port.

!!!tip
    If you’re using a PHP version below 7.2, you’ll be using Xdebug version 2.x instead of 3.x and your port config should be `xdebug.remote_port` instead.

## Troubleshooting Xdebug

The basic thing to understand about Xdebug is that it’s a network protocol. Your IDE (like PhpStorm) will listen on the Xdebug port (9003 by default in v1.19+, previously 9000). If Xdebug is enabled in the DDEV web container (`ddev xdebug on`), PHP inside that container will try to open a TCP connection to the IDE. Docker’s networking places the host-side listening IDE at `host.docker.internal:9003`, so you have to make sure the network connection is clear and can be made and everything should work.

Here are basic steps to take to sort out any difficulty:

* Remember the port in play is port 9003 for DDEV v1.19+, and port 9000 for earlier versions.
* Reboot your computer.
* Temporarily disable any *firewall* or *VPN* if you’re having trouble. Xdebug is a network protocol, and the PHP process inside the web container must be able to establish a TCP connection to the listening IDE (PhpStorm, for example).
* Use `ddev xdebug on` to enable Xdebug when you want it, and `ddev xdebug off` when you’re done with it.
* Set a breakpoint at the first executable line of your `index.php`.
* Tell your IDE to start listening. (PhpStorm: click the telephone button, VS Code: run the debugger.)
* Use `curl` or a browser to create a web request. For example, `curl https://d9.ddev.site`.
* If the IDE doesn’t respond, take a look at `ddev logs`. A message like this means Xdebug inside the container can’t make a connection to port 9003:  

    > PHP message: Xdebug: [Step Debug] Could not connect to debugging client. Tried: host.docker.internal:9003 (through xdebug.client_host/xdebug.client_port)

* `ddev ssh` into the web container. Can you run `telnet host.docker.internal 9003` and have it connect? If not, you might have an over-aggressive firewall. Disable it, or add a rule that would allow the connection to pass through. For example, on Debian/Ubuntu that would be `sudo ufw allow 9003/tcp`.
* In PhpStorm, disable the “listen for connections” button so it won’t listen. Or just exit PhpStorm. With another IDE like VS Code, stop the debugger from listening.
* `ddev ssh` into the web container. Can you run `telnet host.docker.internal 9003` and have it connect? If so, you have something else running on port 9003. On the host, use `sudo lsof -i :9003 -sTCP:LISTEN` to find out what’s there and stop it. Don’t continue debugging until your telnet command does not connect. (On Windows WSL2 you may have to look for listeners both inside WSL2 and on the Windows side.)
* Now click the “listen” button on PhpStorm to start listening for connections.
* `ddev ssh` and try the `telnet host.docker.internal 9003` again. It should connect. If not, maybe PhpStorm is not listening, or not configured to listen on port 9003?
* Check to make sure that Xdebug is enabled. You can use `php -i | grep -i xdebug` inside the container, or use any other technique you want that gives the output of `phpinfo()`, including Drupal’s `admin/reports/status/php`. You should see `with Xdebug v3` and `php -i | grep xdebug.mode` should give you `xdebug.mode => debug,develop => debug,develop"`.
* Set a breakpoint in the first relevant line of your `index.php` and then visit the site in a browser. It should stop at that first line.
* If you’re using PhpStorm inside WSL2 (or perhaps other Linux configurations), go to *Help* → *Edit Custom VM Options* and add an additional line: `-Djava.net.preferIPv4Stack=true` This makes PhpStorm listen for Xdebug using IPv4; the Linux version of PhpStorm seems to default to using only IPv6.
* If you’re on WSL2 using Docker Desktop, make sure that the `docker` command is the one provided by Docker Desktop. `ls -l $(which docker)` should show a link to `/mnt/wsl/docker-desktop...`. If you’re on WSL2 using Docker installed inside WSL2, make sure that `ls -l $(which docker)` is *not* a link to `/mnt/wsl`.
