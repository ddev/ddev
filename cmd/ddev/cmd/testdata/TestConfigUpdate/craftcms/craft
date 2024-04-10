#!/usr/bin/env php
<?php
/**
 * Craft console bootstrap file
 */

// Load shared bootstrap
require __DIR__ . '/bootstrap.php';

// Load and run Craft
/** @var craft\console\Application $app */
$app = require CRAFT_VENDOR_PATH . '/craftcms/cms/bootstrap/console.php';
$exitCode = $app->run();
exit($exitCode);
