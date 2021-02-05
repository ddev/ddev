## DDEV-Live Hosting Integration

DDEV provides a hosting solution, [DDEV-Live](https://ddev.com/ddev-live/), as part of the full DDEV development and deployment platform. You can use DDEV-Live with DDEV-Local as you would use any hosting provider, by pushing code to GitHub or Gitlab, exporting databases to a file and then pushing them up, etc.

Visit the [documentation pages for DDEV-Live](https://docs.ddev.com/getting-started/) to get started with Drupal, TYPO3, or other PHP projects.

### DDEV-Live Quickstart

If you have a DDEV-Live account with an active site, you can follow this quick start guide to spin up a DDEV-Live site locally. If you don't already have a DDEV-Live account, a free short-term trial is [available on signup](https://dash.ddev.com/).

1. Get your DDEV-Live API token on the [DDEV Dashboar](https://dash.ddev.com/settings/integration).
2. Using either `ddev-live` on the host or in the web container, authenticate and create a database backup, `ddev-live create backup database <sitename>`. You can do this again any time you need a fresh version from the upstream site.
3. Create a files backup using `ddev-live create backup files <sitename>`. This doesn't need to be done again until you have upstream files that have changed that you need.
4. Check out the git repository of the site
5. Use `ddev config` to configure it.
6. Copy .ddev/providers/ddev-live.yaml.example to .ddev/providers/ddev-live.yaml
7. Edit ddev-live.yaml to place the database backup name in the `database_backup:` section
8. Add add an entry to web_environment in ~/.ddev/global_config.yaml with the token:

   ```yaml
   web_environment:
   - DDEV_LIVE_API_TOKEN=xxxxxxxx
   ```

9. Update project_id and database_backup.
10. `ddev restart`
11. Use `ddev pull ddev-live` to pull the project database and files.

_**Note for WordPress Users:** In order for your local project to load file assets from your local environment rather than the DDEV-Live environment it was pulled from, the URL of the project must be changed in the database by performing a search and replace. ddev provides an example `wp search-replace` as a post-pull hook in the config.yaml for your project. It is recommended to populate and uncomment this example so that replacement is done any time a backup is pulled from DDEV-Live._
