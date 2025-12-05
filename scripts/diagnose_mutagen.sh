#!/usr/bin/env bash

# This script tries to diagnose mutagen issues. Please run it in the project
# directory where you're having trouble and provide its
# output in a gist at gist.github.com with any issue you open about mutagen.

ddev list
ddev describe
ddev mutagen version
ddev utility mutagen sync list -l
ddev utility mutagen version
