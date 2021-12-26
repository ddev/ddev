## Sharing your project with others

Even though DDEV is intended for local development on a single machine, not as a public server, there are a number of reasons you might want to expose your work in progress more broadly:

* Testing with a mobile device
* Sharing on a local network so that everybody on the local network can see your project
* Some CI applications

There are at least three different ways to share a running DDEV-Local project outside the local developer machine:

* `ddev share` (using ngrok to share over the internet)
* Local name resolution and sharing the project on the local network
* Sharing just the http port of the local machine on the local network

### Using `ddev share` to share project (easiest)

`ddev share` proxies the project via [ngrok](https://ngrok.com), and it's by far the easiest way to solve the problem of sharing your project with others on your team or around the world. It's built into ddev and "just works" for most people, but it does require a free or paid account on [ngrok.com](https://ngrok.com). All you do is run `ddev share` and then give the resultant URL to your collaborator or use it on your mobile device. [Read the basic how-to from DrupalEasy](https://www.drupaleasy.com/blogs/ultimike/2019/06/sharing-your-ddev-local-site-public-url-using-ddev-share-and-ngrok) or run `ddev share -h` for more.

There are CMSs that make this a little harder, especially WordPress and Magento 2. Both of those only respond to a single base URL, and that URL is coded into the database, so it makes this a little harder. For both of these I recommend paying ngrok the $5/month for  a [basic plan](https://ngrok.com/pricing) so you can use a stable subdomain with ngrok.</p>

#### Setting up a stable subdomain with ngrok

1. Get a paid token with at least the basic plan, and configure it. It will be in ~/.ngrok2/ngrok.yml as authtoken.
2. Configure ngrok_args to use a stable subdomain. In `.ddev/config.yaml`, `ngrok_args: --subdomain wp23` will result in ngrok always using "wp23.ngrok.io" as the URL, so it's not changing on you all the time.

#### WordPress: Change the URL with wp search-replace

WordPress only has the one base URL, but the wp command is built into DDEV-Local's web container.

This set of steps assumes an ngrok subdomain of "wp23" and a starting URL of `https://wordpress.ddev.site`.

* Configure .ddev/config.yaml to use a custom subdomain: `ngrok_args: --subdomain wp23`
* Make a backup of your database with `ddev export-db` or `ddev shapshot`
* Edit wp-config-ddev.php (or whatever your config is) to change WP_HOME, for example, `define('WP_HOME', 'https://wp23.ngrok.io');`
* `ddev wp search-replace https://wordpress.ddev.site https://wp23.ngrok.io` (assuming your project is configured for `https://wordpress.ddev.site` and your `ngrok_args` are configured for the wp23 subdomain)
* Now `ddev share`

#### Magento2: Change the URL with magento tool

This set of steps assumes an ngrok subdomain "mg2"

* Configure `.ddev/config.yaml` to use a custom subdomain: `ngrok_args: --subdomain mg2`
* Make a backup of your database.
* Edit your `.ddev/config.yaml`
* `ddev ssh` and
* `bin/magento setup:store-config:set --base-url="https://mg2.ngrok.io/`
* `ddev share` and you'll see your project on `mg2.ngrok.io`

### Using nip.io and or your own name resolution and open up to the local network

Another solution is to **not** use `*.ddev.site` as your project URLs, but to use DNS that you control (and that points to the host machine where your project lives). In general, you'll want to use http URLs with this approach, because it requires manual configuration of the client machine to get it to trust the development certificate that ddev uses (and configures with mkcert on the local machine).

* Use [nip.io](http://nip.io/) to point a domain name to your host.  If your computer's IP address is 192.168.5.101, you can use a domain name like `mysite.192.168.5.101.nip.io` and that domain name will point to your computer. Now add that as an additional_fqdn to your project, `ddev config --additional-fqdns=mysite.192.168.5.101.nip.io` and `ddev start`. Now people in your internal network should be able to `ping mysite.192.168.5.101.nip.io` if your firewall allows it. (Note that if you have other convenient ways to create a DNS entry for this, you can use those instead of using nip.io.)
* Configure `~/.ddev/global_config.yaml` to bind to all ports: `ddev config global --router-bind-all-interfaces && ddev poweroff && ddev start`
* Now mobile apps or other computers which are on your **local** network should be able to access your project. Use the http URL rather than the https URL because computers outside yours don't know how to trust the developer TLS certificate you're using. (You can use `ddev describe` to see the http URL, but it's typically the same as the https URL, but with "http" instead of "https".)
* Make sure your firewall allows access from your local network to the main interface you're using. In the example here you should be able to ping 192.168.5.101 and `curl http://192.168.5.101` and get an answer in each case.
* If you're using WordPress or Magento 2 you'll need to change the base URL as described in the `ddev share` instructions above.

### Exposing just a port from the host and providing a direct URL

DDEV's web container also exposes an HTTP port directly (in addition to the normal routing by name and via ddev_router). You can expose this port and it may be a useful approach in some situations.

* Configure the project `host_webserver_port` to a known port (that does not conflict with already configured ports). For example, using port 8080, `ddev config --host-webserver-port=8080 --bind-all-interfaces`. This will configure the host-bound port to 8080 and allow it to bind to all network interfaces so colleagues (or hackers) on your local network can access this project's ports.
* Make sure your firewall allows access to the port on your host machine.
* If you're using WordPress or Magento 2 you'll need to change the base URL as described in the `ddev share` instructions above.
* Each project on your computer must use different ports or you'll have port conflicts, and you can't typically use ports 80 or 443 because ddev-router is already using those for normal routing.
* If you don't want to run ddev-router at all you can actually omit it globally, `ddev config global --omit-containers=ddev-router`. This is a specialty thing to do, when you don't need the reverse proxy at all for anything, as for [DrupalPod](https://github.com/shaal/DrupalPod) or other [GitPod](https://www.gitpod.io/) applications.

Computers and mobile devices on your local network should now be able to access port 8080, on the (example) host address 192.168.5.23, so `http://192.168.5.23:8080` You'll probably want to use the http URL; your coworker's browser will not trust the developer TLS certificate you're using.
