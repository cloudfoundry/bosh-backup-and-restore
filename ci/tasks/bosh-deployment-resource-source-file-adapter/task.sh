#!/usr/bin/env bash
set -euo pipefail

cat <<EOF> source-file/source-file.yml
---
jumpbox_username: ${BOSH_GW_USER}
jumpbox_url: ${BOSH_GW_HOST}
target: ${BOSH_ENVIRONMENT}
client: ${BOSH_CLIENT}
client_secret: ${BOSH_CLIENT_SECRET}
ca_cert:
EOF
yq -iy ".ca_cert=\"$BOSH_CA_CERT\"" source-file/source-file.yml
yq -iy ".jumpbox_ssh_key=\"$BOSH_GW_PRIVATE_KEY\"" source-file/source-file.yml
