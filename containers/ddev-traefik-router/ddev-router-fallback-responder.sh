#!/bin/sh

# Invoked by socat (see docker-entrypoint.sh) for every connection to the
# catch-all fallback port. Writes a real HTTP 404 response with the
# informative body from unmatched-route.html.
#
# This lives in its own script (rather than inline in socat's SYSTEM:
# address) because socat's address-string parser does its own tokenizing
# of quotes/commas/colons, which mangled an inline multi-line command.

printf 'HTTP/1.1 404 Not Found\r\nContent-Type: text/html; charset=utf-8\r\nConnection: close\r\n\r\n'
cat /var/www/ddev-router-fallback/unmatched-route.html
