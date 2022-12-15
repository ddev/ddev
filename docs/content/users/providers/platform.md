# Platform.sh Integration

DDEV provides integration with the [Platform.sh Website Management Platform](https://platform.sh/), which allows Platform.sh users to quickly download and provision a project from Platform.sh in a local DDEV-managed environment.

!!!tip
    Consider using `ddev get platformsh/ddev-platformsh` ([platformsh/ddev-platformsh](https://github.com/platformsh/ddev-platformsh)) for more complete Platform.sh integration.

DDEV’s Platform.sh integration pulls database and files from an existing Platform.sh site/environment into your local system so you can develop locally.

## Platform.sh Global Configuration

You need to obtain and configure an API token first. This is only needed once.

1. Login to the Platform.sh Dashboard and go to *Account* → *API Tokens*. Create an API token DDEV can use.
2. Add the API token to the `web_environment` section in your global DDEV configuration at `~/.ddev/global_config.yaml`:

```yaml
web_environment:
  - PLATFORMSH_CLI_TOKEN=abcdeyourtoken
```

## Platform.sh Per-Project Configuration

1. Check out the site from Platform.sh and configure it with [`ddev config`](../basics/commands.md#config). You’ll want to use [`ddev start`](../basics/commands.md#start) and make sure the basic functionality is working.
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

3. Run [`ddev restart`](../basics/commands.md#restart).
4. Run `ddev pull platform`. After you agree to the prompt, the current upstream database and files will be downloaded.
5. Optionally use `ddev push platform` to push local files and database to Platform.sh. The [`ddev push`](../basics/commands.md#push) command can potentially damage your production site, so we don’t recommend using it.

If you have more than one database on your Platform.sh project, you'll need to choose which one you want to use
as the 'db' primary database on DDEV, and that will be the one pulled or pushed.
Do this by setting PLATFORM_PRIMARY_RELATIONSHIP, for example,

```
ddev config --web-environment-add="PLATFORM_PRIMARY_RELATIONSHIP=main"
```

or run `ddev pull platform` with the `--environment` flag, for example,

```
ddev pull platform --environment="PLATFORM_PRIMARY_RELATIONSHIP=main"
```

## Usage

* `ddev pull platform` will connect to Platform.sh to download database and files. To skip downloading and importing either file or database assets, use the `--skip-files` and `--skip-db` flags.
* If you need to change the `platform.yaml` recipe, you can change it to suit your needs, but remember to remove the `#ddev-generated` line from the top of the file.
