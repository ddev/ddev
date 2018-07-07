#!/bin/bash
for i in `seq 1 60`;
do
    # status contains uptime and health in parenthesis, sed to return health
    status="$(docker ps -a --format "{{.Status}}" --filter "name=$CONTAINER_NAME" | sed  's/.*(\(.*\)).*/\1/')"
    if [[ "$status" == "healthy" ]]
    then
        exit 0
    fi
    sleep 2
done
echo "FAIL: ddev-router failed to become ready"
set -x
echo "========== INFO ABOUT FAILURE ============="
docker logs $CONTAINER_NAME
docker ps -a
docker inspect $CONTAINER_NAME
echo "========== END ============="

set +x
exit 1
