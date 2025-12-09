# Sharing Your Project

Even though DDEV is intended for local development on a single machine, not as a public server, there are a number of reasons you might want to expose your work in progress more broadly:

* Testing with a mobile device
* Sharing on a local network so that everybody on the local network can see your project
* Some CI applications

There are at least three different ways to share a running DDEV project outside the local developer machine:

* [`ddev share`](../usage/commands.md#share) (using a tunnel provider like ngrok or cloudflared to share over the internet)
* Local name resolution and sharing the project on the local network
* Sharing the HTTP port of the local machine on the local network

## Using `ddev share` (Easiest)

`ddev share` proxies the project via a tunnel provider for sharing your project with others on your team or around the world. DDEV supports multiple providers:

* **ngrok** (default) - Requires an [ngrok.com](https://ngrok.com) account
* **cloudflared** - Free, no account required. Requires [cloudflared](https://developers.cloudflare.com/cloudflare-one/connections/connect-apps/install-and-setup/installation) to be installed
* **Custom providers** - You can add your own providers in `.ddev/share-providers/`

Run `ddev share` to use the default provider, or `ddev share --provider=cloudflared` to use a specific provider. The URL will be displayed and can be shared with collaborators or used on mobile devices.

!!!tip "ngrok in depth"
    Run `ddev share -h` for more, and and read [ngrok’s getting started guide](https://ngrok.com/docs/getting-started/) and [DrupalEasy’s more detailed walkthrough of the `share` command](https://www.drupaleasy.com/blogs/ultimike/2019/06/sharing-your-ddev-local-site-public-url-using-ddev-share-and-ngrok).

CMSes like WordPress and Magento 2 make this a little harder by only responding to a single base URL that’s coded into the database. ngrok allows you to use one static domain for free so you won’t have to frequently change the base URL.

### Setting up a Stable ngrok Domain

1. [Get a free static domain](https://ngrok.com/blog-post/free-static-domains-ngrok-users) from ngrok. Let's say we got `wp23.ngrok-free.app`.
2. Pass the domain to the ngrok args:
    * In `.ddev/config.yaml`, `share_provider_args: --domain wp23.ngrok-free.app` will result in ngrok always using `wp23.ngrok-free.app` as the URL, so it's not changing on you all the time.
    * Alternatively you can pass the domain directly to `ddev share --provider-args="--domain wp23.ngrok-free.app"`

### WordPress: Change the URL with `wp search-replace`

WordPress only has the one base URL, but the `wp` command is built into DDEV’s web container.

This set of steps assumes an ngrok domain of `wp23.ngrok-free.app` and a starting URL of `https://wordpress.ddev.site`.

* Configure `.ddev/config.yaml` to use a custom domain: `share_provider_args: --domain wp23.ngrok-free.app`.
* Make a backup of your database with [`ddev export-db`](../usage/commands.md#export-db) or [`ddev snapshot`](../usage/commands.md#snapshot).
* Edit `wp-config-ddev.php` (or whatever your config is) to change `WP_HOME`, for example, `define('WP_HOME', 'https://wp23.ngrok-free.app');`
* `ddev wp search-replace https://wordpress.ddev.site https://wp23.ngrok-free.app`, assuming your project is configured for `https://wordpress.ddev.site` and your `share_provider_args` are configured for the `wp23.ngrok-free.app` domain.
* Now run [`ddev share`](../usage/commands.md#share).

### Magento2: Change the URL with Magento Tool

This set of steps assumes an ngrok domain `mg2.ngrok-free.app`:

* Configure `.ddev/config.yaml` to use a custom domain with `share_provider_args: --domain mg2.ngrok-free.app`.
* Make a backup of your database.
* Run [`ddev ssh`](../usage/commands.md#ssh).
* Run `bin/magento setup:store-config:set --base-url="https://mg2.ngrok-free.app/`.
* Run [`ddev share`](../usage/commands.md#share) and you'll see your project at `mg2.ngrok-free.app`.

## Using Cloudflared

[Cloudflared](https://developers.cloudflare.com/cloudflare-one/connections/connect-apps/) is a free alternative to ngrok that doesn't require an account. Each tunnel gets a random temporary URL like `https://example-name.trycloudflare.com`.

### Prerequisites

Install cloudflared from [Cloudflare's installation guide](https://developers.cloudflare.com/cloudflare-one/connections/connect-apps/install-and-setup/installation).

### Usage

```bash
# Use cloudflared for a single share session
ddev share --provider=cloudflared

# Set cloudflared as default for all projects
ddev config global --share-default-provider=cloudflared

# Set cloudflared as default for current project only
ddev config --share-default-provider=cloudflared
```

The provider priority is: command-line flag > project config > global config > default (ngrok).

### Cloudflared Configuration

You can configure cloudflared arguments in your `.ddev/config.yaml`:

```yaml
share_provider_args: "--your-args-here"
```

Or pass them on the command line:

```bash
ddev share --provider=cloudflared --provider-args="--your-args-here"
```

### Setting up a Stable Cloudflared Domain

If you have a domain managed by Cloudflare, you can use a named tunnel for a stable, permanent URL instead of the random `trycloudflare.com` URLs.

#### Requirements

* A domain managed by Cloudflare (DNS hosted on Cloudflare)
* cloudflared installed and authenticated: `cloudflared tunnel login`

#### Setup Steps

1. **Create a named tunnel:**

    ```bash
    cloudflared tunnel create my-ddev-tunnel
    ```

    This creates a tunnel and saves credentials in `~/.cloudflared/`.

2. **Add a DNS route for your tunnel:**

    ```bash
    cloudflared tunnel route dns my-ddev-tunnel mysite.example.com
    ```

    This creates a `CNAME` record pointing your subdomain to the tunnel.

3. **Configure DDEV to use the named tunnel:**

    ```bash
    ddev config --share-provider-args="--tunnel my-ddev-tunnel --hostname mysite.example.com"
    ```

    Or add to `.ddev/config.yaml`:

    ```yaml
    share_provider_args: --tunnel my-ddev-tunnel --hostname mysite.example.com
    ```

4. **Share your project:**

    ```bash
    ddev share --provider=cloudflared
    ```

    Or use `--provider-args` to pass the tunnel configuration on the command line:

    ```bash
    ddev share --provider=cloudflared --provider-args="--tunnel my-ddev-tunnel --hostname mysite.example.com"
    ```

    Your project will be available at `https://mysite.example.com`.

#### Example for WordPress or Magento

With a stable URL, you can configure WordPress or Magento to use your custom domain without needing to change the database URL each time:

```bash
# Configure the tunnel
ddev config --share-provider-args="--tunnel my-wp-tunnel --hostname wp.example.com"
ddev config --share-default-provider=cloudflared

# Update WordPress to use the tunnel URL
ddev wp search-replace https://myproject.ddev.site https://wp.example.com

# Start sharing
ddev share
```

!!!tip "Multiple Tunnels"
    You can create multiple named tunnels for different projects. Each tunnel needs a unique name and DNS record.

## Debugging Share Issues

If you're having trouble with `ddev share`, you can enable verbose output to see what's happening:

```bash
DDEV_VERBOSE=true ddev share
```

This will show the full output from the share provider script, including any error messages that might be suppressed in normal mode.

## Using nip.io or Custom Name Resolution Locally

Another solution is to **not** use `*.ddev.site` as your project URLs, but to use DNS that you control and that points to the host machine where your project lives. In general, you’ll want to use HTTP URLs with this approach, because it requires manual configuration of the client machine to get it to trust the development certificate that DDEV uses and configures with `mkcert` on the local machine.

* Use [nip.io](http://nip.io/) to point a domain name to your host. If your computer’s IP address is 192.168.5.101, you can use a domain name like `mysite.192.168.5.101.nip.io` and that domain name will point to your computer. Add that to your project’s `additional_fqdns` with `ddev config --additional-fqdns=mysite.192.168.5.101.nip.io` and [`ddev start`](../usage/commands.md#start). Now people in your internal network should be able to `ping mysite.192.168.5.101.nip.io` if your firewall allows it. (If you have other convenient ways to create a DNS entry for this, you can use those instead of nip.io.)
* Configure `$HOME/.ddev/global_config.yaml` (see [global configuration directory](../usage/architecture.md#global-files)) to bind to all ports: `ddev config global --router-bind-all-interfaces && ddev poweroff && ddev start`.
* Now mobile apps or other computers which are on your **local** network should be able to access your project. Use the HTTP URL rather than the HTTPS URL because computers outside yours don’t know how to trust the developer TLS certificate you’re using. (You can run [`ddev describe`](../usage/commands.md#describe) to see the HTTP URL, but it’s typically the same as the HTTPS URL, but with `http` instead of `https`.)
* Make sure your firewall allows access from your local network to the main interface you’re using. In the example here, you should be able to ping 192.168.5.101 and `curl http://192.168.5.101` and get an answer in each case.
* If you’re using WordPress or Magento 2, you’ll need to change the base URL as described in the [`ddev share`](../usage/commands.md#share) instructions above.

## Exposing a Host Port and Providing a Direct URL

DDEV’s web container also exposes an HTTP port directly, in addition to the normal routing by name and via `ddev_router`. You can expose this port and it may be a useful approach in some situations.

* Configure the project `host_webserver_port` to a known port (that does not conflict with already configured ports). For example, using port 8080, `ddev config --host-webserver-port=8080 --bind-all-interfaces`. This will configure the host-bound port to 8080 and allow it to bind to all network interfaces so colleagues (or hackers) on your local network can access this project’s ports.
* Make sure your firewall allows access to the port on your host machine.
* If you’re using WordPress or Magento 2 you’ll need to change the base URL as described in the [`ddev share`](../usage/commands.md#share) instructions above.
* Each project on your computer must use different ports, or you’ll have port conflicts, and you can’t typically use ports 80 or 443 because `ddev-router` is already using those for normal routing.
* If you don’t want to run `ddev-router` at all, you can omit it globally with `ddev config global --omit-containers=ddev-router`. This is a specialty thing to do when you don’t need the reverse proxy.

Computers and mobile devices on your local network should now be able to access port 8080, on the (example) host address 192.168.5.23, so `http://192.168.5.23:8080` You’ll probably want to use the HTTP URL; your coworker’s browser will not trust the developer TLS certificate you’re using.
