# Acquia Integration

DDEV provides integration with the [Acquia Cloud Platform](https://www.acquia.com/choosing-right-acquia-cloud-platform), which allows Acquia users to quickly download and provision a project from Acquia in a local DDEV-managed environment.

DDEV’s Acquia integration pulls database and files from an existing project into your local system so you can develop locally.

## Acquia Quickstart

1. Get your Acquia API token from *Account Settings* → *API Tokens*.
2. Make sure you’ve added your SSH key to your Acquia account in *Account Settings* → *SSH Keys*.
3. Run [`ddev auth ssh`](../usage/commands.md#auth-ssh). (Typically once per DDEV session, not every pull.)
4. In `~/.ddev/global_config.yaml`, add or update the [`web_environment`](../configuration/config.md#web_environment) section with the API keys:

   ```yaml
   web_environment:
   - ACQUIA_API_KEY=xxxxxxxx
   - ACQUIA_API_SECRET=xxxxx
   ```

5. Copy `.ddev/providers/acquia.yaml.example` to `.ddev/providers/acquia.yaml`.
6. Update the `project_id` and database corresponding to the environment you want to work with.
   - If you have `acli` installed, you can run: `acli remote:aliases:list`.
   - Or, on the Acquia Cloud Platform navigate to the *Environments* page, click on the header, and look for the *SSH URL* line. For example, `project1.dev@cool-projects.acquia-sites.com` uses project ID `project1.dev`.
7. Your project must include Drush. Run `ddev composer require drush/drush` if it isn’t there already.
8. Run [`ddev restart`](../usage/commands.md#restart).
9. Use `ddev pull acquia` to pull the project database and files.
10. Optionally use `ddev push acquia` to push local files and database to Acquia. Be aware that [`ddev push`](../usage/commands.md#push) is a command that can potentially damage your production site, so we don’t recommend using it.

## Usage

`ddev pull acquia` will connect to the Acquia Cloud Platform to download database and files. To skip downloading and importing either file or database assets, use the `--skip-files` and `--skip-db` flags.
