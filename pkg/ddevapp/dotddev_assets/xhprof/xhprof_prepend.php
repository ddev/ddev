<?php

// #ddev-generated
// If you want to take over and customize this file, remove the line above
// And check this file in.

// This file is used by `ddev xhprof on` and determines the behavior
// of xhprof when it is enabled. It is mounted into the ddev-webserver container
// as /usr/local/bin/xhprof/xhprof_prepend.php

// It can be customized for particular sites or for particular CMS versions.
// Some suggestions and examples are provided below.

$uri = "none";
if (!empty($_SERVER) && array_key_exists('REQUEST_URI', $_SERVER)) {
  $uri = $_SERVER['REQUEST_URI'];
}

// Enable xhprof profiling if we're not on an xhprof page
if (extension_loaded('xhprof') && strpos($uri, '/xhprof') === false) {
  // If this is too much information, just use xhprof_enable(), which shows CPU only
  xhprof_enable(XHPROF_FLAGS_CPU + XHPROF_FLAGS_MEMORY);
  register_shutdown_function('xhprof_completion');
}

// Write to the xhprof_html output and latest on completion
function xhprof_completion()
{
  $xhprof_link_dir = "/var/xhprof/xhprof_html/latest/";

  $xhprof_data = xhprof_disable();
  $appNamespace = "ddev";
  include_once '/var/xhprof/xhprof_lib/utils/xhprof_lib.php';
  include_once '/var/xhprof/xhprof_lib/utils/xhprof_runs.php';

  $xhprof_runs = new XHProfRuns_Default();
  $run_id = $xhprof_runs->save_run($xhprof_data, $appNamespace);

  // Uncomment to append profile link to the page (and remove the ddev generated first line)
  // append_profile_link($run_id, $appNamespace);
}

// If invoked, this will append a profile link to the output HTML
// This works on some CMSs, like Drupal 7. It does not work on Drupal8/9
// and can have unwanted side-effects on TYPO3
function append_profile_link($run_id, $appNamespace)
{
  $base_link = (isset($_SERVER['HTTPS']) && $_SERVER['HTTPS'] === 'on' ? "https" : "http") . "://$_SERVER[HTTP_HOST]/xhprof/";

  $profiler_url = sprintf('%sindex.php?run=%s&amp;source=%s', $base_link, $run_id, $appNamespace);
  echo '<div id="xhprof"><a href="' . $profiler_url . '" target="_blank">xhprof profiler output</a></div>';
}
