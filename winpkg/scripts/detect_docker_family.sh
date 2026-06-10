#!/bin/bash
# Outputs "ubuntu" or "debian" to indicate which Docker CE APT repository
# family to use. Ubuntu and Ubuntu-based distros use the ubuntu repo;
# all others (Debian, Kali, Parrot, etc.) use the debian repo.

# shellcheck source=/dev/null
. /etc/os-release

if echo "$ID $ID_LIKE" | grep -qi ubuntu; then
    printf 'ubuntu'
else
    printf 'debian'
fi
