<?php
if (extension_loaded('xhprof') && $_SERVER["SCRIPT_URL"] != "/xhprof/index.php") {
    $xhprof_data = xhprof_disable();
    $appNamespace = "ddev";
    include_once '/var/www/xhprof/xhprof_lib/utils/xhprof_lib.php';
    include_once '/var/www/xhprof/xhprof_lib/utils/xhprof_runs.php';


    $xhprof_runs = new XHProfRuns_Default();
    $run_id =$xhprof_runs->save_run($xhprof_data, $appNamespace);

    $base_link = (isset($_SERVER['HTTPS']) && $_SERVER['HTTPS'] === 'on' ? "https" : "http") . "://$_SERVER[HTTP_HOST]/xhprof/";

    $profiler_url = sprintf('%s/index.php?run=%s&amp;source=%s', $base_link, $run_id, $appNamespace);
    echo '<a href="'. $profiler_url .'" target="_blank">xhprof profiler output</a>';
}
