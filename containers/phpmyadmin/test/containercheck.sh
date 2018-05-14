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
exit 2
