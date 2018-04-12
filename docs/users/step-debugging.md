<h1>Step-debugging with ddev and xdebug</h1>

Every ddev project is automatically configured with xdebug so that popular IDEs can do step-debugging of PHP code. It is disabled by default for performance reasons, so you'll need to enable it in your config.yaml.

xdebug is a server-side tool: It is installed automatically on the container and you do *not* need to install it on your workstation. All you have to do on your workstation perhaps to add a browser extension or bookmark.

All IDEs basically work the same: They listen on a port and react when they're contacted there. So IDEs other than those listed here should work fine, if listening on the default xdebug port 9000.

**Key facts:**
* You need to explicitly enable xdebug in your config.yaml.
* The debug server port on the IDE must be set to port 9000, which is the default and is probably already set in most IDEs. (If you need to change the xdebug port due to a port conflict on your host computer, you can do it with a PHP override, explained below.)

For more background on XDebug see [XDebug documentation](https://xdebug.org/docs/remote). The intention here is that one won't have to understand XDebug to do debugging.

For each IDE the link to their documentation is provided, and the skeleton steps required are listed here.

## Setup Instructions

### Enable or disable xdebug in your config.yaml

Use a post-start hook to enable or disable xdebug on startup:

```
hooks:
    post-start:
      - exec: enable_xdebug
```


* [PHPStorm](#phpstorm)
* [NetBeans](#netbeans)
* [Atom](#atom)


<a name="phpstorm"></a>
### PHPStorm Debugging Setup

[PHPStorm](https://www.jetbrains.com/phpstorm/download) is a leading PHP development IDE with extensive built-in debugging support. It provides two different ways to do debugging. One requires very little effort in the PHPStorm IDE (they call it zero-configuration debugging) and the other requires you to set up a "run configuration", and is basically identical to the Netbeans or Eclipse setup.

#### PHPStorm Zero-Configuration Debugging

PHPStorm [zero-configuration debugging](https://confluence.jetbrains.com/display/PhpStorm/Zero-configuration+Web+Application+Debugging+with+Xdebug+and+PhpStorm) means you only have to:

1. Toggle the “Start Listening for PHP Debug Connections” button:
  ![Start listening for debug connections button](images/phpstorm_listen_for_debug_connections.png)
2. Set a breakpoint.
3. Using bookmarks from https://www.jetbrains.com/phpstorm/marklets/, "start debugger"
4. Visit a page that should stop in the breakpoint you set.

#### PHPStorm "Run/Debug configuration" Debugging

PHPStorm [run/debug configurations](https://www.jetbrains.com/help/phpstorm/2017.1/run-debug-configurations.html) require slightly more up-front work but can offer more flexibility and may be easier for some people.

1. Under the "Run" menu select "Edit configurations"
2. Click the "+" in the upper left and choose "PHP Web Application" to create a configuration. Give it a reasonable name.
3. Create a "server" for the project. (Screenshot below)
4. Add file mappings for the docroot of the server. If your repo has the main code in the root of the repo, that will map to /var/www/html. If it's in a docroot directory, it would map to /var/www/html/docroot.
5. Set an appropriate breakpoint.
6. Start debugging by clicking the "debug" button, which will launch a page in your browser.

![PHPStorm debug start](images/phpstorm_config_debug_button.png)


Server creation:

![PHPStorm server creation](images/phpstorm_config_server_config.png)

<a name="netbeans"></a>
### Netbeans Debugging Setup

[Netbeans](https://netbeans.org/) is a free IDE which has out-of-the-box debugging configurations for PHP. You'll want the *PHP* download bundle from the [download page](https://netbeans.org/downloads/).

![Netbeans Debugging Port](images/netbeans_debugger_port.png)

1. Create a PHP project that relates to your project repository. (File->New Project->PHP Application with Existing Sources)
2. Under "Run as", choose "Local web site (running on local web server)".
3. Under "Name and Location", give the sources folder of the **docroot/webroot** of your project.
![Netbeans project name and location](images/netbeans_project_name_location.png)
4. Under "Run configuration" the project URL to the full URL of your dev project, for example http://drud-d8.ddev.local/, and choose the index file.
![Netbeans run configuration](images/netbeans_project_run_configuration.png)
5. Set a breakpoint.
6. Click the "Debug" button.

<a name="atom"></a>
### Atom Debugging Setup

[Atom](https://atom.io/) is an extensible developers' editor promoted by GitHub. The available extensions include [php-debug](https://atom.io/packages/php-debug) which you can use to conduct PHP debugging with the Xdebug PHP extension. This project is currently an alpha release. 

1. Install an xdebug helper extension for your browser, [as suggested in documentation](https://atom.io/packages/php-debug#setting-up-xdebug)
2. Under Preferences->+Install install the php-debug add-on:
![php-debug installation](images/atom_php_debug_install.png)
3. Add configuration to the Atom config.cson by choosing "Config..." under the "Atom" menu. A "php-debug" stanza must be added, with file mappings that relate to your project. (Example [config.cson snippet](snippets/atom_config_cson_snippet.txt)
![Atom cson config](images/atom_cson_config.png)
4. Open a project/folder and open a PHP file you'd like to debug.
5. Set a breakpoint. (Right-click->PHP Debug->Toggle breakpoint)
6. Turn on debugging in Atom (Right-click->PHP Debug->Toggle Debugging)
7. Turn on debugging in your browser using the browser extension.
8. Visit a page that should trigger your breakpoint.

An example configuration from [user contribution](https://github.com/drud/ddev/issues/610#issuecomment-359244922):
```
"php-debug":
    AutoExpandLocals: true
    DebugXDebugMessages: true
    MaxDepth: 6
    PathMaps: [
      "/path/to/container/docroot;/path/to/host/docroot"
    ]
    PhpException: {}
```
