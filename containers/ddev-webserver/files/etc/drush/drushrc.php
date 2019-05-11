<?php
if (!empty($_ENV['DDEV_URL'])) {
    $options['uri'] = $_ENV['DDEV_URL'];
}
