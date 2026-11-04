<?php

// Reports the request host/port environment that PHP receives.
// Used by tests to verify that HTTP_HOST preserves a nonstandard port,
// which apps depend on for building absolute URLs when
// router_http(s)_port is not 80/443.
header('Content-Type: text/plain');
echo "HTTP_HOST=" . ($_SERVER['HTTP_HOST'] ?? '(unset)') . "\n";
echo "SERVER_PORT=" . ($_SERVER['SERVER_PORT'] ?? '(unset)') . "\n";
