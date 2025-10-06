<?php

// PHP 8.4+ deprecated E_USER_ERROR with trigger_error()
// For PHP 8.4+, write directly to stderr to match trigger_error behavior
// For older versions, use trigger_error with E_USER_ERROR
if (PHP_VERSION_ID >= 80400) {
    $stderr = fopen('php://stderr', 'w');
    fwrite($stderr, "PHP Fatal error:  Fatal error in " . __FILE__ . " on line " . __LINE__ . "\n");
    fclose($stderr);
    http_response_code(500);
    exit(1);
} else {
    trigger_error("Fatal error", E_USER_ERROR);
}
