<?php
/* DDEV xhgui configuration */

$DDEV_DATABASE_FAMILY = getenv('DDEV_DATABASE_FAMILY', 'mysql');
$XHGUI_PDO_DSN = $DDEV_DATABASE_FAMILY === 'postgres'
    ? 'pgsql:host=db;dbname=xhgui'
    : "$DDEV_DATABASE_FAMILY:host=db;dbname=xhgui";

return [
    // Always 'pdo'.
    'save.handler' => getenv('XHGUI_SAVE_HANDLER') ?: 'pdo',

    // Database options for PDO.
    'pdo' => [
        'dsn' => $XHGUI_PDO_DSN,
        'user' => getenv('XHGUI_PDO_USER') ?: 'db',
        'pass' => getenv('XHGUI_PDO_PASS') ?: 'db',
        'table' => getenv('XHGUI_PDO_TABLE') ?: 'results',
        'tableWatch' => getenv('XHGUI_PDO_TABLE_WATCHES') ?: 'watches',
    ],

    'run.view.filter.names' => [
        'Zend*',
        'Composer*',
    ],

    // If defined, add imports via upload (/run/import) must pass token parameter with this value
    'upload.token' => getenv('XHGUI_UPLOAD_TOKEN') ?: '',

    // Add this path prefix to all links and resources
    // If this is not defined, auto-detection will try to find it itself
    // Example:
    // - prefix=null: use auto-detection from request
    // - prefix='': use '' for prefix
    // - prefix='/xhgui': use '/xhgui'
    'path.prefix' => null,

    // Setup timezone for date formatting
    // Example: 'UTC', 'Europe/Tallinn'
    // If left empty, php default will be used (php.ini or compiled in default)
    'timezone' => '',

    // Date format used when browsing XHGui pages.
    //
    // Must be a format supported by the PHP date() function.
    // See <https://secure.php.net/date>.
    'date.format' => 'M jS H:i:s',

    // The number of items to show in "Top lists" with functions
    // using the most time or memory resources, on XHGui Run pages.
    'detail.count' => 6,

    // The number of items to show per page, on XHGui list pages.
    'page.limit' => 25,
];
