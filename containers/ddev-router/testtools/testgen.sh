#!/bin/bash

# Test ddev.conf generation for https://github.com/ddev/ddev/issues/2648

cpfiles="nginx.tmpl gen-cert-and-nginx-config.sh.tmpl"

for item in $cpfiles; do
  docker cp $item ddev-router:/app
done

docker exec ddev-router docker-gen -only-exposed /app/nginx.tmpl /tmp/ddev.conf

docker exec ddev-router grep "upstream.*-" /tmp/ddev.conf
