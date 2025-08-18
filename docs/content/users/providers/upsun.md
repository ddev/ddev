# Upsun Integration

DDEV provides integration with the [Upsun by Platform](https://upsun.com/) hosting system, allowing Upsun users to easily download database and files from Upsun to a local DDEV-managed environment.

DDEV’s Upsun integration pulls databases and files from an existing Upsun site/environment into your local system so you can develop locally.

## Upsun Global Configuration

You need to obtain and configure an API token first. This only needs to be done once.

1. Login to the Upsun Dashboard and go to *My Profile* → *API Tokens*. Create an API token DDEV can use.
2. Add the API token to the `web_environment` section in your global DDEV configuration at `~/.ddev/global_config.yaml`:

    ```yaml
    web_environment:
        - UPSUN_CLI_TOKEN=abcdeyourtoken
    ```

    You can also do this with:

    ```bash
    ddev config global --web-environment-add="UPSUN_CLI_TOKEN=abcdeyourtoken"
    ```

    !!!tip "What if I have more than one API token?"
        To use multiple API tokens for different projects, add them to your per-project configuration using the [`.ddev/config.local.yaml`](../configuration/config.md#environmental-overrides) file instead. This file is gitignored by default.

        ```yaml
        web_environment:
            - UPSUN_CLI_TOKEN=abcdeyourtoken
        ```

## Upsun Per-Project Configuration

1. Check out the Upsun site and configure it with [`ddev config`](../usage/commands.md#config). You’ll want to use [`ddev start`](../usage/commands.md#start) and make sure the basic functionality is working.
2. Add `PLATFORM_PROJECT` and `PLATFORM_ENVIRONMENT` variables to your project.

    * Either in `.ddev/config.yaml` or a `.ddev/config.*.yaml` file:

        ```yaml
        web_environment:
            - PLATFORM_PROJECT=nf4amudfn23biyourproject
            - PLATFORM_ENVIRONMENT=main
        ```

    * Or with a command from your terminal:

        ```bash
        ddev config --web-environment-add="PLATFORM_PROJECT=nf4amudfn23bi,PLATFORM_ENVIRONMENT=main"
        ```

3. Run [`ddev restart`](../usage/commands.md#restart).
4. Run `ddev pull upsun`. After you agree to the prompt, the current upstream databases and files will be downloaded.
5. Optionally use `ddev push upsun` to push local files and database to Upsun. The [`ddev push`](../usage/commands.md#push) command can potentially damage your production site, so we don’t recommend using it.

### Managing Multiple Apps

If your environment contains more than one app, add `PLATFORM_APP` variable to your project:

* Either in `.ddev/config.yaml` or a `.ddev/config.*.yaml` file:

    ```yaml
    web_environment:
        - ...
        - PLATFORM_APP=app
    ```

* Or with a command from your terminal:

    ```bash
    ddev config --web-environment-add="PLATFORM_APP=app"
    ```

### Managing Multiple Databases

If your project has only one database, it will automatically be pulled into and pushed from DDEV’s `'db'` database.

If your project has multiple databases, they’ll all be pulled into DDEV with their respective remote names. You can optionally designate a *primary* to use DDEV’s default `'db'` database, which may be useful in some cases—particularly if you’ve been using the default solo-database behavior and happened to add another one to your project.

You can designate the primary database using the `PLATFORM_PRIMARY_RELATIONSHIP` environment variable:

```bash
ddev config --web-environment-add="PLATFORM_PRIMARY_RELATIONSHIP=main"
```

You can also do the same thing by running `ddev pull upsun` and using the `--environment` flag:

```bash
ddev pull upsun --environment="PLATFORM_PRIMARY_RELATIONSHIP=main"
```

## Usage

* `ddev pull upsun` will connect to Upsun to download database and files. To skip downloading and importing either file or database assets, use the `--skip-files` and `--skip-db` flags.
* If you need to change the `upsun.yaml` recipe, you can change it to suit your needs, but remember to remove the `#ddev-generated` line from the top of the file.
