<?php

print "phpversion=" . phpversion() . "\n";
print "SOMEENV=" . getenv("SOMEENV"). "\n";
print "IS_DDEV_PROJECT=" . getenv("IS_DDEV_PROJECT"). "\n";
print "loaded_extensions=" . implode(",", get_loaded_extensions()) . "\n";
