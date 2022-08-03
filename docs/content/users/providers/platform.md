# Platform.sh Integration

Consider using `ddev get platformsh/ddev-platformsh` ([platformsh/ddev-platformsh](https://github.com/platformsh/ddev-platformsh) for more complete Platform.sh integration.

ddev provides integration with the [Platform.sh Website Management Platform](https://platform.sh/), which allows Platform.sh users to quickly download and provision a project from Platform.sh in a local ddev-managed environment.

ddev's Platform.sh integration pulls database and files from an existing Platform.sh site/environment into your local system so you can develop locally.

## Platform.sh Quickstart

1. Check out the site from platform.sh and then configure it with `ddev config`. You'll want to use `ddev start` and make sure the basic functionality is working.
2. Obtain and configure an API token.
   a. Login to the Platform.sh Dashboard and go to Account->API Tokens to create an API token for ddev to use.
   b. Add the API token to the `web_environment` section in your global ddev configuration at ~/.ddev/global_config.yaml:

   ```yaml
   web_environment:
   - PLATFORMSH_CLI_TOKEN=abcdeyourtoken
   ```

3. Add PLATFORM_PROJECT and PLATFORM_ENVIRONMENT variables to your project `.ddev/config.yaml` or a `.ddev/config.*.yaml`:

```yaml
   web_environment:
   - PLATFORM_PROJECT=nf4amudfn23biyourproject
   - PLATFORM_ENVIRONMENT=main
   ```

You can also do this with `ddev config --web-environment-add="PLATFORM_PROJECT=nf4amudfn23bi,PLATFORM_ENVIRONMENT=main"`
4. `ddev restart`
5. Run `ddev pull platform`. After you agree to the prompt, the current upstream database and files will be downloaded.
6. Optionally use `ddev push platform` to push local files and database to platform.sh. Note that `ddev push` is a command that can potentially damage your production site, so this is not recommended.

## Usage

* `ddev pull platform` will connect to Platform.sh to download database and files. To skip downloading and importing either file or database assets, use the `--skip-files` and `--skip-db` flags.
* If you need to change the `platform.yaml` recipe, you can change it to suit your needs, but remember to remove the "#ddev-generated" line from the top of the file.
