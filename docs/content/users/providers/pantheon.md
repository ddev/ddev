# Pantheon Integration

DDEV provides configurable integration with the [Pantheon Website Management Platform](https://pantheon.io/), which allows Pantheon users to quickly download and provision a project from Pantheon in a local DDEV-managed environment.

DDEV’s Pantheon integration pulls an existing backup from an existing Pantheon site/environment into your local system. That means you must already have a Pantheon site with a backup in order to use it.

## Pantheon Quickstart

If you have DDEV installed, and have an active Pantheon account with an active site, you can follow this guide to spin up a Pantheon project locally.

!!!tip "`ddev pull pantheon` fails due to Terminus version mismatch"
    See [How to Downgrade Terminus in DDEV's Web Container and Customize Other Bundled Tools](https://ddev.com/blog/ddev-bundled-tools-using-custom-versions/).

1. Get your Pantheon machine token:
    1. Log in to your Pantheon Dashboard and [Generate a Machine Token](https://pantheon.io/docs/machine-tokens/) for DDEV to use.
    2. Add the API token to the `web_environment` section in your global DDEV configuration at `~/.ddev/global_config.yaml`.

        ```yaml
        web_environment:
            - TERMINUS_MACHINE_TOKEN=your_token
        ```

    !!!tip "What if I have more than one API token?"
        To use multiple API tokens for different projects, add them to your per-project configuration using the [`.ddev/config.local.yaml`](../configuration/config.md#environmental-overrides) file instead. This file is gitignored by default.

        ```yaml
        web_environment:
            - TERMINUS_MACHINE_TOKEN=your_token
        ```

2. Choose a Pantheon site and environment you want to use with DDEV. You can usually use the site name, but in some environments you may need the site ID, which is the long third component of your site dashboard URL. So if the site dashboard is at `https://dashboard.pantheon.io/sites/009a2cda-2c22-4eee-8f9d-96f017321555#dev/`, the site ID is `009a2cda-2c22-4eee-8f9d-96f017321555`.

3. On the Pantheon dashboard for the site, make sure that at least one backup has been created. (When you need to refresh what you pull, create a new backup.)

4. For `ddev push pantheon` make sure your public SSH key is configured in Pantheon under *Account* → *SSH Keys*.

5. Check out the project codebase from Pantheon. Enable the “Git Connection Mode” and use `git clone` to check out the code locally.

6. Configure the local checkout for DDEV using [`ddev config`](../usage/commands.md#config).

7. If using Drupal 8+, verify that Drush is installed in your project with `ddev composer require drush/drush`. If using Drupal 6 or 7, Drush 8 is already provided in the web container’s `/usr/local/bin/drush`, so you can skip this step.

8. In your **project's** `.ddev/providers` directory, copy `pantheon.yaml` to your providers directory (*This refers to your project `.ddev` folder and not the global `.ddev` folder*). Add `PANTHEON_SITE` and `PANTHEON_ENVIRONMENT` variables to your project `.ddev/config.yaml`:

    ```yaml
    web_environment:
        - PANTHEON_SITE=yourprojectname
        - PANTHEON_ENVIRONMENT=dev
    ```

    You can also do this with `ddev config --web-environment-add="PANTHEON_SITE=yourprojectname,PANTHEON_ENVIRONMENT=dev"`.

    You can usually use the site name, but in some environments you may need the site ID, which is the long third component of your site dashboard URL. So if the site dashboard is at `https://dashboard.pantheon.io/sites/009a2cda-2c22-4eee-8f9d-96f017321555#dev/`, the site ID is `009a2cda-2c22-4eee-8f9d-96f017321555`.

    Instead of setting the environment variables in configuration files, you can use
    `ddev pull pantheon --environment=PANTHEON_SITE=yourprojectname,PANTHEON_ENVIRONMENT=dev` for example.

9. If using Colima, may need to set an explicit nameserver in `~/.colima/default/colima.yaml` like `1.1.1.1`. If this configuration is changed, may also need to restart Colima.

10. Run [`ddev restart`](../usage/commands.md#restart).

11. Run `ddev pull pantheon`. DDEV will download the Pantheon database and files and bring them into the local DDEV environment. You should now be able to access the project locally.

12. Optionally use `ddev push pantheon` to push local files and database to Pantheon. The [`ddev push`](../usage/commands.md#push) command can potentially damage your production site, so we don’t recommend using it.
