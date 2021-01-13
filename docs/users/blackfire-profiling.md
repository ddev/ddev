## Profiling with Blackfire.io

DDEV-Local has built-in [blackfire.io](https://blackfire.io) integration.

### Basic Blackfire Usage

1. Create an account on [blackfire.io](https://blackfire.io)
2. Get the Server ID and the Server Token from your Account->Credentials on blackfire.io
3. Configure ddev with the credentials, `ddev config global --web-environment="BLACKFIRE_SERVER_ID=<id>,BLACKFIRE_SERVER_TOKEN=<token>"`. It's easiest to do this globally, but you can do the same thing with project-level `ddev config`.
4. `ddev start`
5. `ddev blackfire on` to enable, `ddev blackfire off` to disable, `ddev blackfire status` to see status.
6. With blackfire enabled, you can use the [browser extension](https://blackfire.io/docs/profiling-cookbooks/profiling-http-via-browser) or [blackfire cli](https://blackfire.io/docs/profiling-cookbooks/profiling-http-via-cli) to profile.

### Profiling with the Blackfire CLI

The blackfire CLI is built into the web container, but you still have to provide the client id and token.

1. `ddev config global --web-environment="BLACKFIRE_SERVER_ID=<id>,BLACKFIRE_SERVER_TOKEN=<token>,BLACKFIRE_CLIENT_ID=<id>,BLACKFIRE_CLIENT_TOKEN=<token>"`
2. `ddev start`
3. Examples of using the blackfire CLI:

    * `ddev exec blackfire curl https://<yoursite>.ddev.site`
    * `ddev exec blackfire drush st
    * `ddev exec blackfire curl https://<yoursite>.ddev.site`
    * `ddev ssh` and then use the blackfire CLI as described in [Profiling HTTP Requests with the CLI](https://blackfire.io/docs/profiling-cookbooks/profiling-http-via-cli).
