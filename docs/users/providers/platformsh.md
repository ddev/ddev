## Platform.sh Hosting Provider Integration

ddev provides integration with the [Platform.sh Website Management Platform](https://platform.sh/), which allows Platform.sh users to quickly download and provision a project from Platform.sh in a local ddev-managed environment.

ddev's Platform.sh integration pulls database and files from an existing Platform.sh site/environment into your local system so you can develop locally. Of course that means you must already have a Platform.sh site in order to use it.

### Platform.sh Quickstart

If you have ddev installed, and have an active Platform.sh account with an active site, you can follow this guide to spin up a Platform.sh project locally.

1. Check out the site from platform.sh and then configure it with `ddev config`. You'll want to use `ddev start` and make sure the basic functionality is working.
2. Obtain and configure an API token.
   a. Login to the Platform.sh Dashboard and go to Account->API Tokens to create an API token for ddev to use.
   b. Add the API token to the `web_environment` section in your global ddev configuration at ~/.ddev/global_config.yaml, `- PLATFORMSH_CLI_TOKEN=abcdeyourtoken`
   c. `ddev restart`
3. Obtain your project id with `ddev exec platform`. The platform tool should show you all the information about your account and project.
4. In your project's .ddev/platforms directory, copy platform.sh.example to platform.sh and edit the `project_id` and `environment_name`.
5. Run `ddev pull platform`. After you agree to the prompt, the current upstream database and files will be downloaded.

### Usage

`ddev pull platform` will connect to Platform.sh to download database and files. To skip downloading and importing either file or database assets, use the `--skip-files` and `--skip-db` flags.
