# Using DDEV offline, and top-level-domain options

DDEV-Local attempts to make offline use work as well as possible, and you really shouldn't have to do anything to make it work:

* It doesn't attempt instrumentation or update reporting if offline
* It uses /etc/hosts entries instead of DNS resolution if DNS resolution fails

However, it does not (yet) attempt to prevent docker pulls if a new docker image is required, so you'll want to make sure that you try a `ddev start` before going offline to make sure everything has been pulled.

If you have a project running when you're online (using DNS for name resolution) and you then go offline, you'll want to do a `ddev restart` to get the hostname added into /etc/hosts for name resolution.

You have general options as well:

In `.ddev/config.yaml` `use_dns_when_possible: false` will make ddev never try to use DNS for resolution, instead adding hostnames to /etc/hosts. You can also use `ddev config --use-dns-when-possible=false` to set this configuration option.
In `.ddev/config.yaml` `project_tld: example.com` (or any other domain) can set ddev to use a project that could never be looked up in DNS. You can also use `ddev config --project-tld=example.com`

You can also set up a local DNS server like dnsmasq (Linux and macOS, `brew install dnsmasq`) or ([unbound](https://github.com/NLnetLabs/unbound) or many others on Windows) in your own host environment that serves the project_tld that you choose, and DNS resolution will work just fine. You'll likely want a wildcard A record pointing to 127.0.0.1 (on most ddev installations). If you use dnsmasq you must configure it to allow DNS rebinding.

If you're using a browser on Windows, accessing a DDEV project in WSL2, Windows will attempt to resolve the site name via DNS. If you do not have an internet connection, this will fail. To resolve this, update your `C:\Windows\System32\drivers\etc\hosts` file.

```
127.0.0.1 example.ddev.site
```

!!!note "Administrative privileges required"
    You must have administrative privileges to save the Windows hosts file.

* See [Windows Hosts File limited to 10 hosts per IP address line](../../troubleshooting/#windows-hosts-file-limited) for additional troubleshooting.
