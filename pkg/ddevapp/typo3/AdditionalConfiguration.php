<?php

/**
 * #ddev-generated: Automatically generated TYPO3 additional.php file.
 * ddev manages this file and may delete or overwrite the file unless this comment is removed.
 * It is recommended that you leave this file alone.
 */

if (getenv('IS_DDEV_PROJECT') == 'true') {
    $ddevConfig = [
        // This GFX configuration allows processing by installed ImageMagick 6
        'GFX' => [
            'processor' => 'ImageMagick',
            'processor_path' => '/usr/bin/',
            'processor_path_lzw' => '/usr/bin/',
        ],
        // This mail configuration sends all emails to mailpit
        'MAIL' => [
            'transport' => 'smtp',
            'transport_smtp_encrypt' => false,
            'transport_smtp_server' => 'localhost:1025',
        ],
        'SYS' => [
            'trustedHostsPattern' => '.*.*',
            'devIPmask' => '*',
            'displayErrors' => 1,
        ],
    ];
{{if .HasDBContainer}}
    // Only override the database connection if the project uses a driver
    // provided by the DDEV db container. If the Default connection has been
    // configured for anything else (like SQLite), leave it alone.
    $ddevDriver = $GLOBALS['TYPO3_CONF_VARS']['DB']['Connections']['Default']['driver'] ?? '{{ .DBDriver }}';
    if (in_array($ddevDriver, ['mysqli', 'pdo_mysql', 'pdo_pgsql'], true)) {
        $ddevConfig['DB'] = [
            'Connections' => [
                'Default' => [
                    'dbname' => 'db',
                    'driver' => '{{ .DBDriver }}',
                    'host' => '{{ .DBHostname }}',
                    'password' => 'db',
                    'port' => {{ .DBPort }},
                    'user' => 'db',
                ],
            ],
        ];
    }
{{end}}
    $GLOBALS['TYPO3_CONF_VARS'] = array_replace_recursive(
        $GLOBALS['TYPO3_CONF_VARS'],
        $ddevConfig
    );
    unset($ddevConfig{{if .HasDBContainer}}, $ddevDriver{{end}});
}
