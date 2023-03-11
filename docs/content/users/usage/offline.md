# Using DDEV Offline

DDEV attempts to work smoothly offline, and you shouldn’t have to do anything to make it work:

* It doesn’t attempt instrumentation or update reporting if offline
* It falls back to using `/etc/hosts` entries if DNS resolution fails

However, it does not (yet) attempt to prevent Docker pulls if a new Docker image is required, so you’ll want to make sure that you try a [`ddev start`](../usage/commands.md#start) before going offline to make sure everything has been pulled.

If you have a project running when you’re online (using DNS for name resolution) and you then go offline, you’ll want to do a [`ddev restart`](../usage/commands.md#restart) to get the hostname added into `/etc/hosts` for name resolution.

You have general options as well:

In `.ddev/config.yaml`, [`use_dns_when_possible: false`](../configuration/config.md#use_dns_when_possible) will make DDEV never try to use DNS for resolution, instead adding hostnames to `/etc/hosts`. You can also use `ddev config --use-dns-when-possible=false` to set this configuration option.

In `.ddev/config.yaml`, you can use `project_tld: example.com` to have DDEV use a project TLD that won’t be looked up via DNS. You can do the equivalent with `ddev config --project-tld=example.com`. This also works as a global option in `~/.ddev/global_config.yaml` or running `ddev config global --project-tld=example.com`.

You can also set up a local DNS server like [dnsmasq](https://dnsmasq.org) (Linux and macOS, `brew install dnsmasq`) or ([unbound](https://github.com/NLnetLabs/unbound) or many others on Windows) in your own host environment that serves the project_tld that you choose, and DNS resolution will work fine. You’ll likely want a wildcard A record pointing to 127.0.0.1 on most DDEV installations. If you use dnsmasq, you must configure it to allow DNS rebinding.

If you’re using a browser on Windows and accessing a DDEV project in WSL2, Windows will attempt to resolve the site name via DNS. This will fail if you don’t have an internet connection. To resolve this, update your `C:\Windows\System32\drivers\etc\hosts` file manually:

```
127.0.0.1 example.ddev.site
```

!!!note "Administrative Privileges Required"

    You must have administrative privileges to save the hosts file on any OS.
