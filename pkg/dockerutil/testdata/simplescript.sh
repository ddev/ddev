#!/bin/sh

echo "simplescript.sh; TEMPENV=$TEMPENV UID=$(id -u)"
if [ "$ERROROUT" = "true" ] ; then
  exit 5
fi