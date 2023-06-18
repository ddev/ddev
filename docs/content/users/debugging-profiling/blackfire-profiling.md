# Blackfire Profiling

DDEV has built-in [Blackfire](https://www.blackfire.io/) integration.

## Basic Blackfire Usage (Using Browser Plugin)

1. Create a [Blackfire](https://www.blackfire.io/) account. (Free Blackfire accounts are no longer available; see [Blackfire pricing](https://www.blackfire.io/pricing/).)
2. Install the Chrome or Firefox [browser plugin](https://blackfire.io/docs/profiling-cookbooks/profiling-http-via-browser).
3. From Blackfire’s control panel, get the Server ID, Server Token, Client ID, and Client Token from your Account → Credentials or environment.
4. Configure DDEV with the credentials, `ddev config global --web-environment-add="BLACKFIRE_SERVER_ID=<id>,BLACKFIRE_SERVER_TOKEN=<token>,BLACKFIRE_CLIENT_ID=<id>,BLACKFIRE_CLIENT_TOKEN=<token>"`. It’s easiest to do this globally, but you can do the same thing at the project level using `ddev config --web-environment-add`. (It may be easier to manually edit the relevant config file, `.ddev/config.yaml` or `~/.ddev/global_config.yaml`.)
5. [`ddev start`](../usage/commands.md#start).
6. [`ddev blackfire on`](../usage/commands.md#blackfire) to enable, `ddev blackfire off` to disable, `ddev blackfire status` to see status.
7. With Blackfire enabled, you can use the [browser extension](https://blackfire.io/docs/profiling-cookbooks/profiling-http-via-browser).

## Profiling with the Blackfire CLI

The Blackfire CLI is built into the web container, so you don’t need to install it yourself.

1. Add the `BLACKFIRE_SERVER_ID`, `BLACKFIRE_SERVER_TOKEN`, `BLACKFIRE_CLIENT_ID`, and `BLACKFIRE_CLIENT_TOKEN` environment variables to `~/.ddev/global_config.yaml` via the `web_environment` key:

    ```yaml
      web_environment:
      - OTHER_ENV=something
      - BLACKFIRE_SERVER_ID=dde5f66d-xxxxxx
      - BLACKFIRE_SERVER_TOKEN=09b0ec-xxxxx
      - BLACKFIRE_CLIENT_ID=f5e88b7e-xxxxx
      - BLACKFIRE_CLIENT_TOKEN=00cee15-xxxxx1
    ```

    You can also do this with `ddev config global --web-environment-add="BLACKFIRE_SERVER_ID=<id>,BLACKFIRE_SERVER_TOKEN=<token>,BLACKFIRE_CLIENT_ID=<id>,BLACKFIRE_CLIENT_TOKEN=<token>"`, but any existing environment variables will be deleted.
  
2. [`ddev start`](../usage/commands.md#start).
3. `ddev blackfire on`.
4. Click the “Blackfire” browser extension to profile.

## Examples of Using the Blackfire CLI

* `ddev blackfire on` and `ddev blackfire off`
* [`ddev exec blackfire curl https://<yoursite>.ddev.site`](../usage/commands.md#exec)
* `ddev exec blackfire drush st`
* `ddev exec blackfire curl https://<yoursite>.ddev.site`
* [`ddev ssh`](../usage/commands.md#ssh) and use the Blackfire CLI as described in [Profiling HTTP Requests with the CLI](https://blackfire.io/docs/profiling-cookbooks/profiling-http-via-cli).
