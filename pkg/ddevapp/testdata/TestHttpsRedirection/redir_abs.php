<?php

$scheme = (!empty($_SERVER['HTTPS']) && $_SERVER['HTTPS'] == "on") ? "https://" : "http://";
$url = $scheme . $_SERVER['HTTP_HOST'] . "/landed.php";
header('Location: ' . $url, true, 302);              // Use either 301 or 302
