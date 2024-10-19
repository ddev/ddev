#!/bin/bash

# Read secrets into environment from 1password

for item in DDEV_ACQUIA_API_KEY DDEV_ACQUIA_API_SECRET DDEV_PANTHEON_API_TOKEN DDEV_PLATFORM_API_TOKEN DDEV_UPSUN_API_TOKEN; do
  printf "export ${item}=$(op item get ${item} --field=credential --reveal)\n"
done >/tmp/1penv.sh

for item in DDEV_ACQUIA_SSH_KEY DDEV_LAGOON_SSH_KEY DDEV_PANTHEON_SSH_KEY; do
  printf "export ${item}=$(op item get --field='private key' --reveal ${item})\n"
done >> /tmp/1penv.sh

echo "You can now 'source /tmp/1penv.sh' to get the new environment variables"
