<?php
// Prints, as JSON, each variable named in argv[1] as the container environment
// has it, as its compose-interpolated INTERP_ copy has it, and as PHP dotenv
// parsers read it from the application .env next to this script.
require __DIR__ . '/vendor/autoload.php';

$text = file_get_contents(__DIR__ . '/.env');
$parsed = [
    'phpdotenv' => Dotenv\Dotenv::parse($text),
    'symfony' => (new Symfony\Component\Dotenv\Dotenv())->parse($text),
];
$out = [];
foreach (json_decode($argv[1], true) as $name) {
    $out[$name] = ['env' => getenv($name), 'interpolated' => getenv('INTERP_' . $name)];
    foreach ($parsed as $parser => $values) {
        $out[$name][$parser] = $values[$name] ?? null;
    }
}
echo json_encode($out);
