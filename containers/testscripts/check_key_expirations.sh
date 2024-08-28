#!/bin/bash

# Test all keyrings to make sure they are not going to expire
# within DDEV_MAX_DAYS_BEFORE_CERT_EXPIRATION days

set -eu -o pipefail

# Directories containing keyrings
directories=("/etc/apt/trusted.gpg.d/" "/usr/share/keyrings/")
# Today's date in Unix time
today=$(date +%s)
# Days ahead to check for expiration
days_ahead=${DDEV_MAX_DAYS_BEFORE_CERT_EXPIRATION:-90}
printf "Checking key expirations for ${days_ahead} days ahead"
# Seconds ahead (DDEV_MAX_DAYS_BEFORE_CERT_EXPIRATION days)
seconds_ahead=$((days_ahead * 24 * 3600))

# Process each directory
for dir in "${directories[@]}"; do
    if [ ! -d ${dir} ]; then
        echo "Skipping non-existent ${dir}"
        continue
    fi
    echo "Checking directory: $dir"
    cd "$dir"
    shopt -s nullglob
    for keyring in *.{gpg,asc}; do
        # Skip specific keyrings
        if [[ "$keyring" == "debian-archive-removed-keys.gpg" ]]; then
            continue
        fi

        # Prepare keyring path for GPG command
        keyring_path="$dir$keyring"
        # Determine if temporary keyring is needed (for .asc files)
        temp_keyring="/tmp/temp-${keyring%.asc}.gpg"
        if [[ "$keyring" == *.asc ]]; then
            gpg --dearmor < "$keyring_path" > "$temp_keyring" 2>/dev/null
            keyring_to_check="$temp_keyring"
        else
            keyring_to_check="$keyring_path"
        fi

        echo "Checking keyring: $keyring"
        # List keys using GPG, parse with AWK
        gpg --no-default-keyring --keyring "$keyring_to_check" --list-keys --with-colons --fixed-list-mode | \
        awk -F: -v today="$today" -v seconds_ahead="$seconds_ahead" '
            /^pub/ {keyid = $5; expiry_date = $7}
            /^uid/ && keyid {
                # Calculate the time until expiration
                time_to_expire = expiry_date - today;
                # print "Key ID:", keyid, "Name:", $10, "Expires in:", int(time_to_expire / 86400), "days"
                # Check if the key expires within DDEV_MAX_DAYS_BEFORE_CERT_EXPIRATION days
                if (expiry_date != "" && expiry_date > 0 && time_to_expire <= seconds_ahead) {
                    print "Key ID:", keyid, "Name:", $10, "Expires in:", int(time_to_expire / 86400), "days";
                    exit 1;
                }
                keyid = ""; expiry_date = "";
            }
        '
    done
done
