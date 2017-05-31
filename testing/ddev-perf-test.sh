#!/bin/bash

set -o errexit

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
CURLIT='curl  -o /dev/null --fail -sfL -w %{time_total}'
unset DRUD_DEBUG

PROFILE=standard
PROFILE=minimal

export tmpfolder=/tmp
# Docker can't mount on straight /tmp
if [ $(uname -s) = "Darwin" ] ; then
	tmpfolder=/private/tmp
fi
# OSX TMPDIR comes out in /var/... which isn't compatible with mounts on docker by default
cd $tmpfolder

# Parallel arrays to describe features of the sites to test.
# Not using associative arrays because they only appear in bash 4
sites=(d7perf kickperf d8perf)
types=(drupal7 drupal7 drupal8)
downloads=(drupal-7.54 commerce_kickstart-7.x-2.45 drupal-8.3.2)

#sites=(d8perf) types=(drupal8) downloads=(drupal-8.3.2)
for ((i=0; i<${#sites[@]}; ++i)); do
	site=${sites[$i]}


	if [ ! -d "$site" ] ; then
		drush dl -y ${downloads[$i]} --drupal-project-rename=$site 2>/dev/null
	fi

	cd "$tmpfolder/$site";
    echo
    echo "Starting $site tests"

	# Start with fresh cookies file
	cookiefile=$site.cookies.txt
	rm -f $cookiefile

	# Create the config.yml
    mkdir -p .ddev
    cat <<END >.ddev/config.yaml
APIVersion: 1
docroot: ""
type: ${types[$i]}
END

	# Remove any site that is already running, but don't both reporting or worrying about it.
	ddev rm -y >/dev/null 2>&1 || true
	chmod -R ugo+w sites/default && rm -f sites/default/settings.php

    $TIMEIT ddev start >/dev/null 2>&1
    echo "${BLUE}$site: ddev start: $(cat time.out) ${RESET}"

    elapsed=$($CURLIT  http://"$site".ddev.local)
    echo "${BLUE}$site: curl not-installed site: $elapsed${RESET}"

    if [[ "$site" = "kickperf" ]]; then
    	PROFILE=commerce_kickstart
	fi

	$TIMEIT ddev exec "drush si $PROFILE -y --db-url=mysql://root:root@db/data" >/dev/null 2>&1
    echo "${BLUE}$site: drush site-install: $(cat time.out) ${RESET}"

	# Create user adminuser password adminuser, in administrator group
    ddev exec "drush ucrt adminuser --password=adminuser" >/dev/null 2>&1
    ddev exec "drush urol administrator adminuser" >/dev/null 2>&1

    elapsed=$($CURLIT -fL http://"$site".ddev.local)
    echo "${BLUE}$site: curl after site install: $elapsed ${RESET}"

    elapsed=$($CURLIT -fL http://"$site".ddev.local)
    echo "${BLUE}$site: curl again after site install: $elapsed ${RESET}"

	form=user_login
	if [ ${types[i]} = "drupal8" ] ; then
		form=user_login_form
	fi
	elapsed=$($CURLIT -X POST --cookie-jar $cookiefile -H 'content-type: multipart/form-data' -F pass=adminuser -F name=adminuser -F form_id=$form  http://$site.ddev.local/user/login)
    echo "${BLUE}$site: login: $elapsed ${RESET}"

    elapsed=$($CURLIT --cookie $cookiefile http://$site.ddev.local/)
    echo "${BLUE}$site: front authenticated: $elapsed ${RESET}"

    elapsed=$($CURLIT --cookie $cookiefile http://$site.ddev.local/)
    echo "${BLUE}$site: front authenticated again: $elapsed ${RESET}"

    elapsed=$($CURLIT --cookie $cookiefile http://$site.ddev.local/user/1)
    echo "${BLUE}$site: user/1: $elapsed ${RESET}"

    elapsed=$($CURLIT --cookie $cookiefile http://$site.ddev.local/user/1)
    echo "${BLUE}$site: user/1 again: $elapsed ${RESET}"

    elapsed=$($CURLIT --cookie $cookiefile http://$site.ddev.local/admin/modules)
    echo "${BLUE}$site: admin/modules: $elapsed ${RESET}"

    $TIMEIT ddev exec "drush pml" >/dev/null 2>&1
    echo "${BLUE}$site: drush pml: $(cat time.out) ${RESET}"

    echo "Tests completed for $site."
    echo "Removing..."
 	ddev rm -y >/dev/null 2>&1

    cd $tmpfolder
done
