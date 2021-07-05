<?php

$uri = "none";
if (!empty($_SERVER) && array_key_exists('REQUEST_URI', $_SERVER)) {
    $uri = $_SERVER['REQUEST_URI'];
}

// Enable xhprof profiling if we're not on an xhprof page
if (extension_loaded('xhprof') && strpos($uri, '/xhprof') === false) {
    xhprof_enable();
    register_shutdown_function('xhprof_completion');
}

// Write to the xhprof_html output and latest on completion
function xhprof_completion() {
    $xhprof_link_dir = "/var/www/xhprof/xhprof_html/latest/";

    $xhprof_data = xhprof_disable();
    $appNamespace = "ddev";
    include_once '/var/www/xhprof/xhprof_lib/utils/xhprof_lib.php';
    include_once '/var/www/xhprof/xhprof_lib/utils/xhprof_runs.php';

    if (!is_dir($xhprof_link_dir) && !mkdir($xhprof_link_dir, 0777, true)) {
      exit("failed to create $xhprof_link_dir");
    }

    $xhprof_runs = new XHProfRuns_Default();
    $run_id = $xhprof_runs->save_run($xhprof_data, $appNamespace);

    $base_link = (isset($_SERVER['HTTPS']) && $_SERVER['HTTPS'] === 'on' ? "https" : "http") . "://$_SERVER[HTTP_HOST]/xhprof/";

    $run_url = sprintf('%s/xhprof/index.php?run=%s&amp;source=%s', getenv('DDEV_PRIMARY_URL') , $run_id, $appNamespace);

    $run_content = sprintf("<div id='xhprof-footer'><a class='xhprof-link' href='%s'>Click for latest xhprof run</a> or <a class='xhprof-link' target=_blank href='/xhprof/'>all runs</a></div>", $run_url);
    if (!file_put_contents($xhprof_link_dir . "index.html", $run_content)) {
        exit("Failed to file_put_contents($xhprof_link_dir/index.html)");
    }
}
