# Lagoon Integration

DDEV provides integration with [Lagoon](https://lagoon.sh/), allowing users to quickly sync the files and database from a Lagoon environment to the local DDEV project.

## Lagoon Per-Project Configuration

1. Check out the Lagoon project and configure it by running [`ddev config`](../usage/commands.md#config). You’ll want to run [`ddev start`](../usage/commands.md#start) and make sure the basic functionality is working.
2. Add `LAGOON_PROJECT` and `LAGOON_ENVIRONMENT` variables to your project using `'web_environment'` in its YAML configuration or a `.ddev/.env` file. For example, run `ddev config --web-environment-add="LAGOON_PROJECT=<project-name>,LAGOON_ENVIRONMENT=<environment-name>"`.
3. Configure an [SSH key](https://docs.lagoon.sh/using-lagoon-advanced/ssh/) for your Lagoon user.
4. Run `ddev auth ssh` to make your SSH key available in the project’s web container.
5. Run [`ddev restart`](../usage/commands.md#restart).
6. Run `ddev pull lagoon`. After you agree to the prompt, the current upstream databases and files will be downloaded.
7. Optionally run `ddev push lagoon` to push local files and database to Lagoon. The [`ddev push`](../usage/commands.md#push) command can potentially damage your production site, so we don’t recommend using it.

## Usage

* `ddev pull lagoon` will connect to the Lagoon environment to download database and files. To skip downloading and importing either file or database assets, use the `--skip-files` or `--skip-db` flags.
* If you need to change the `.ddev/providers/lagoon.yaml` recipe, you can change it to suit your needs, but remember to remove the `#ddev-generated` line from the top of the file.
