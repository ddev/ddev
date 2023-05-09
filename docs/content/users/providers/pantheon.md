# Pantheon Integration

DDEV provides configurable integration with the [Pantheon Website Management Platform](https://pantheon.io/), which allows Pantheon users to quickly download and provision a project from Pantheon in a local DDEV-managed environment.

DDEV’s Pantheon integration pulls an existing backup from an existing Pantheon site/environment into your local system. That means you must already have a Pantheon site with a backup in order to use it.

## Pantheon Quickstart

If you have DDEV installed, and have an active Pantheon account with an active site, you can follow this guide to spin up a Pantheon project locally.

1. Get your Pantheon machine token:
   a. Log in to your Pantheon Dashboard and [Generate a Machine Token](https://pantheon.io/docs/machine-tokens/) for DDEV to use.
   b. Add the API token to the `web_environment` section in your global DDEV configuration at `~/.ddev/global_config.yaml`.

   ```
   web_environment:
   - TERMINUS_MACHINE_TOKEN=abcdeyourtoken
   ```

2. Choose a Pantheon site and environment you want to use with DDEV. You can usually use the site name, but in some environments you may need the site ID, which is the long third component of your site dashboard URL. So if the site dashboard is at `https://dashboard.pantheon.io/sites/009a2cda-2c22-4eee-8f9d-96f017321555#dev/`, the site ID is `009a2cda-2c22-4eee-8f9d-96f017321555`.

3. On the Pantheon dashboard for the site, make sure that at least one backup has been created. (When you need to refresh what you pull, create a new backup.)

4. For `ddev push pantheon` make sure your public SSH key is configured in Pantheon under *Account* → *SSH Keys*.

5. Check out the project codebase from Pantheon. Enable the “Git Connection Mode” and use `git clone` to check out the code locally.

6. Configure the local checkout for DDEV using [`ddev config`](../usage/commands.md#config).

7. If using Drupal 8+, verify that Drush is installed in your project with `ddev composer require drush/drush`. If using Drupal 6 or 7, Drush 8 is already provided in the web container’s `/usr/local/bin/drush`, so you can skip this step.

8. In your **project’s** `.ddev/providers` directory, copy `pantheon.yaml.example` to `pantheon.yaml` (*This refers to your project `.ddev` folder and not the global `.ddev` folder*).  Edit the `project` environment variable under `environment_variables`. It will be in the format `<projectname>.<environment>`, for example `yourprojectname.dev` or (in cases of ambiguity) `<project_uuid>.<environment>`, for example `009a2cda-2c22-4eee-8f9d-96f017321555.dev`.

9. If using Colima, may need to set an explicit nameserver in `~/.colima/default/colima.yaml` like `1.1.1.1`. If this configuration is changed, may also need to restart Colima.

10. Run [`ddev restart`](../usage/commands.md#restart).

11. Run `ddev pull pantheon`. DDEV will download the Pantheon database and files and bring them into the local DDEV environment. You should now be able to access the project locally.

12. Optionally use `ddev push pantheon` to push local files and database to Pantheon. The [`ddev push`](../usage/commands.md#push) command can potentially damage your production site, so we don’t recommend using it.
