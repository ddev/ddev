<h1> Additional Project Hostnames</h1>

Add additional hostnames to a project in the project's .ddev/config.yaml:

```
name: mysite

additional_hostnames:
- extraname
- fr.mysite
- es.mysite
- it.mysite
```

This configuration would result in working hostnames of mysite.ddev.local, extraname.ddev.local, fr.mysite.ddev.local, es.mysite.ddev.local, and it.mysite.ddev.local (with full http and https URLs for each).

You can also provide additional_fqdn entries, which don't use the ".ddev.local" top-level domain. 

```
name: somename

additional_fqdns:
- mysite.com
- somesite.yoursite.com
- anothersite.yoursite.com
```

This configuration would result in working FQDNs of somename.ddev.local, mysite.com, somesite.yoursite.com http://anothersite.yoursite.com, and anothersite.yoursite.com.

**Warning**: this may not work predictably on all systems. There are operating systems and machines where /etc/hosts may not be the first or only resolution technique, especially if the additional_fqdn you use is also in DNS.

**Warning**: if you use an additional_fqdn that exists on the internet (like "www.google.com"), your hosts file will override access to the original (internet) site, and you'll be sad and confused that you can't get to it.