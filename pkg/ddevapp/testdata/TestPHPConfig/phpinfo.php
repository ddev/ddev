<?php

print "phpversion=" . phpversion() . "\n";
print "SOMEENV=" . getenv("SOMEENV"). "\n";
print "DOLLAR_SINGLE_QUOTES=" . getenv("DOLLAR_SINGLE_QUOTES"). "\n";
print "DOLLAR_DOUBLE_QUOTES=" . getenv("DOLLAR_DOUBLE_QUOTES"). "\n";
print "DOLLAR_DOUBLE_QUOTES_ESCAPED=" . getenv("DOLLAR_DOUBLE_QUOTES_ESCAPED"). "\n";
print "IS_DDEV_PROJECT=" . getenv("IS_DDEV_PROJECT"). "\n";
print "loaded_extensions=" . implode(",", get_loaded_extensions()) . "\n";
