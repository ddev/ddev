#!/bin/bash
# Multi-homed DNS repro: web is on app and shared; proxy only on shared; client only on app.
cd "$(dirname "$0")" || exit 1
docker compose -p dnsrepro down -t 1 >/dev/null 2>&1
docker compose -p dnsrepro up -d >/dev/null 2>&1 || { echo "compose up failed"; exit 1; }
sleep 5
echo "-- web's addresses"
docker inspect dnsrepro-web-1 | jq -r '.[0].NetworkSettings.Networks | to_entries[] | "\(.key) \(.value.IPAddress)"'
for peer in proxy client; do
  echo "-- from $peer: getent hosts web"
  docker exec dnsrepro-$peer-1 getent hosts web
  echo "-- from $peer: wget http://web/"
  docker exec dnsrepro-$peer-1 timeout 5 wget -q -O /dev/null http://web/ && echo "OK" || echo "FAILED rc=$?"
done
docker compose -p dnsrepro down -t 1 >/dev/null 2>&1
