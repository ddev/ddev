<?php

// PHP 8.4+ deprecated E_USER_ERROR with trigger_error()
if (PHP_VERSION_ID >= 80400) {
    exit("Fatal error");
} else {
    trigger_error("Fatal error", E_USER_ERROR);
}
