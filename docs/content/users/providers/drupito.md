# Drupito Integration

DDEV provides integration with [Drupito](https://drupito.com), a consumer-friendly managed Drupal hosting platform, so you can pull a site's backups into a local DDEV project and push your local database and files back to Drupito as new backups.

`ddev pull drupito` downloads the site's latest database and files backups and imports them locally. `ddev push drupito` uploads the local database and files to Drupito, where each becomes a new entry in the site's backup list; nothing on the site changes until you restore one of those backups in Drupito.

## Drupito Quickstart

1. Log in to Drupito and create an API token [app.drupito.com/manage/api-tokens](https://app.drupito.com/manage/api-tokens).

2. Add the token to the `web_environment` section of your global DDEV configuration at `$HOME/.ddev/global_config.yaml` (see [global configuration directory](../usage/architecture.md#global-files)), so it never lands in the project's Git repository:

    ```yaml
    web_environment:
        - DRUPITO_API_TOKEN=drupito_xxxxxxxx
    ```

    You can also do this with:

    ```bash
    ddev config global --web-environment-add="DRUPITO_API_TOKEN=drupito_xxxxxxxx"
    ```

    !!!tip "What if I have more than one API token?"
        To use multiple API tokens for different projects, add them to your per-project configuration using the [`.ddev/config.local.yaml`](../configuration/config.md#environmental-overrides) file instead. This file is gitignored by default.

3. In the project `.ddev/config.yaml`, add the site's machine name as `DRUPITO_SITE`:

    ```yaml
    web_environment:
        - DRUPITO_SITE=your_site
    ```

    You can also do this with:

    ```bash
    ddev config --web-environment-add="DRUPITO_SITE=your_site"
    ```

4. Optionally choose which backups `ddev pull drupito` uses, in the same place:

    ```yaml
    web_environment:
        - DRUPITO_SITE=your_site
        - DRUPITO_ENVIRONMENT=live
    ```

    | Variable | Value | Pulls |
    | --- | --- | --- |
    | `DRUPITO_ENVIRONMENT` | `live` (default) | the newest backup taken from the live environment |
    | `DRUPITO_ENVIRONMENT` | `next` | the newest backup taken from the next (aka stage) environment |
    | `DRUPITO_ENVIRONMENT` | `any` | the newest backup taken from any Drupito environment |
    | `DRUPITO_INSTANCE_ID` | an instance id, such as `K7Q2M` | the newest backup of that instance; `DRUPITO_ENVIRONMENT` is ignored when this is passed |

    Backups uploaded with `ddev push` are never chosen.

5. Run [`ddev restart`](../usage/commands.md#restart).

6. Run `ddev pull drupito` to download and import the site's latest database and files backups.

7. Optionally use `ddev push drupito` to upload your local database and files. Each upload becomes a new backup on the site; apply it to an environment using Drupito's "restore backup" feature.

## Usage

* [`ddev pull drupito`](../usage/commands.md#pull) downloads the newest database and files backups selected by `DRUPITO_ENVIRONMENT` or `DRUPITO_INSTANCE_ID`, verifies them, and imports them. To skip either part, use `--skip-files` or `--skip-db`.
* [`ddev push drupito`](../usage/commands.md#push) pushes the local database and files to Drupito as two separate backups. Nothing on the site changes; to use a pushed backup, import or restore it from the Drupito UI.
* To use another site, environment or instance without changing your project config, pass `--environment`:

    ```bash
    ddev pull drupito --environment="DRUPITO_ENVIRONMENT=any"
    ```

    You can combine multiple variables:

    ```bash
    ddev pull drupito --environment="DRUPITO_SITE=other_site,DRUPITO_INSTANCE_ID=00005"
    ```

* If you need to change the `drupito.yaml` recipe, you can change it to suit your needs, but remember to remove the `#ddev-generated` line from the top of the file.

## Files Directories

Drupito's files backups can hold two top-level directories, `public-files/` and `private-files/`. DDEV maps them to the project like this:

* `public-files/` is the project's public files directory, taken from [`upload_dirs`](../configuration/config.md#upload_dirs) (`sites/default/files` for Drupal projects). On push, the `php/`, `css/`, `js/` and `styles/` subdirectories and the top-level `.htaccess` are left out, since Drupal regenerates them.
* `private-files/` is included in `ddev push drupito` only when `DRUPITO_PRIVATE_FILES_DIR` names the directory, as a project-relative path in `web_environment` or with `--environment`. Use the same directory your `settings.php` names in `file_private_path`:

    ```bash
    ddev config --web-environment-add="DRUPITO_PRIVATE_FILES_DIR=private"
    ```

    On pull, storing `private-files/` is skipped when this is not configured.

Drupito does not accept symbolic links inside a files backup, so a push leaves them out.

## Limits

* Each uploaded backup must be 5 GB or smaller after compression. Limits are subject to change, see Drupito's documentation for the current value.
* API Tokens (and therefore the push & pull commands) are available for sites with a paid hosting plan.

## Debugging

Uncomment `set -x` in a stanza of `.ddev/providers/drupito.yaml` to see each step, or [`ddev ssh`](../usage/commands.md#ssh) into the web container and run the steps by hand. Error messages from the Drupito API are shown verbatim and say what to do next.
