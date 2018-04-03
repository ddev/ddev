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