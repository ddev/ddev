# Sharing Your Project

Even though DDEV is intended for local development on a single machine, not as a public server, there are a number of reasons you might want to expose your work in progress more broadly:

* Testing with a mobile device
* Sharing on a local network so that everybody on the local network can see your project
* Some CI applications

There are at least three different ways to share a running DDEV project outside the local developer machine:

* [`ddev share`](../usage/commands.md#share) (using ngrok to share over the internet)
* Local name resolution and sharing the project on the local network
* Sharing the HTTP port of the local machine on the local network

## Using `ddev share` (Easiest)

`ddev share` proxies the project via [ngrok](https://ngrok.com) for sharing your project with others on your team or around the world. It’s built into DDEV and requires a free or paid [ngrok.com](https://ngrok.com) account. Run `ddev share` and then give the resultant URL to your collaborator or use it on your mobile device. [Read the basic how-to from DrupalEasy](https://www.drupaleasy.com/blogs/ultimike/2019/06/sharing-your-ddev-local-site-public-url-using-ddev-share-and-ngrok) or run `ddev share -h` for more.

CMSes like WordPress and Magento 2 make this a little harder by only responding to a single base URL that’s coded into the database. ngrok’s $8/month [personal plan](https://ngrok.com/pricing) allows you to use a persistent subdomain so you won’t have to frequently change the base URL.

### Setting up a Stable ngrok Subdomain

1. Get a paid token with at least the basic plan, and configure it. It will be in `~/.ngrok2/ngrok.yml` as `authtoken`.
2. Configure `ngrok_args` to use a stable subdomain. In `.ddev/config.yaml`, `ngrok_args: --subdomain wp23` will result in ngrok always using `wp23.ngrok.io` as the URL, so it’s not changing on you all the time.

### WordPress: Change the URL with `wp search-replace`

WordPress only has the one base URL, but the `wp` command is built into DDEV’s web container.

This set of steps assumes an ngrok subdomain of `wp23` and a starting URL of `https://wordpress.ddev.site`.

* Configure `.ddev/config.yaml` to use a custom subdomain: `ngrok_args: --subdomain wp23`.
* Make a backup of your database with [`ddev export-db`](../usage/commands.md#export-db) or [`ddev shapshot`](../usage/commands.md#snapshot).
* Edit `wp-config-ddev.php` (or whatever your config is) to change `WP_HOME`, for example, `define('WP_HOME', 'https://wp23.ngrok.io');`
* `ddev wp search-replace https://wordpress.ddev.site https://wp23.ngrok.io`, assuming your project is configured for `https://wordpress.ddev.site` and your `ngrok_args` are configured for the wp23 subdomain.
* Now run [`ddev share`](../usage/commands.md#share).

### Magento2: Change the URL with Magento Tool

This set of steps assumes an ngrok subdomain `mg2`:

* Configure `.ddev/config.yaml` to use a custom subdomain with `ngrok_args: --subdomain mg2`.
* Make a backup of your database.
* Edit your `.ddev/config.yaml`.
* Run [`ddev ssh`](../usage/commands.md#ssh).
* Run `bin/magento setup:store-config:set --base-url="https://mg2.ngrok.io/`.
* Run [`ddev share`](../usage/commands.md#share) and you’ll see your project at `mg2.ngrok.io`.

## Using nip.io or Custom Name Resolution Locally

Another solution is to **not** use `*.ddev.site` as your project URLs, but to use DNS that you control and that points to the host machine where your project lives. In general, you’ll want to use HTTP URLs with this approach, because it requires manual configuration of the client machine to get it to trust the development certificate that DDEV uses and configures with `mkcert` on the local machine.

* Use [nip.io](http://nip.io/) to point a domain name to your host. If your computer’s IP address is 192.168.5.101, you can use a domain name like `mysite.192.168.5.101.nip.io` and that domain name will point to your computer. Add that to your project’s `additional_fqdns` with `ddev config --additional-fqdns=mysite.192.168.5.101.nip.io` and [`ddev start`](../usage/commands.md#start). Now people in your internal network should be able to `ping mysite.192.168.5.101.nip.io` if your firewall allows it. (If you have other convenient ways to create a DNS entry for this, you can use those instead of nip.io.)
* Configure `~/.ddev/global_config.yaml` to bind to all ports: `ddev config global --router-bind-all-interfaces && ddev poweroff && ddev start`.
* Now mobile apps or other computers which are on your **local** network should be able to access your project. Use the HTTP URL rather than the HTTPS URL because computers outside yours don’t know how to trust the developer TLS certificate you’re using. (You can run [`ddev describe`](../usage/commands.md#describe) to see the HTTP URL, but it’s typically the same as the HTTPS URL, but with “http” instead of “https”.)
* Make sure your firewall allows access from your local network to the main interface you’re using. In the example here, you should be able to ping 192.168.5.101 and `curl http://192.168.5.101` and get an answer in each case.
* If you’re using WordPress or Magento 2, you’ll need to change the base URL as described in the [`ddev share`](../usage/commands.md#share) instructions above.

## Exposing a Host Port and Providing a Direct URL

DDEV’s web container also exposes an HTTP port directly, in addition to the normal routing by name and via `ddev_router`. You can expose this port and it may be a useful approach in some situations.

* Configure the project `host_webserver_port` to a known port (that does not conflict with already configured ports). For example, using port 8080, `ddev config --host-webserver-port=8080 --bind-all-interfaces`. This will configure the host-bound port to 8080 and allow it to bind to all network interfaces so colleagues (or hackers) on your local network can access this project’s ports.
* Make sure your firewall allows access to the port on your host machine.
* If you’re using WordPress or Magento 2 you’ll need to change the base URL as described in the [`ddev share`](../usage/commands.md#share) instructions above.
* Each project on your computer must use different ports or you’ll have port conflicts, and you can’t typically use ports 80 or 443 because `ddev-router` is already using those for normal routing.
* If you don’t want to run `ddev-router` at all, you can omit it globally with `ddev config global --omit-containers=ddev-router`. This is a specialty thing to do when you don’t need the reverse proxy, as for [DrupalPod](https://github.com/shaal/DrupalPod) or other [Gitpod](https://www.gitpod.io/) applications.

Computers and mobile devices on your local network should now be able to access port 8080, on the (example) host address 192.168.5.23, so `http://192.168.5.23:8080` You’ll probably want to use the HTTP URL; your coworker’s browser will not trust the developer TLS certificate you’re using.
