#!/bin/bash

## SETUP REQUIREMENTS:
##
## 1) Run "kbox create" to create 2 drupal7 sites and one drupal8 site,
## naming them d7perf, kickperf, d8perf respectively.
## 2) Overwrite contents of kickperf/code with a commerce_kickstart install.
## 3) Run "kbox poweroff" to stop all running containers.
## 4) Run this script above the 3 site directories.

GREEN=$(tput setaf 2)
YELLOW=$(tput setaf 3)
RESET=$(tput sgr0)

for site in "d7perf" "d8perf" "kickperf"; do
    cd "$site";
    echo "Starting $site tests"

    echo "${YELLOW}Running kbox start${RESET}"
    time kbox start
    echo "${GREEN}Completed kbox start${RESET}"

    echo "${YELLOW}Running curl not-installed site${RESET}"
    time curl -fL http://"$site".kbox.site > /dev/null
    echo "${GREEN}Completed curl not-installed site${RESET}"

    echo "${YELLOW}Running drush site-install${RESET}"
    if [[ "$site" == "kickperf" ]]; then
        time kbox drush si commerce_kickstart -y --db-url=mysql://drupal:drupal@database/drupal
    else
        time kbox drush si -y --db-url=mysql://drupal:drupal@database/drupal
    fi
    echo "${GREEN}Completed drush site-install${RESET}"

    echo "${YELLOW}Running curl after site install${RESET}"
    time curl -fL http://"$site".kbox.site > /dev/null
    echo "${GREEN}Completed curl after site install${RESET}"

    echo "${YELLOW}Running drush pml${RESET}"
    time kbox drush pml
    echo "${GREEN}Completed drush pml${RESET}"

    echo "Tests completed for $site. Stopping..."
    kbox stop
    cd -
done
