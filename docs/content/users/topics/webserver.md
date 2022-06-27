# Webserver-Specific Help and Techniques

## Apache Specifics

### TLS redirects

It's a common practice to set up HTTP to TLS redirects in the .htaccess file, which leads to issues with the DDEV proxy setup. The TLS endpoint of a DDEV project is always the `ddev-router` container and requests are forwarded through plain HTTP to the project's webserver. This of course results in endless redirects and can never work. Therefore, you need to change the root `.htaccess` file so the Apache webserver of the project correctly handles these requests for your local development environment with DDEV. The following snippet should work for most scenarios and not just DDEV and could replace an existing redirect:

```apache
# http:// -> https:// plain or behind proxy for Apache 2.2 and 2.4
# behind proxy
RewriteCond %{HTTP:X-FORWARDED-PROTO} ^http$
RewriteRule (.*) https://%{HTTP_HOST}/$1 [R=301,L]

# plain
RewriteCond %{HTTP:X-FORWARDED-PROTO} ^$
RewriteCond %{REQUEST_SCHEME} ^http$ [NC,OR]
RewriteCond %{HTTPS} off
RewriteRule (.*) https://%{HTTP_HOST}/$1 [R=301,L]
```
