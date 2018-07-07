#!/bin/bash
for i in {1..10}
do
    status="$(docker ps --format "{{.Status}}" --filter "name=$CONTAINER_NAME" | sed  's/.*(\(.*\)).*/\1/')"
    if [[ "$status" == "healthy" ]]
    then
        exit 0
    fi
    sleep 2
done
echo "phpmyadmin container failed to become ready"
set -x
docker ps -a
docker logs $CONTAINER_NAME
exit 2
