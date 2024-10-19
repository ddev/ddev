# Using DDEV Offline

!!!warning "Make sure to do a `ddev start` on every project you want to use before going offline"

    To work offline, you have to already have any needed Docker images pulled. That means you should do a `ddev start` on all of the projects before going offline. 

DDEV attempts to work smoothly offline, and you shouldn’t have to do anything to make it work.

However, it cannot pull needed Docker images when offline if a new Docker image is required, so you’ll want to make sure that you try a [`ddev start`](../usage/commands.md#start) before going offline to make sure everything has been pulled.

If you have a project running when you’re online (using DNS for name resolution) and you then go offline, do a [`ddev restart`](../usage/commands.md#restart) to get the hostname added into `/etc/hosts` for name resolution.

You have some general options as well:

In `.ddev/config.yaml`, [`use_dns_when_possible: false`](../configuration/config.md#use_dns_when_possible) will make DDEV never try to use DNS for resolution, instead adding hostnames to `/etc/hosts`. You can also use `ddev config --use-dns-when-possible=false` to set this configuration option.

You can also set up a local DNS server like [dnsmasq](https://dnsmasq.org) (Linux and macOS, `brew install dnsmasq`) or ([unbound](https://github.com/NLnetLabs/unbound) or many others on Windows) in your own host environment that serves the `project_tld` that you choose, and DNS resolution will work fine. You’ll likely want a wildcard A record pointing to `127.0.0.1` on most DDEV installations. If you use dnsmasq, you must configure it to allow DNS rebinding.

!!!note "Administrative Privileges Required"

    If you `ddev start` when offline, DDEV will try to add `<projectname>.ddev.site` to your `/etc/hosts` file. You must have administrative privileges to edit the hosts file on any operating system.

!!!note "Oddities of using `buildx` `docker-container` driver"

    This is an unusual situation, mostly encountered by DDEV developers who have been pushing images, but you may not be able to work offline if your `docker buildx inspect` shows the `docker-container` driver, and you'll need to switch to a builder that has the `docker` driver. `docker buildx ls` will show available drivers, or you can switch to one with `docker buildx create --name docker-driver --driver docker --use`. This is reported in [docker buildx issue](https://github.com/moby/buildkit/issues/5214).
