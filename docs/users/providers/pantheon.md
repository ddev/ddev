## Pantheon Hosting Provider Integration

ddev provides configurable integration with the [Pantheon Website Management Platform](https://pantheon.io/), which allows Pantheon users to quickly download and provision a project from Pantheon in a local ddev-managed environment.

ddev's Pantheon integration pulls an existing backup from an existing Pantheon site/environment into your local system. Of course that means you must already have a Pantheon site with a backup in order to use it.

### Pantheon Quickstart

If you have ddev installed, and have an active Pantheon account with an active site, you can follow this guide to spin up a Pantheon project locally.

1. Get your Pantheon.io machine token:
   a. Login to your Pantheon Dashboard, and [Generate a Machine Token](https://pantheon.io/docs/machine-tokens/) for ddev to use.
   b. Add the API token to the `web_environment` section in your global ddev configuration at ~/.ddev/global_config.yaml

   ```
   web_environment:
   - TERMINUS_MACHINE_TOKEN=abcdeyourtoken`
   ```

2. Choose a Pantheon site and environment you want to use with ddev. You can usually use the site name, but in some environments you may need the site ID, which is the long 3rd component of your site dashboard URL. So if the site dashboard is at <https://dashboard.pantheon.io/sites/009a2cda-2c22-4eee-8f9d-96f017321555#dev/>, the site ID is 009a2cda-2c22-4eee-8f9d-96f017321555.

3. Check out project codebase from Pantheon. Enable the "Git Connection Mode" and use `git clone` to check out the code locally.

4. Configure the local checkout for ddev using `ddev config`

5. In your project's .ddev/providers directory, copy pantheon.yaml.example to pantheon.yaml and edit the `project_id` and `environment_name`.

6. `ddev start`

7. Run `ddev pull pantheon`. The ddev environment will spin up, download the latest backup from Pantheon, and import the database and files into the ddev environment. You should now be able to access the project locally.

_**Note for WordPress Users:** In order for your local project to load file assets from your local environment rather than the Pantheon environment it was pulled from, the URL of the project must be changed in the database by performing a search and replace. ddev provides an example `wp search-replace` as a post-pull hook in the config.yaml for your project. It is recommended to populate and uncomment this example so that replacement is done any time a backup is pulled from Pantheon._
