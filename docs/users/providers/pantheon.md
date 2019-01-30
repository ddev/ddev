<h1>Pantheon Hosting Provider Integration</h1>

ddev provides an integration with the [Pantheon Website Management Platform](https://pantheon.io/), which allows for Pantheon users to quickly download and provision a project from Pantheon in a local ddev-managed environment.

ddev's Pantheon integration pulls an existing backup from an existing Pantheon site/environment into your local system so you can develop locally. Of course that means you must already have a Pantheon site with a backup in order to use it.

## Quick Start

If you have ddev installed, and have an active Pantheon account with an active WordPress, Drupal 7, or Drupal 8 site, you can follow this quick start guide to spin up a pantheon project locally.

1. Authenticate with Pantheon.

    a. Login to your Pantheon Dashboard, and [Generate a Machine Token](https://pantheon.io/docs/machine-tokens/) for ddev to use.

    b. Run `ddev auth-pantheon <YOUR TOKEN>` (This is a one-time operation, and configures ddev to work with all the sites on your account.)

2. Choose a Pantheon site and environment you want to use with ddev.

3. Get a copy of the project codebase from Pantheon. We recommend enabling the "Git Connection Mode" if not already enabled, and using `git clone` to check out the code locally.

4. Create a new backup of the Pantheon site. This can be done by navigating to Backups->Backup Log->Create New Backup. _Note: this must be done every time you want the latest state of your Pantheon environment provisioned locally._

5. Configure the local checkout for ddev.

    a. Navigate in your terminal to your checkout of the project codebase.

    b. Run `ddev config pantheon`. When asked for the project name you must use the exact name of the Pantheon project. The name is found in the URL of your Pantheon dev site. For example, if your site is viewed at http://dev-foo-bar.pantheonsite.io/ enter 'foo-bar' as the name (not 'Foo Bar').

    c. Configuration prompts will allow you to choose a Pantheon environment, suggesting "dev" as the default.

6. Run `ddev pull`. The ddev environment will spin up, download the latest backup from Pantheon, and import the database and files into the ddev environment. You should now be able to access the project locally.

## Requirements

In order to use ddev with Pantheon.io, you need the following:

- A [Pantheon.io](https://pantheon.io/) account. You can create a basic free account if you don't have one.

- A Pantheon authentication token. See instructions below.

## Authentication

This step only needs to be completed once for your system. We recommend that you create a token specific to ddev by going to https://pantheon.io/docs/machine-tokens/. Once you’ve completed that, run `ddev auth-pantheon <YOUR TOKEN>` and provide the token you just generated. This will store the token in ~/.ddev/pantheonconfig.json. If you ever delete the token, you will need to generate a new token from Pantheon and re-run the `ddev auth-pantheon` command to re-establish your connection to Pantheon.io.

## Usage

### Configuring a Pantheon project for Imports

After you copy your Pantheon project’s codebase locally, you can use `ddev config pantheon` to generate the necessary configuration files for ddev. In addition to the prompts the `ddev config` command provides for any new project, you will also be asked to specify which Pantheon environment you wish to pull your file and database assets from. These environments are usually "live", "test", or "dev". If you wish to later change the Pantheon environment you wish to sync from, you can do so by editing the `import.yaml` file in the `.ddev` directory for your project.

### Imports

Running `ddev pull` will connect to Pantheon through their API to find the latest versions of the database and files backups from the specified environment. If new versions are available on Pantheon, they are downloaded and stored in ~/.ddev/pantheon. If the stored copies there are the latest copies, ddev will use these cached copies instead of downloading them again. To skip downloading and importing either file or database assets, use the `--skip-files` and `--skip-db` flags. Use the `--env` flag to specify the Pantheon environment being pulled from.

_**Note for WordPress Users:** In order for your local project to load file assets from your local environment rather than the Pantheon environment it was pulled from, the URL of the project must be changed in the database by performing a search and replace. ddev provides an example `wp search-replace` as a post-pull hook in the config.yaml for your project. It is recommended to populate and uncomment this example so that replacement is done any time a backup is pulled from Pantheon._

## Troubleshooting

This section will cover tips and tricks to try if you run into any issues specific to the Pantheon integration. For now, please file any issues in the [ddev GitHub issue queue](https://github.com/drud/ddev/issues).
