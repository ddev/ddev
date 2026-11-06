#!/bin/bash
# PortBindings repro: publish ports three ways, then compare HostConfig.PortBindings (the
# request) with NetworkSettings.Ports (what is bound) on inspect.
docker rm -f pbrepro >/dev/null 2>&1
docker run -d --name pbrepro -p 18080:80 -p 127.0.0.1:18081:81 -p 82 alpine sleep 600 >/dev/null || { echo "run failed"; exit 1; }
echo "-- HostConfig.PortBindings:"
docker inspect pbrepro --format '{{json .HostConfig.PortBindings}}' | jq -cS .
echo "-- NetworkSettings.Ports:"
docker inspect pbrepro --format '{{json .NetworkSettings.Ports}}' | jq -cS .
echo "-- docker port:"
docker port pbrepro | sort
docker rm -f pbrepro >/dev/null
