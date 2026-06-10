#!/bin/bash
# Outputs the correct Docker APT suite (codename) for this distro.
# For Ubuntu and Ubuntu-based distros, trusts UBUNTU_CODENAME/VERSION_CODENAME
# directly — Docker maintains packages for all Ubuntu releases.
# For Kali, Parrot, and other pure Debian derivatives whose VERSION_CODENAME
# is not a valid Docker suite, uses /usr/share/distro-info/debian.csv to find
# the current Debian stable release codename.

# shellcheck source=/dev/null
. /etc/os-release

CODENAME="${UBUNTU_CODENAME:-$VERSION_CODENAME}"

# For Ubuntu and Ubuntu-based distros, trust the codename from os-release.
# Docker publishes ubuntu-family packages for every Ubuntu release, so any
# codename present in os-release is valid.
if echo "${ID_LIKE:-$ID}" | grep -qi ubuntu; then
    printf '%s' "$CODENAME"
    exit 0
fi

# For pure Debian derivatives (Kali, Parrot, etc.) check if the codename
# is already a valid Docker Debian suite.
case "$CODENAME" in
    bookworm|bullseye|buster|trixie|stretch|jessie)
        printf '%s' "$CODENAME"
        exit 0
        ;;
esac

# Not a recognized Debian Docker suite (e.g. Kali's "kali-rolling").
# Find the current Debian stable release from distro-info.
if [ -f /usr/share/distro-info/debian.csv ]; then
    today=$(date +%Y%m%d)
    FOUND=""
    while IFS=, read -r _ver _codename series _created release eol _rest; do
        [ "$_ver" = "version" ] && continue
        if [ -z "$release" ] || [ -z "$eol" ]; then continue; fi
        r="${release//-/}"
        e="${eol//-/}"
        if [ "$today" -ge "$r" ] && [ "$today" -lt "$e" ]; then
            FOUND="$series"
            break
        fi
    done < /usr/share/distro-info/debian.csv
    if [ -n "$FOUND" ]; then
        printf '%s' "$FOUND"
        exit 0
    fi
fi

# Ultimate fallback
printf 'bookworm'
