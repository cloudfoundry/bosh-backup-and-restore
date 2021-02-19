#!/usr/bin/env bash
set -euo pipefail

if [ -z "${BBL_STATE:=}" ]; then 2>&1 echo "BBL_STATE must be provided"; fi

function get_ip_port() {
  grep -o '[0-9]\{1,\}\.[0-9]\{1,\}\.[0-9]\{1,\}\.[0-9]\{1,\}:[0-9]\{1,\}' <<< "$1"
}

eval "$( bbl --state-dir "bosh-backup-and-restore-meta/$BBL_STATE" print-env )"

yq write <( cat <<EOF
---
jumpbox_username: jumpbox
jumpbox_url: $( get_ip_port "$BOSH_ALL_PROXY" )
target: ${BOSH_ENVIRONMENT}
client: ${BOSH_CLIENT}
client_secret: ${BOSH_CLIENT_SECRET}
EOF
) -- "ca_cert" "$BOSH_CA_CERT" > source-file/source-file.yml

yq write --inplace -- source-file/source-file.yml jumpbox_ssh_key "$( cat "$JUMPBOX_PRIVATE_KEY" )"

