# Amazee Lagoon Integration

DDEV provides integration with the [Amazee Lagoon](https://lagoon.sh/), allowing Lagoon users to quickly download database and files from a Lagoon project to the local DDEV project.

## Amazee Lagoon Per-Project Configuration

1. Check out the Lagoon project and configure it with [`ddev config`](../usage/commands.md#config). You’ll want to use [`ddev start`](../usage/commands.md#start) and make sure the basic functionality is working.
2. Configure an SSH key at on the [Amazee Cloud Dashboard](https://dashboard.amazeeio.cloud/settings).
3. Add LAGOON_PROJECT and LAGOON_ENVIRONMENT variables to your project in 'web_environment' or a '.ddev/.env'. For example, `ddev config --web-environment-add="LAGOON_PROJECT=<project-name> LAGOON_ENVIRONMENT=<environment-name>"`.
4. `ddev auth ssh` if you haven't already done so. This will make your SSH key available in the web container of your project.
5. Run [`ddev restart`](../usage/commands.md#restart).
6. Run `ddev pull lagoon`. After you agree to the prompt, the current upstream databases and files will be downloaded.
7. Optionally use `ddev push lagoon` to push local files and database to Lagoon. The [`ddev push`](../usage/commands.md#push) command can potentially damage your production site, so we don’t recommend using it.

## Usage

* `ddev pull lagoon` will connect to Amazee Lagoon to download database and files. To skip downloading and importing either file or database assets, use the `--skip-files` and `--skip-db` flags.
* If you need to change the `.ddev/providers/lagoon.yaml` recipe, you can change it to suit your needs, but remember to remove the `#ddev-generated` line from the top of the file.
