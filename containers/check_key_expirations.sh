#!/bin/bash

# Directories containing keyrings
directories=("/etc/apt/trusted.gpg.d/" "/usr/share/keyrings/")
# Today's date in Unix time
today=$(date +%s)
# Days ahead to check for expiration
days_ahead=${DDEV_MAX_DAYS_BEFORE_CERT_EXPIRATION:-90}
# Seconds ahead (60 days)
seconds_ahead=$((days_ahead * 24 * 3600))

# Process each directory
for dir in "${directories[@]}"; do
    echo "Checking directory: $dir"
    cd "$dir"
    for keyring in *.gpg *.asc; do
        # Skip specific keyrings
        if [[ "$keyring" == "debian-archive-removed-keys.gpg" ]]; then
            continue
        fi

        # Prepare keyring path for GPG command
        keyring_path="$dir$keyring"
        # Determine if temporary keyring is needed (for .asc files)
        if [[ "$keyring" == *.asc ]]; then
            temp_keyring="/tmp/temp-${keyring%.asc}.gpg"
            gpg --dearmor < "$keyring_path" > "$temp_keyring"
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
                # Check if the key expires within 60 days
                if (expiry_date != "" && expiry_date > 0 && time_to_expire <= seconds_ahead) {
                    print "Key ID:", keyid, "Name:", $10, "Expires in:", int(time_to_expire / 86400), "days"
                }
                keyid = ""; expiry_date = "";
            }
        '
        # Clean up temporary file if it was created
        if [[ "$keyring" == *.asc ]]; then
            rm "$temp_keyring"
        fi
    done
done
