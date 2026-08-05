<?php

// An app route that returns its own 403 body; ddev-webserver must pass it
// through instead of showing its own 403 explanation.
header($_SERVER['SERVER_PROTOCOL'] . ' 403 Forbidden', true, 403);
echo "App-level forbidden page";
