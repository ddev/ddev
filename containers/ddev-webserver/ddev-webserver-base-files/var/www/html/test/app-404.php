<?php

// Simulates a framework/CMS route that decides on its own that a page
// doesn't exist and returns its own 404 response with its own body. This
// must be passed through unchanged by the webserver's error_page/
// ErrorDocument handling -- ddev-webserver's own "this 404 came from the
// webserver layer" message must never shadow it. See
// nginx-site-default.conf and apache-site.conf for the reasoning.
header($_SERVER['SERVER_PROTOCOL'] . ' 404 Not Found', true, 404);
echo "App-level not found page";
