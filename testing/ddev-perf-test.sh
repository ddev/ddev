#!/bin/bash


BLUE=$(tput setaf 4)
YELLOW=$(tput setaf 3)
RESET=$(tput sgr0)

TIMECMD=gtime
command -v $TIMECMD >/dev/null 2>&1 || { echo >&2 "please install gnu time with \"brew install gnu-time\"."; exit 1; }
TIMEFMT='-f %e'
TIMEIT="$TIMECMD $TIMEFMT -o time.out"
CURLIT='curl  -o /dev/null --fail -sfL -w %{time_total}'
unset DDEV_DEBUG

PROFILE=standard

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

# Alternate approach for testing just one site
#sites=(d8perf) types=(drupal8) downloads=(drupal-8.3.2)
#sites=(d7perf) types=(drupal7) downloads=(drupal-7.54)

for ((i=0; i<${#sites[@]}; ++i)); do
	site=${sites[$i]}
	base_url=http://$site.ddev.local


	if [ ! -d "$site" ] ; then
		drush dl -y ${downloads[$i]} --drupal-project-rename=$site > /dev/null 2>&1
	fi

	PROFILE=standard

	cd "$tmpfolder/$site";
    echo
    echo "Starting $site tests using profile $PROFILE"

	# Start with fresh cookies file
	cookiefile=$site.cookies.txt
	rm -f $cookiefile
	log=$site.log.txt
	rm -f $log

	# Create the config.yml
    mkdir -p .ddev
    cat <<END >.ddev/config.yaml
APIVersion: 1
docroot: ""
type: ${types[$i]}
END

	# Remove any site that is already running, but don't both reporting or worrying about it.
	ddev rm -y >>$log 2>&1 || true
	chmod -R ugo+w sites/default && rm -f sites/default/settings.php

    $TIMEIT ddev start >$log 2>&1
    echo "${BLUE}$site: ddev start: $(cat time.out) ${RESET}"

    elapsed=$($CURLIT  $base_url)
    echo "${BLUE}$site: curl not-installed site ($?): $elapsed${RESET}"

	# Normally use the commerce_kickstart profile unless profile is set to "minimal" for testing speedup.
    if [[ "$site" = "kickperf" && "$PROFILE" != "minimal" ]]; then
    	PROFILE=commerce_kickstart
	fi

	# drush si - if it fails we continue to next site.
	$TIMEIT ddev exec drush si $PROFILE -y --db-url=mysql://db:db@db/db >>$log 2>&1 || (echo "Failed drush si" && continue)
    echo "${BLUE}$site: drush site-install: $(cat time.out) ${RESET}"

    elapsed=$($CURLIT -fL $base_url)
    echo "${BLUE}$site: anon curl ($?) after site install: $elapsed ${RESET}"

    elapsed=$($CURLIT -fL $base_url)
    echo "${BLUE}$site: anon curl ($?) again after site install: $elapsed ${RESET}"

	$TIMEIT ddev exec drush uli -l $site.ddev.local >$site.uli.txt 2>&1 || (echo "Failed drush uli" && continue)
	echo "${BLUE}$site: ddev uli: $(cat time.out) ${RESET}"

	# This technique doesn't work on d8 due to some new form fields on the uli page.
#	uli_url=$(cat $site.uli.txt)
#	elapsed=$($CURLIT -X POST --cookie-jar $cookiefile -H 'content-type: multipart/form-data' -F form_id=user_pass_reset  $uli_url)
#    echo "${BLUE}$site: uli login $uli_url ($?): $elapsed ${RESET}"

	# Create user adminuser password adminuser, in administrator group
	# Note that this doesn't work with the 'minimal' profile since there is no 'administrator' role created.
    ddev exec drush ucrt adminuser --password=adminuser >>$log 2>&1 || echo "Failed drush ucrt"
    ddev exec drush urol administrator adminuser >>$log 2>&1 || echo "Failed drush urol"

	form=user_login
	if [ ${types[i]} = "drupal8" ] ; then
		   form=user_login_form
	fi
	elapsed=$($CURLIT -X POST --cookie-jar $cookiefile -H 'content-type: multipart/form-data' -F pass=adminuser -F name=adminuser -F form_id=$form  $base_url/user/login)
    echo "${BLUE}$site: login ($?): $elapsed ${RESET}"

    elapsed=$($CURLIT --cookie $cookiefile $base_url)
    echo "${BLUE}$site: front authenticated ($?): $elapsed ${RESET}"

    elapsed=$($CURLIT --cookie $cookiefile $base_url)
    echo "${BLUE}$site: front authenticated again ($?): $elapsed ${RESET}"

    elapsed=$($CURLIT --cookie $cookiefile $base_url/user/1)
    echo "${BLUE}$site: user/1 ($?): $elapsed ${RESET}"

    elapsed=$($CURLIT --cookie $cookiefile $base_url/user/1)
    echo "${BLUE}$site: user/1 again ($?): $elapsed ${RESET}"

    elapsed=$($CURLIT --cookie $cookiefile $base_url/admin/modules)
    echo "${BLUE}$site: admin/modules ($?): $elapsed ${RESET}"

    $TIMEIT ddev exec drush pml >>$log 2>&1 || echo "Failed drush pml"
    echo "${BLUE}$site: drush pml: $(cat time.out) ${RESET}"

    echo "Tests completed for $site."
#   echo "Removing..."
# 	ddev rm -y >>$log 2>&1

    cd $tmpfolder
done
