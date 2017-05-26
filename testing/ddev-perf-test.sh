#!/bin/bash

set -o errexit

if [ $(docker ps | grep perf | wc -l) -ne 0 ]; then
	echo "You seem to have running perf sites and may want to kill them off with docker rm -f \$(docker ps -aq)"
	exit 1
fi

BLUE=$(tput setaf 4)
YELLOW=$(tput setaf 3)
RESET=$(tput sgr0)

TIMECMD=gtime
if [ $(which $TIMECMD) = "" ]; then
  echo "please install gnu time with \"brew install gnu-time\"."
  exit 1
fi
TIMEFMT='-f %e'
TIMEIT="$TIMECMD $TIMEFMT -o time.out"
CURLIT='curl -o /dev/null -s -w %{time_total}'
unset DRUD_DEBUG

export folder=/tmp
# Docker can't mount on straight /tmp
if [ $(uname -s) = "Darwin" ] ; then
	folder=/private/tmp
fi
# OSX TMPDIR comes out in /var/... which isn't compatible with mounts on docker by default
folder=$folder/$(date +%s)
mkdir -p $folder && cd $folder


echo "Downloading and prepping cms test candidates into $folder"
drush dl -y drupal-7.54 --drupal-project-rename=d7perf 2>/dev/null
drush dl -y drupal-8.3.2 --drupal-project-rename=d8perf 2>/dev/null
drush dl -y commerce_kickstart-7.x-2.45 --drupal-project-rename=kickperf 2>/dev/null
mkdir -p d7perf/.ddev d8perf/.ddev kickperf/.ddev

for site in "d7perf" "d8perf" "kickperf"; do
    echo "APIVersion: \"1\"" >> "$site"/.ddev/config.yaml
    echo "docroot: \"\"" >> "$site"/.ddev/config.yaml
done

echo "type: drupal7" >> d7perf/.ddev/config.yaml
echo "type: drupal7" >> kickperf/.ddev/config.yaml
echo "type: drupal8" >> d8perf/.ddev/config.yaml

for site in "d7perf" "d8perf" "kickperf"; do
    cd "$folder/$site";
    echo
    echo "Starting $site tests"

    $TIMEIT ddev start >/dev/null 2>&1
    echo "${BLUE}$site: ddev start: $(cat time.out) ${RESET}"

    elapsed=$($CURLIT -o /dev/null -sfL http://"$site".ddev.local)
    echo "${BLUE}$site: curl not-installed site: $elapsed${RESET}"

    if [[ "$site" == "kickperf" ]]; then
        $TIMEIT ddev exec "drush si commerce_kickstart -y --db-url=mysql://root:root@db/data" >/dev/null 2>&1
    else
        $TIMEIT ddev exec "drush si -y --db-url=mysql://root:root@db/data" >/dev/null 2>&1
    fi
    echo "${BLUE}$site: drush site-install: $(cat time.out) ${RESET}"

    elapsed=$($CURLIT -fL http://"$site".ddev.local)
    echo "${BLUE}$site: curl after site install: $elapsed ${RESET}"

    $TIMEIT ddev exec "drush pml" >/dev/null 2>&1
    echo "${BLUE}$site: drush pml: $(cat time.out) ${RESET}"

    echo "Tests completed for $site. Removing..."
    ddev rm -y >/dev/null 2>&1
done
