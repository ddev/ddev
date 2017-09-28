<h1>ddev Documentation</h1>

[ddev](https://github.com/drud/ddev) is an open source tool that makes it dead simple to get local PHP development environments up and running within minutes. It's powerful and flexible as a result of its per-project environment configurations, which can be extended, version controlled, and shared. In short, ddev aims to allow development teams to use Docker in their workflow without the complexities of bespoke configuration.



## System Requirements

- [Docker](https://www.docker.com/community-edition) version 17.05 or greater
- OS Support
  - macOS Sierra
  - Linux (See [Linux notes](users/linux_notes.md))
    * Ubuntu 16.04 LTS
    * Debian Jessie
    * Fedora 25
  - Windows 10 Pro (**experimental support!**)
    * See [Decisions and Plan for Linux & Windows Support](https://github.com/drud/ddev/issues/196#issuecomment-300178008) for more information and the various options on getting ddev operational on Windows 10.

We are open to expanding this list to include additional OSs as well as improve our existing support for the ones listed above. Please [let us know](https://github.com/drud/ddev/issues/new) if you hit an issue!

### Using ddev with other development environments
ddev requires ports 80 and 3306 to be available for use on your system when sites are running. If you are using another local development environment alongside ddev, please ensure the other environment is turned off or otherwise not occupying ports 80 and 3306.

If you need to use another environment after using ddev, simply ensure all of your ddev sites are stopped or removed. ddev only occupies system ports when at least one site is running.

## Installation
### Homebrew - macOS

For macOS users, we recommend downloading and installing ddev via [homebrew](https://brew.sh/):
```
brew tap drud/ddev && brew install ddev
```
Later, to upgrade to a newer version of ddev, simply run:
```
brew upgrade ddev
```

### Installation Script - Linux and macOS

Linux and macOS end-users can use this line of code to your terminal to download, verify, and install ddev using our [install script](https://github.com/drud/ddev/blob/master/install_ddev.sh):
```
curl https://raw.githubusercontent.com/drud/ddev/master/install_ddev.sh | bash
```

### Manual Installation - Linux and macOS
You can also easily perform the installation manually if preferred:

- Download and extract the latest [ddev release](https://github.com/drud/ddev/releases) for your architecture.
- Make ddev executable: `chmod ugo+x ddev`
- Move ddev to /usr/local/bin: `mv ddev /usr/local/bin/` (may require sudo), or another directory in your `$PATH` as preferred.
- Run `ddev` to test your installation. You should see ddev's command usage output.

### Manual Installation - Windows

- Download and extract the latest [ddev release](https://github.com/drud/ddev/releases) for Windows.
- Copy `ddev.exe` into `%HOMEPATH%\AppData\Local\Microsoft\WindowsApps`, or otherwise add `ddev.exe` to a folder defined in your `PATH`
- Run `ddev` from a Command Prompt or PowerShell to test your installation. You should see ddev's command usage output.

## Quickstart
ddev is designed to be as simple as possible to incorporate into existing Wordpress and Drupal workflows. You can start using ddev with any site just by running a few commands.

Below are quickstart instructions for each app type; Wordpress, Drupal 7, and Drupal 8.

**Note:** If you do not have ddev already on your machine, please follow the [installation instructions](https://ddev.readthedocs.io/en/latest/#installation) before beginning the quickstart tutorial. 
### Wordpress
To get started using ddev with a Wordpress site, simply clone the site's repository and checkout its directory.
```
git clone https://github.com/user/worpress_site
cd wordpress_site
```
Time to start setting up ddev. Inside of your site's working directory, enter the command:
```
ddev config
```

_Note: ddev config will prompt you for a site name and docroot._

After you've run `ddev config` you're ready to start running your site. Run ddev using a simple:
```
ddev start
``` 
When running `ddev start` you should see output informing you that the site's environment is being started. If startup is successful, you'll see a message like the one below telling you where the site can be reached.
```
Successfully started wordpress_site
Your application can be reached at: http://wordpress_site.ddev.local
```

##### Databases
**Important:** Before importing any databases for your site, please remove its' wp-config.php file (if there is one). 

_ddev will create its own wp-config.php automatically._

We're happy to say that importing a database into a site running on ddev is painless. 

Database imports can be accomplished using one command. And we currently offer support for several file types. Including: **.sql, sql.gz, tar, tar.gz, and zip**.

Here's an example of a database import using ddev:
```
ddev import-db --src=dumpfile.sql.gz
```
The `import-db` command will produce output so you can monitor the progress of the database import. 

For more in depth application monitoring, use the `ddev describe` command to see details about the status of your ddev app.

### Drupal 7
Beginning to use ddev with a Drupal 7 site is as simple as cloning the site's repository and checking out its directory.
```
git clone https://github.com/user/my_drupal7_site
cd my_drupal7_site
```
Now to start working with ddev. Inside of your site's working directory, enter the following command:
```
ddev config
```

_Note: ddev config will prompt you for a site name and docroot._

After you've run `ddev config` you're ready to start running your site. Run ddev using a simple:
```
ddev start
``` 
When running `ddev start` you should see output informing you that the site's environment is being started. If startup is successful, you'll see a message like the one below telling you where the site can be reached.
```
Successfully started my_drupal7_site
Your application can be reached at: http://my_drupal7_site.ddev.local
```

##### Databases
**Important:** Before importing any databases for your site, please remove its' settings.php file (if there is one). 

_ddev will create its own settings.php file automatically._

We're happy to say that importing a database into a site running on ddev is painless. 

Database imports can be accomplished using one command. And we currently offer support for several file types. Including: **.sql, sql.gz, tar, tar.gz, and zip**.

Here's an example of a database import using ddev:
```
ddev import-db --src=dumpfile.sql.gz
```
The `import-db` command will produce output so you can monitor the progress of the database import. 

For more in depth application monitoring, use the `ddev describe` command to see details about the status of your ddev app.

### Drupal 8

## Support
If you've encountered trouble using ddev, please use these resources to get help with your issue:

1. Please review the [ddev Documentation](https://ddev.readthedocs.io) to ensure your question isn't answered there.
2. Review the [ddev issue queue](https://github.com/drud/ddev/issues) to see if an issue similar to yours already exists.
3. If you've exhausted these options and still need help, please [file an issue](https://github.com/drud/ddev/issues/new) following the pre-populated guidelines and our [Contributing Guidelines](https://github.com/drud/ddev/blob/master/CONTRIBUTING.md) as best as possible.
