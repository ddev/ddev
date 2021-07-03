<?php
$xhprof_link_dir = "/var/www/xhprof/xhprof_html/latest/";
if (extension_loaded('xhprof')) {
    xhprof_enable();

    if (!is_dir($xhprof_link_dir) && !mkdir($xhprof_link_dir, 0777, true)) {
      exit("failed to create $xhprof_link_dir");
    }

    print("
    <style>
    #xhprof-footer {
      position: fixed;
      left: 0;
      bottom: 0;
      width: 100%;
      background-color: blue;
      color: white;
      text-align: center;
    }
    </style>
    <div id=xhprof-footer><a id=xhprof-link target=_blank href='/xhprof/latest/'>Click for latest xhprof run</a></div>
    ");

    register_shutdown_function('xhprof_completion');
}

function xhprof_completion() {
    $xhprof_data = xhprof_disable();
    $appNamespace = "ddev";
    include_once '/var/www/xhprof/xhprof_lib/utils/xhprof_lib.php';
    include_once '/var/www/xhprof/xhprof_lib/utils/xhprof_runs.php';


    $xhprof_runs = new XHProfRuns_Default();
    $run_id = $xhprof_runs->save_run($xhprof_data, $appNamespace);

    $base_link = (isset($_SERVER['HTTPS']) && $_SERVER['HTTPS'] === 'on' ? "https" : "http") . "://$_SERVER[HTTP_HOST]/xhprof/";

    $profiler_url = sprintf('%sindex.php?run=%s&amp;source=%s', $base_link, $run_id, $appNamespace);

    $run_content = sprintf("<div class='explanation'><div class='xhprof-run'><a href='%s' target=_blank>Click for latest xhprof run</a></div>", $profiler_url);
    if (!file_put_contents($GLOBALS['xhprof_link_dir'] . "index.html", $run_content)) {
        exit("Failed to file_put_contents($xhprof_link_dir/index.html)");
    }
    $c = "<a id=xhprof-link href='/xhprof/latest/'>Click for latest xhprof run</a>";
    printf('<script>window.onload = xhprof_footer("%s");</script>', $c);
}
