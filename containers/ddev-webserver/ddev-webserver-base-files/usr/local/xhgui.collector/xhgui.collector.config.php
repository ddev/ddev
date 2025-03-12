<?php
#ddev-generated

/**
 * @file
 * Default configuration for Xhgui.
 */

use Xhgui\Profiler\Profiler;

return [
  'save.handler' => Profiler::SAVER_STACK,
  'save.handler.stack' => [
    'savers' => [
      Profiler::SAVER_UPLOAD,
      Profiler::SAVER_FILE,
    ],
    // If saveAll=false, break the chain on successful save.
    'saveAll' => FALSE,
  ],
  'save.handler.file' => [
    // Appends jsonlines formatted data to this path.
    'filename' => '/tmp/xhgui.data.jsonl',
  ],
  'save.handler.upload' => [
    'url' => 'http://xhgui/run/import',
    // The timeout option is in seconds and defaults to 3 if unspecified.
    'timeout' => 3,
    // The token must match 'upload.token' config in XHGui.
    'token' => getenv('XHGUI_UPLOAD_TOKEN', 'token'),
  ],
  // Profile all requests.
  'profiler.enable' => function () {
    return TRUE;
  },

  // 'profiler.simple_url' => function ($url) {
  //   return preg_replace('/\=\d+/', '', $url);
  // },

  'profiler.simple_url' => function ($url) {
    return str_replace(getenv('DDEV_HOSTNAME'), '', $url);
  },

];
