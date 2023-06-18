# Starting a Project

Once [DDEV’s installed](./install/ddev-installation.md), setting up a new project should be quick:

1. Clone or create the code for your project.
2. `cd` into the project directory and run [`ddev config`](./usage/commands.md#config) to initialize a DDEV project.
3. Run [`ddev start`](./usage/commands.md#start) to spin up the project.
4. Run [`ddev launch`](./usage/commands.md#launch) to open your project in a browser.

DDEV automatically detects your project type and docroot. If it guessed wrong or there’s something else you want to change, update [project options](./configuration/config.md) by editing `.ddev/config.yaml` and running [`ddev describe`](./usage/commands.md#start), or using the [`ddev config`](./usage/commands.md#config) command.

!!!tip "What’s a project type?"
    A `php` project type is the most general, ready for whatever modern PHP or static HTML/JS project you might be working on. It’s as full-featured as other [CMS-specific options](./quickstart.md), without any assumptions about your configuration or presets. (You can use this with a CMS or framework fine!)

If you need to configure your app to connect to the database, the hostname, username, password, and database name are each `db`.

While you’re getting your bearings, use [`ddev describe`](./usage/commands.md#describe) to get project details, and [`ddev help`](./usage/commands.md#help) to investigate commands.

Next, you may want to run [`ddev composer install`](./usage/commands.md#composer), [import a database](./usage/commands.md#import-db), or [load user-managed files](./usage/commands.md#import-files).

If you’re new to DDEV, check out [Using the `ddev` Command](./usage/cli.md) for an overview of what’s available.
