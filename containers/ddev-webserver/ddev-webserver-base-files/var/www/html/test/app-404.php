<?php

// An app route that returns its own 404 body; ddev-webserver must pass it
// through instead of showing its own 404 explanation.
header($_SERVER['SERVER_PROTOCOL'] . ' 404 Not Found', true, 404);
echo "App-level not found page";
