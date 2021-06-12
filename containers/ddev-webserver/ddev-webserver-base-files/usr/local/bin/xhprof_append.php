<?php
$xhprof_data = xhprof_disable();
$appNamespace = getenv("DDEV_HOSTNAME");
include_once '/var/www/xhprof/xhprof_lib/utils/xhprof_lib.php';
include_once '/var/www/xhprof/xhprof_lib/utils/xhprof_runs.php';

$xhprof_runs = new XHProfRuns_Default();
$xhprof_runs->save_run($xhprof_data, $appNamespace);
