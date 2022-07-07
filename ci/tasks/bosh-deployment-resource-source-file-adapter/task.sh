#!/usr/bin/env bash
set -euo pipefail

yq write <( cat <<EOF
---
jumpbox_username: ${BOSH_GW_USER}
jumpbox_url: ${BOSH_GW_HOST}
target: ${BOSH_ENVIRONMENT}
client: ${BOSH_CLIENT}
client_secret: ${BOSH_CLIENT_SECRET}
EOF
) -- "ca_cert" "$BOSH_CA_CERT" > source-file/source-file.yml

yq write --inplace -- source-file/source-file.yml jumpbox_ssh_key "$BOSH_GW_PRIVATE_KEY"

