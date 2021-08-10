<?php

$uri = "none";
if (!empty($_SERVER) && array_key_exists('REQUEST_URI', $_SERVER)) {
    $uri = $_SERVER['REQUEST_URI'];
}

// Enable xhprof profiling if we're not on an xhprof page
if (extension_loaded('xhprof') && strpos($uri, '/xhprof') === false) {
    xhprof_enable(XHPROF_FLAGS_CPU + XHPROF_FLAGS_MEMORY);
    register_shutdown_function('xhprof_completion');
}

// Write to the xhprof_html output and latest on completion
function xhprof_completion() {
    $xhprof_link_dir = "/var/www/xhprof/xhprof_html/latest/";

    $xhprof_data = xhprof_disable();
    $appNamespace = "ddev";
    include_once '/var/www/xhprof/xhprof_lib/utils/xhprof_lib.php';
    include_once '/var/www/xhprof/xhprof_lib/utils/xhprof_runs.php';

    $xhprof_runs = new XHProfRuns_Default();
    $run_id = $xhprof_runs->save_run($xhprof_data, $appNamespace);
}
