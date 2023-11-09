# Acquia Integration

DDEV provides integration with the [Acquia Cloud Platform](https://www.acquia.com/choosing-right-acquia-cloud-platform), which allows Acquia users to quickly download and provision a project from Acquia in a local DDEV-managed environment.

DDEV’s Acquia integration pulls database and files from an existing project into your local system so you can develop locally.

## Acquia Quickstart

1. Get your Acquia API token from *Account Settings* → *API Tokens*.
2. Make sure you’ve added your SSH key to your Acquia account in *Account Settings* → *SSH Keys*.
3. Run [`ddev auth ssh`](../usage/commands.md#auth-ssh). (Typically once per DDEV session, not every pull.)
4. In `~/.ddev/global_config.yaml` (or the project `config.yaml`), add or update the [`web_environment`](../configuration/config.md#web_environment) section with the API keys:

   ```yaml
   web_environment:
   - ACQUIA_API_KEY=xxxxxxxx
   - ACQUIA_API_SECRET=xxxxx
   ```

5. In the project `.ddev/config.yaml` add the `ACQUIA_ENVIRONMENT_ID` environment variable:

   ```yaml
   web_environment:
   - ACQUIA_ENVIRONMENT_ID=yoursite.dev
   ```

6. Run [`ddev restart`](../usage/commands.md#restart).
7. Use `ddev pull acquia` to pull the project database and files.
8. Optionally use `ddev push acquia` to push local files and database to Acquia. Be aware that [`ddev push`](../usage/commands.md#push) is a command that can potentially damage your production site, so we don’t recommend using it.

## Usage

`ddev pull acquia` will connect to the Acquia Cloud Platform to download database and files. To skip downloading and importing either file or database assets, use the `--skip-files` and `--skip-db` flags.
