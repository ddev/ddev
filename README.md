#ddev

The purpose of *ddev* is to support developers with a local copy of a site for development purposes. It runs the site in a Docker containers similar to a normal hosting environment.

You can see all "ddev" usages using the help commands, like `ddev -h`, `ddev add -h`, etc.

## Key prerequisites
- The *workspace* where the code will be checked out is specified in "workspace" in your drud.yaml. It defaults to ~/.drud, but you may want to change it to something like ~/workspace with `drud config set --workspace ~/workspace`
- The *client* in drud.yaml is the name of the organization where the code repository is to be found. Where the app name "drudio" is used below, the client specified in drud.yaml is the default organization on github. So if "client" in drud.yaml is "newmediadenver", it will look for the repo in https://github.com/newmediadenver/drudio.
- In `ddev add drudio production` the first argument is the repo/site name, and the second is an arbitrary "environment name" (and source for the dev database), which is typically either "production" or "staging".

| Where you run the ddev command from is important. You must be in the top-level of your drud workspace (e.g. `cd ~/.drud`) to run commands. (If you don't, `local` directories will be spawned wherever you run the command from)|
---

## Usage
```
➜  .drud ddev --help
This Command Line Interface (CLI) gives you the ability to interact with the DRUD platform to manage applications, create a local development environment, or deploy an application to production. DRUD also provides utilities for securely uploading files and secrets associated with applications.

Usage:
  ddev [flags]
  ddev [command]

Available Commands:
  add         Add an existing application to your local development environment
  config      Set or view DRUD configurations.
  exec        run a command in an app container.
  hostname    Manage your hostfile entries.
  list        List applications that exist locally
  logs        Get the logs from your running services.
  restart     Stop and Start the app.
  rm          Remove an application's local services.
  sequelpro   Easily connect local site to sequelpro
  ssh         SSH to an app container.
  start       Start an application's local services.
  stop        Stop an application's local services.
  update      Update DRUD cli tool
  version     print ddev version and component versions
  workon      Set a site to work on

Flags:
      --config string   yaml config file (default "$HOME/drud.yaml")
  -p, --plugin string   Choose which plugin to use (default "legacy")

Use "ddev [command] --help" for more information about a command.
```


## Getting Started
Check out the git repository for the site you want to work on. `cd` into the directory and run `ddev config` and follow the prompts.

```
$ cd ~/Projects
$ git clone <git-url>/drud-d8.git
$ cd drud-d8 
$ ddev config
Name (drud-d8):
Type [drupal7, drupal8, wordpress]: drupal8
Docroot location: src

Your ddev configuration has been written to .ddev/config.yaml
```
Configuration files have now been created for your site. (Available for inspection/modification at .ddev/ddev.yaml and .ddev/ddev-compose.yaml).
Now that the configuration has been created, you can start your site with `ddev start` (still from within the project working directory):
```
$ ddev start

Successfully added drud-d8
Your application can be reached at: http://drud-d8.ddev.local
You can run "ddev describe" to get additional information about your site, such as database credentials.
```
And you can now visit your working site. Enjoy!

## Site Lifecyle
Create a new site using `ddev add sitename environment`. This command will spin-up a new site and make it available at http://local-sitename-environment/.

To see a list of your current sites you can use `ddev list`.

```
➜  ddev list
1 local site found.
NAME        ENVIRONMENT TYPE    URL                                 DATABASE URL    STATUS
sitename  environment   drupal  http://local-sitename-environment 127.0.0.1:32770 running
```

To stop the site, run `ddev stop sitename environment`.

If you run `ddev list` again, the site will still be listed, but now you'll see the status has changed to 'exited'

```
➜  ddev list
1 local site found.
NAME        ENVIRONMENT TYPE    URL                                 DATABASE URL  STATUS
sitename  environment   drupal  http://local-sitename-environment 127.0.0.1:0   exited
```

Once you are done with your site, you can remove it with `ddev rm sitename environment`.

## Interacting with your Site
All of the commands can be performed by explicitly specifying the sitename or, to save time, you can execute commands from the site directory. All of the following examples assume you are in the working directory of your site.

### Retrieve Site Metadata
To view information about a specific site (such as URL, MySQL credentials, mailhog credentials), run `ddev describe` from within the working directory of the site. To view information for any site, use `ddev describe sitename`. 

### Viewing Error Logs
To follow an error log (watch the lines in real time), run `ddev logs -f`. When you are done, press CTRL+C to exit from the log trail. If you only want to view the most recent events, omit the `-f` flag.

### Executing Commands
To run a command against your site use `ddev exec`. e.g. `ddev exec 'drush core-status'` would execute `drush core-status` against your site root. You are free to use any of [the tools included in the container](#tools-included-in-the-container).

### Getting into the Container
To interact with the site more fully, `ddev ssh` will drop you into a bash shell for your container.

## Tools Included in the Container
- Composer
- Drush
- WP-CLI
- Mailhog

## Building
 Environment variables:
 * DRUD_DEBUG: Will display more extensive information as a site is deployed.
 
 ```
 make 
 make linux
 make darwin
 make test
 make clean
 ```

## Testing
Normal test invocation is just `make test`. Run a single test with an invocation like `go test -v -run TestDevAddSites ./pkg/...`

* DRUD_DEBUG: It helps a lot to set DRUD_DEBUG=true to see what ddev commands are being executed in the tests.
* DDEV_BINARY_FULLPATH should be set to the full pathname of the ddev binary we're attempting to test. That way it won't accidentally use some other version of ddev that happens to be on the filesystem.
* SKIP_COMPOSE_TESTS=true allows skipping tests that require docker-compose. 