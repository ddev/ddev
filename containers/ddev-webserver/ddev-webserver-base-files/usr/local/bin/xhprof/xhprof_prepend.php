<?php
// This xhprof_prepend.php is the default if nothing is mounted on top of it
// It invokes the xhgui collector by default.
// However, if xhprof_mode=prepend in DDEV the .ddev/xhprof/xhprof_prepend.php will
// be mounted on top of it.
$homeDir = getenv('HOME');
$globalAutoload = $homeDir . '/.composer/vendor/autoload.php';
if (file_exists($globalAutoload)) {
    require_once $globalAutoload;
    // echo "Global autoloader loaded successfully from: $globalAutoload\n";
} else {
    error_log("Global autoloader not found at: $globalAutoload");
}

$collector_php = "/usr/local/xhgui.collector/xhgui.collector.php";
if (file_exists($collector_php)) {
    require_once $collector_php;
} else {
    error_log("File '$collector_php' not found");
}
