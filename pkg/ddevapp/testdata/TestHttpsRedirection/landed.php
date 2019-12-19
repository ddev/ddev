<?php

echo "You landed at ${_SERVER['REQUEST_URI']}<br/>";
echo "HTTPS is {$_SERVER['HTTPS']}<br/>";

echo 'You can go to <a href="redir_abs.php">redir_abs.php</a> or to <a href="redir_relative.php">redir_relative.php</a><br/>';


if (!empty($_SERVER['HTTPS']) && $_SERVER["HTTPS"] == "on") {
    echo "You can  <a href='http://${_SERVER['HTTP_HOST']}/landed.php'>switch from https to http</a><br/>";
} else {
    echo "You can <a href='https://${_SERVER['HTTP_HOST']}/landed.php'>switch from http to https</a><br/>";
}
