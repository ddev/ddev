#!/bin/bash

set -o errexit

GREEN=$(tput setaf 2)
YELLOW=$(tput setaf 3)
RESET=$(tput sgr0)

TIMECMD=gtime
if [ $(which $TIMECMD) = "" ]; then
  echo "please install gnu time with \"brew install gnu-time\"."
  exit 1
fi
TIMEFMT='-f %e'
TIMEIT="$TIMECMD $TIMEFMT"
CURLIT='curl -o /dev/null -s -w %{time_total}'
unset DRUD_DEBUG

# Docker can't mount on straight /tmp
if [ $(uname -s) = "Darwin" ] ; then
	export TMPDIR=/private/tmp
fi

# Download into a temp folder
folder=$(mktemp -d)


echo "Downloading and prepping cms test candidates into $folder"
drush dl -y drupal-7.54 --drupal-project-rename=d7perf 2>/dev/null
drush dl -y drupal-8.3.2 --drupal-project-rename=d8perf 2>/dev/null
drush dl -y commerce_kickstart-7.x-2.45 --drupal-project-rename=kickperf 2>/dev/null
mkdir d7perf/.ddev d8perf/.ddev kickperf/.ddev

for site in "d7perf" "d8perf" "kickperf"; do
    echo "APIVersion: \"1\"" >> "$site"/.ddev/config.yaml
    echo "docroot: \"\"" >> "$site"/.ddev/config.yaml
done

echo "type: drupal7" >> d7perf/.ddev/config.yaml
echo "type: drupal7" >> kickperf/.ddev/config.yaml
echo "type: drupal8" >> d8perf/.ddev/config.yaml

for site in "d7perf" "d8perf" "kickperf"; do
    cd "$site";
    echo "Starting $site tests"

    echo "${YELLOW}Running ddev start${RESET}"
    $TIMEIT ddev start 2>/dev/null
    echo "${GREEN}Completed ddev start${RESET}"

    echo "${YELLOW}Running curl not-installed site${RESET}"
    $CURLIT -o /dev/null -sfL http://"$site".ddev.local
    echo "${GREEN}Completed curl not-installed site${RESET}"

    echo "${YELLOW}Running drush site-install${RESET}"
    $TIMEIT ddev exec "drush si -y --db-url=mysql://root:root@db/data" 2>/dev/null
    echo "${GREEN}Completed drush site-install${RESET}"

    echo "${YELLOW}Running drush site-install${RESET}"
    if [[ "$site" == "kickperf" ]]; then
        $TIMEIT ddev exec "drush si commerce_kickstart -y --db-url=mysql://root:root@db/data" 2>/dev/null
    else
        $TIMEIT ddev exec "drush si -y --db-url=mysql://root:root@db/data" 2>/dev/null
    fi
    echo "${GREEN}Completed drush site-install${RESET}"

    echo "${YELLOW}Running curl after site install${RESET}"
    $CURLIT -fL http://"$site".ddev.local
    echo "${GREEN}Completed curl after site install${RESET}"

    echo "${YELLOW}Running drush pml${RESET}"
    $TIMEIT ddev exec "drush pml" 2>/dev/null
    echo "${GREEN}Completed drush pml${RESET}"

    echo "Tests completed for $site. Removing..."
    ddev rm -y
    cd -
done
