#!/usr/bin/env bash

# Test all keyrings to make sure they are not going to expire
# within DDEV_MAX_DAYS_BEFORE_CERT_EXPIRATION days

set -eu -o pipefail

# Detect whether apt-get update supports --audit
apt_supports_audit() {
  # First word after 'apt' is the version, e.g. "apt 3.0.3 (amd64)"
  local ver
  ver="$(apt-get --version 2>/dev/null | awk 'NR==1 {print $2}')"

  # Bookworm: 2.6.x  → no --audit
  # Trixie/testing/sid: 2.9.x / 3.x → has --audit
  dpkg --compare-versions "$ver" ge "2.9.7"
}

# Directories containing keyrings
directories=("/etc/apt/trusted.gpg.d/" "/usr/share/keyrings/")
# Today's date in Unix time
today=$(date +%s)
# Days ahead to check for expiration
days_ahead=${DDEV_MAX_DAYS_BEFORE_CERT_EXPIRATION:-90}
printf "Checking key expirations for ${days_ahead} days ahead\n"
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
        if [[ "$keyring" =~ -removed-keys\.gpg$ ]]; then
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
        gpg --no-default-keyring --keyring "$keyring_to_check" --list-keys --with-colons --fixed-list-mode | \
        awk -F: -v today="$today" -v seconds_ahead="$seconds_ahead" '
            /^pub/ {keyid = $5; expiry_date = $7}
            /^uid/ && keyid {
                # Calculate the time until expiration only if the key has an expiration date
                if (expiry_date == 0 || expiry_date == "") {
                    print "Key ID:", keyid, "Name:", $10, "has no expiration date"
                } else if (expiry_date > 0) {
                    time_to_expire = expiry_date - today;
                    print "Key ID:", keyid, "Name:", $10, "Expires in:", int(time_to_expire / 86400), "days"
                    # Check if the key expires within DDEV_MAX_DAYS_BEFORE_CERT_EXPIRATION days
                    if (expiry_date != "" && expiry_date > 0 && time_to_expire <= seconds_ahead) {
                        print "Key ID:", keyid, "Name:", $10, "Expires in:", int(time_to_expire / 86400), "days";
                        exit 1;
                    }
                    keyid = ""; expiry_date = "";
                } else {
                    print "Error: Key MESSUP negative:", keyid, "Expiry date:", expiry_date, "Name:", $10, "Expires in:", int(time_to_expire / 86400), "days"
                    exit 2
                }
            }
        '
    done
done

# Run apt-get update --audit (if available) to check for signature issues
if apt_supports_audit; then \
  printf "\nRunning apt-get update --audit to check for signature issues...\n"
  apt-get update --audit 2>&1 | tee /tmp/apt-audit.log; \
else \
  exit 0; \
fi


# Check if there are any warnings or errors related to signature validation
# Ignore notices about missing Signed-By or .sources format conversion recommendations
if grep -iE "warning.*signature|audit.*signature|error.*signature|SHA1.*not.*secure" /tmp/apt-audit.log | grep -v "Missing Signed-By" | grep -v "should be upgraded to deb822"; then
    printf "\nERROR: Found signature warnings or errors in apt-get update --audit\n"
    exit 1
fi

printf "apt-get update --audit completed successfully with no SHA1 signature issues\n"
