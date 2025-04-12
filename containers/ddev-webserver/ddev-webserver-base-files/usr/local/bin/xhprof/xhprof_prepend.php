<?php
// This xhprof_prepend.php is the default if nothing is mounted on top of it
// It invokes the xhgui collector by default.
// However, if xhprof_mode=prepend in DDEV the .ddev/xhprof/xhprof_prepend.php will
// be mounted on top of it.
$phpProfilerAutoload = '/usr/local/xhgui.collector/php-profiler/autoload.php';
if (file_exists($phpProfilerAutoload)) {
    require_once $phpProfilerAutoload;
    // echo "php-profiler autoloader loaded successfully from: $phpProfilerAutoload\n";
} else {
    error_log("php-profiler autoloader not found at: $phpProfilerAutoload");
}

$collector_php = "/usr/local/xhgui.collector/xhgui.collector.php";
if (!(bool) ('XHGUI_ENABLED')) {
    # some output?
    return;
}
if (file_exists($collector_php)) {
    require_once $collector_php;
} else {
    error_log("File '$collector_php' not found");
}
