#!/bin/bash
docker daemon > /dev/null 2>&1 &

eval exec $1 "${@:2}"