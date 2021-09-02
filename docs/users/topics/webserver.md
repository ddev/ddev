## Webserver-Specific Help and Techniques

### Apache Specifics

#### TLS redirects

It's a common practice to set up HTTP to TLS redirects in the .htaccess file which will lead to issues with the DDEV's proxy setup. The TLS endpoint of a DDEV project will always be the ddev-router container and the requests will be forwarded through plain HTTP to the project's webserver. This of course will end in endless redirects and can never work. Therefor you need to change the root .htaccess so the Apache webserver of the project correctly handles these requests for your local development environment with DDEV. The following snippet should work for the most scenarios and not just DDEV and the existing redirect could simply be replaced by it:

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
