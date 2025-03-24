<?php

#ddev-generated
/**
 * @file
 */

use Xhgui\Profiler\Profiler;

// Add this block inside some bootstrapper or other "early central point in execution".
try {
  $config = require_once __DIR__ . '/xhgui.collector.config.php';

  // The constructor will throw an exception if the environment
  // isn't fit for profiling (extensions missing, other problems)
  $profiler = new Profiler($config);

  // The profiler itself checks whether it should be enabled
  // for request (executes lambda function from config)
  $profiler->start();
}
catch (Exception $e) {
  // Throw away or log error about profiling instantiation failure.
  //
  // An "Unable to create profiler: No suitable profiler found." may mean you need to turn on profiling
  // Eg. `ddev xhprof on`.
}
