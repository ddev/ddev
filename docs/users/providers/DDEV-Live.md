## DDEV-Live Hosting Integration

DDEV provides a hosting solution, [DDEV-Live](https://ddev.com/ddev-live/), as part of the full DDEV development and deployment platform. You can use DDEV-Live with DDEV-Local as you would use any hosting provider, by pushing code to Github, exporting databases to a file and then pushing them up, etc.

Visit the [documentation pages for DDEV-Live](https://docs.ddev.com/getting-started/) to get started with Drupal, TYPO3, or other PHP projects.

### DDEV-Live Quickstart

If you have the DDEV-Local CLI (ddev) [installed](https://ddev.readthedocs.io/en/latest/#installation), and have a DDEV-Live account with an active WordPress, Drupal 7, or Drupal 8 site, you can follow this quick start guide to spin up a DDEV-Live site locally. If you don't already have a DDEV-Live account, a free short-term trial is [available on signup](https://dash.ddev.com/).

1. Authenticate with DDEV-Live.

    a. Log in to your DDEV-Live Dashboard, and [obtain the DDEV-Live API token](https://dash.ddev.com/settings/integration) for ddev to use.

    b. Run `ddev auth ddev-live <YOUR TOKEN>` (This is a one-time operation, and configures ddev to work with all the sites on your account.) (You can also set the default DDEV-Live "org" with `ddev auth ddev-live --default-org=<yourorg> <token>`)

2. Choose the DDEV-Live site you want to use with ddev.

3. Check out the github repo of the DDEV-Live site.

4. Configure the local checkout for ddev.

    * Navigate in your terminal to your checkout of the project codebase.
    * Run `ddev config ddev-live`. When asked for the project name use the exact name of the DDEV-Live project, as shown by `ddev-live list sites`.

5. Run `ddev pull`. The ddev environment will spin up, download the files and database, and import the database and files into the ddev environment. You should now be able to access the project locally.

_**Note for WordPress Users:** In order for your local project to load file assets from your local environment rather than the DDEV-Live environment it was pulled from, the URL of the project must be changed in the database by performing a search and replace. ddev provides an example `wp search-replace` as a post-pull hook in the config.yaml for your project. It is recommended to populate and uncomment this example so that replacement is done any time a backup is pulled from DDEV-Live._
