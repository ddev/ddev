#!/bin/bash
# Outputs the correct Docker APT suite (codename) for this distro.
# For Ubuntu and Ubuntu-based distros, trusts UBUNTU_CODENAME/VERSION_CODENAME
# directly — Docker maintains packages for all Ubuntu releases.
# For Debian-based distros (Kali, Parrot, eLxr, etc.) whose VERSION_CODENAME
# is not a valid Docker suite, derives the suite from /etc/debian_version
# (the Debian base release the distro tracks), then falls back to
# /usr/share/distro-info/debian.csv, then to bookworm.

# shellcheck source=/dev/null
. /etc/os-release

CODENAME="${UBUNTU_CODENAME:-$VERSION_CODENAME}"

# For Ubuntu and Ubuntu-based distros, trust the codename from os-release.
# Docker publishes ubuntu-family packages for every Ubuntu release, so any
# codename present in os-release is valid.
# Check both ID and ID_LIKE: Ubuntu itself has ID=ubuntu but ID_LIKE=debian,
# so checking only ID_LIKE would miss it.
if echo "$ID $ID_LIKE" | grep -qi ubuntu; then
    printf '%s' "$CODENAME"
    exit 0
fi

# For pure Debian derivatives (Kali, Parrot, eLxr, etc.) check if the codename
# is already a valid Docker Debian suite.
case "$CODENAME" in
    bookworm|bullseye|buster|trixie|stretch|jessie)
        printf '%s' "$CODENAME"
        exit 0
        ;;
esac

# Not a recognized Debian Docker suite (e.g. Parrot's "echo", eLxr's "aria",
# Kali's "kali-rolling"). Try /etc/debian_version, which on most Debian
# derivatives contains the Debian base release the distro tracks.
if [ -f /etc/debian_version ]; then
    DEBIAN_VER=$(cat /etc/debian_version 2>/dev/null)
    # Strip anything after a slash or non-digit (e.g. "12.5" -> "12",
    # "trixie/sid" -> "trixie", "kali-rolling" -> "kali-rolling").
    DEBIAN_MAJOR=$(printf '%s' "$DEBIAN_VER" | sed -n 's/^\([0-9][0-9]*\).*/\1/p')
    case "$DEBIAN_MAJOR" in
        9)  printf 'stretch';  exit 0 ;;
        10) printf 'buster';   exit 0 ;;
        11) printf 'bullseye'; exit 0 ;;
        12) printf 'bookworm'; exit 0 ;;
        13) printf 'trixie';   exit 0 ;;
    esac
    # /etc/debian_version may contain a codename directly (e.g. "trixie/sid")
    case "$DEBIAN_VER" in
        trixie*)   printf 'trixie';   exit 0 ;;
        bookworm*) printf 'bookworm'; exit 0 ;;
        bullseye*) printf 'bullseye'; exit 0 ;;
        buster*)   printf 'buster';   exit 0 ;;
    esac
fi

# Fall back to distro-info if installed.
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
