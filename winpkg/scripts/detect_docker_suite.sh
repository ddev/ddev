#!/bin/bash
# Outputs the correct Docker APT suite (codename) for this distro.
# For Ubuntu/Debian, uses UBUNTU_CODENAME or VERSION_CODENAME from os-release.
# For Kali and other Debian derivatives with non-standard codenames,
# uses /usr/share/distro-info/debian.csv to find the current Debian stable.

# shellcheck source=/dev/null
. /etc/os-release

CODENAME="${UBUNTU_CODENAME:-$VERSION_CODENAME}"

# If it's already a recognized Docker suite, use it directly
case "$CODENAME" in
    bookworm|bullseye|buster|trixie|stretch|jessie|\
    noble|jammy|focal|bionic|xenial)
        printf '%s' "$CODENAME"
        exit 0
        ;;
esac

# Not a recognized Docker suite (e.g. Kali's "kali-rolling").
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
