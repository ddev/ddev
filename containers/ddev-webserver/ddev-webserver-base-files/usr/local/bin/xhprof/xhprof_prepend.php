<?php
// The xhprof_prepend.php is the default is nothing is mounted on top of it
// It uses xhgui.
$homeDir = getenv('HOME');
$globalAutoload = $homeDir . '/.composer/vendor/autoload.php';
if (file_exists($globalAutoload)) {
    require_once $globalAutoload;
    // echo "Global autoloader loaded successfully from: $globalAutoload\n";
} else {
    error_log("Global autoloader not found at: $globalAutoload");
}
if (file_exists("/mnt/ddev_config/xhgui/collector/xhgui.collector.php")) {
    require_once "/mnt/ddev_config/xhgui/collector/xhgui.collector.php";
}
