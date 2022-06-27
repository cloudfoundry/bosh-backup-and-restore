#!/usr/bin/env bash
set -euo pipefail

if [[ -z "${BOSH_GW_USER}" ]] && [[ -z "${BOSH_GW_HOST}" ]] && [[ -n "${BOSH_ALL_PROXY}" ]]; then
    export BOSH_GW_USER="$(echo "${BOSH_ALL_PROXY}" | sed -nr 's/^.*:\/\/([^@]+)@([^?]+)?.*$/\1/p')"
    export BOSH_GW_HOST="$(echo "${BOSH_ALL_PROXY}" | sed -nr 's/^.*:\/\/([^@]+)@([^?]+)?.*$/\2/p')"
fi
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

