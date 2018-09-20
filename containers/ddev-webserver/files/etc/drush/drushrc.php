<?php
if (!empty($_ENV['DDEV_URL'])) {
    $options['uri'] = $_ENV['DDEV_URL'];
}
# Skip confirmations since `ddev exec` cannot support interactive prompts
if (!empty($_ENV['DDEV_EXEC'])) {
    $options['yes'] = 1;
}
