#!/bin/bash

# This script tries to diagnose mutagen issues. Please run it in the project
# directory where you're having trouble and provide its
# output in a gist at gist.github.com with any issue you open about mutagen.

export MUTAGEN_DATA_DIRECTORY=~/.ddev_mutagen_data_directory

ddev list
ddev describe
~/.ddev/bin/mutagen sync list -l
~/.ddev/bin/mutagen version
