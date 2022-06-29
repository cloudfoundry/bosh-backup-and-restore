#!/usr/bin/env bash
set -euo pipefail

if [[ -z "${BOSH_GW_USER}" ]] && [[ -z "${BOSH_GW_HOST}" ]] && [[ -n "${BOSH_ALL_PROXY}" ]]; then
    export BOSH_GW_USER="$(echo "${BOSH_ALL_PROXY}" | sed -nr 's/^.*:\/\/([^@]+)@([^?]+)?.*$/\1/p')"
    export BOSH_GW_HOST="$(echo "${BOSH_ALL_PROXY}" | sed -nr 's/^.*:\/\/([^@]+)@([^?]+)?.*$/\2/p')"
fi

if [[ "${BOSH_GW_HOST}" != *":"* ]]; then
    export BOSH_GW_HOST="${BOSH_GW_HOST}:22"
fi

cat <<EOF> source-file/source-file.yml
---
jumpbox_username: ${BOSH_GW_USER}
jumpbox_url: ${BOSH_GW_HOST}
target: ${BOSH_ENVIRONMENT}
client: ${BOSH_CLIENT}
client_secret: ${BOSH_CLIENT_SECRET}
EOF

yq write --inplace -- source-file/source-file.yml ca_cert "$(sed -E 's/(-+(BEGIN|END) CERTIFICATE-+) *| +/\1\n/g' <<< "$BOSH_CA_CERT")"
yq write --inplace -- source-file/source-file.yml jumpbox_ssh_key "$(sed -E 's/(-+(BEGIN|END) RSA PRIVATE KEY-+) *| +/\1\n/g' <<< "$BOSH_GW_PRIVATE_KEY")"
