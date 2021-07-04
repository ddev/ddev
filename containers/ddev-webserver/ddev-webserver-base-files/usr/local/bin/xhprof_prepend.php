<?php
$xhprof_link_dir = "/var/www/xhprof/xhprof_html/latest/";
if (extension_loaded('xhprof') && strpos($_SERVER['REQUEST_URI'], '/xhprof') === false) {
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
      text-align: center;
      color: white
    }
    .xhprof-link {
      color: white
    }
    </style>
    <div id=xhprof-footer><a class=xhprof-link target=_blank href='/xhprof/latest/'>Click for latest xhprof run</a> or <a class='xhprof-link' target=_blank href='/xhprof/'>all runs</a></div>

    <script type='text/javascript'>
    function xhprof_footer(l) {
      document.getElementById('xhprof-footer').innerHTML = l;
    }
    </script>

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

    $run_content = sprintf("<div id='xhprof-footer'><a class='xhprof-link' href='%s'>Click for latest xhprof run</a> or <a class='xhprof-link' target=_blank href='/xhprof/'>all runs</a></div>", $profiler_url);
    if (!file_put_contents($GLOBALS['xhprof_link_dir'] . "index.html", $run_content)) {
        exit("Failed to file_put_contents($xhprof_link_dir/index.html)");
    }
    $c = "<a class=xhprof-link href='$profiler_url' target=_blank>Click for this xhprof run</a>  or <a class='xhprof-link' target=_blank href='/xhprof/'>all runs</a>";
    printf('<script>window.onload = xhprof_footer("%s");</script>', $c);
}
