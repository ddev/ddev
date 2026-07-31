<?php

// Apache's DirectoryIndex fallback from conf-enabled/403.conf: emits a real 403
// (a static document would be a 200) and serves the same page nginx serves from
// common.d/403.conf.
header($_SERVER['SERVER_PROTOCOL'] . ' 403 Forbidden', true, 403);
header('Content-Type: text/html; charset=utf-8');
header('X-Ddev-403-Source: ddev-webserver (apache)');

readfile(__DIR__ . '/ddev-webserver-403-error.html');
