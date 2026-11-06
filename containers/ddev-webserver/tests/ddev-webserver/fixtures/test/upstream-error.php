<?php

header($_SERVER['SERVER_PROTOCOL'] . ' 500 Internal Server Error', true, 500);
echo "Upstream error message";
