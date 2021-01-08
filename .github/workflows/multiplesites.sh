#!/bin/bash

# Attempt to recreate multiple-site-creation failure from
# https://github.com/drud/ddev/issues/2648
# Copied from https://gist.github.com/rfay/378a377aa4f695799ee2b36e2847c2e6

# This is NOT INTENDED to live on, it's really just for a little while in v1.16.2-alpha*
# and then should be removed

set -eu -o pipefail
sitedir=~/tmp/testsites
mkdir -p ${sitedir}
logdir=~/tmp/testlogs
mkdir -p ${logdir}
sites=""

function cleanup {
  printf "\n\nClean up with:\n"
  printf "ddev delete -Oy ${sites} && rm -rf ${sitedir} ${logdir}\n"
}
trap cleanup EXIT

function captureState {
  printf "Capturing state after failure on ${site} ( https://${hostname} )\n"
  ddev list > ${logdir}/ddevlist.txt
  docker ps -a > ${logdir}/dockerps.txt
  docker inspect ddev-${site}-web > ${logdir}/webinspect.txt
  docker inspect ddev-router > ${logdir}/routerinspect.txt
  curl  -s --unix-socket /var/run/docker.sock "http://foo/containers/json" >${logdir}/docker_api_containers.txt
  docker logs ddev-router > ${logdir}/routerLogs.txt 2>&1
  docker cp ddev-router:/etc/nginx/conf.d/default.conf ${logdir}
  docker cp ddev-router:/gen-cert-and-nginx-config.sh ${logdir}
  ddev logs > ${logdir}/weblogs.txt
  tarball=~/tmp/testfailurelogs.$(date '+%Y%m%d%H%M%S').tgz
  tar -czf ${tarball} ${logdir} ${dir} ~/.ddev/.*.yaml ~/.ddev/global_config.yaml
  echo "Full state tarball at ${tarball}"
}

uname -a
ddev --version
docker --version
docker-compose --version

#ddev poweroff
for i in {1..5}; do
  site=testsite-${i}
  sites="$sites $site"
  printf "\n\nStarting ${site}...\n"
  hostname=${site}.ddev.site
  dir=${sitedir}/${site}
  mkdir -p ${dir} && pushd ${dir} >/dev/null
  ddev config --project-type=php
  echo $site >index.html
  ddev start
  curl --fail -slL https://${hostname} || ( printf "\ncurl of https://${hostname} failed\n" && captureState && exit $i )
  popd >/dev/null
done

echo "All project starts succeeded."
