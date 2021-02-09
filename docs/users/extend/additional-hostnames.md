## Additional Project Hostnames

Add additional hostnames to a project in the project's .ddev/config.yaml:

```
name: mysite

additional_hostnames:
- "extraname"
- "fr.mysite"
- "es.mysite"
- "it.mysite"
- "\*.lotsofnames"
```

This configuration would result in working hostnames of mysite.ddev.site, extraname.ddev.site, fr.mysite.ddev.site, es.mysite.ddev.site, and it.mysite.ddev.site (with full http and https URLs for each).

In addition, the wildcard `*.lotsofnames` will result in anything `*.lotsofnames.ddev.site` being recognized by the project. This works only if you're connected to the internet, using "ddev.site" for your top-level-domain, and using DNS for name lookups. (These are all the defaults.)

**Although we recommend extreme care with this feature**, you can also provide additional_fqdn entries, which don't use the ".ddev.site" top-level domain.  **This feature populates your hosts file with entries which may hide the real DNS entries on the internet, causing way too much head-scratching.**

**If you use a FQDN which is resolvable on the internet, you must use `use_dns_when_possible: false` or configure that with `ddev config --use-dns-when-possible=false`.**

```
name: somename

additional_fqdns:
- example.com
- somesite.example.com
- anothersite.example.com
```

This configuration would result in working FQDNs of somename.ddev.site, example.com, somesite.example.com, and anothersite.example.com.

**Note**: If you see ddev-router status become unhealthy in `ddev list`, it's most often a result of trying to use conflicting FQDNs in more than one project. "example.com" can only be assigned to one project, or it will break ddev-router.

**Warning**: this may not work predictably on all systems. There are operating systems and machines where /etc/hosts may not be the first or only resolution technique, especially if the additional_fqdn you use is also in DNS.

**Warning**: if you use an additional_fqdn that exists on the internet (like "www.google.com"), your hosts file will override access to the original (internet) site, and you'll be sad and confused that you can't get to it.
