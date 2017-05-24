#!/bin/bash
GREEN=$(tput setaf 2)
YELLOW=$(tput setaf 3)
RESET=$(tput sgr0)

echo "Downloading and prepping cms test candidates"
drush dl -y drupal-7.54 --drupal-project-rename=d7perf
drush dl -y drupal-8.3.2 --drupal-project-rename=d8perf
drush dl -y commerce_kickstart-7.x-2.45 --drupal-project-rename=kickperf
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
    time ddev start
    echo "${GREEN}Completed ddev start${RESET}"

    echo "${YELLOW}Running curl not-installed site${RESET}"
    time curl -fL http://"$site".ddev.local > /dev/null
    echo "${GREEN}Completed curl not-installed site${RESET}"

    echo "${YELLOW}Running drush site-install${RESET}"
    time ddev exec "drush si -y --db-url=mysql://root:root@db/data"
    echo "${GREEN}Completed drush site-install${RESET}"

    echo "${YELLOW}Running drush site-install${RESET}"
    if [[ "$site" == "kickperf" ]]; then
        time ddev exec "drush si commerce_kickstart -y --db-url=mysql://root:root@db/data"
    else
        time ddev exec "drush si -y --db-url=mysql://root:root@db/data"
    fi
    echo "${GREEN}Completed drush site-install${RESET}"

    echo "${YELLOW}Running curl after site install${RESET}"
    time curl -fL http://"$site".ddev.local > /dev/null
    echo "${GREEN}Completed curl after site install${RESET}"

    echo "${YELLOW}Running drush pml${RESET}"
    time ddev exec "drush pml"
    echo "${GREEN}Completed drush pml${RESET}"

    echo "Tests completed for $site. Removing..."
    ddev rm -y
    cd -
done
