#!/usr/bin/env bash

set -euo pipefail

[ -d environment ]
[ -d bosh-env ]

export ENVIRONMENT_LOCK_METADATA=environment/metadata

readonly bosh_all_proxy_pattern='ssh\+socks5:\/\/(.*)@(([0-9]+\.){3}([0-9]+)):22\?private-key=(.*)'

BOSH_ALL_PROXY="$(jq -r '.bosh.bosh_all_proxy' ${ENVIRONMENT_LOCK_METADATA})"
BOSH_CA_CERT="$(jq -r '.bosh.bosh_ca_cert' ${ENVIRONMENT_LOCK_METADATA})"
CREDHUB_CA_CERT="$(jq -r '.bosh.credhub_ca_cert' ${ENVIRONMENT_LOCK_METADATA})"

# JUMPBOX_PRIVATE_KEY is present for cf-deployment pool envs
: "${JUMPBOX_PRIVATE_KEY:="$(echo "${BOSH_ALL_PROXY}" | sed -n -E "s/${bosh_all_proxy_pattern}/\5/p")"}"

if [ -f "$BOSH_CA_CERT" ]
then
  BOSH_CA_CERT="$(cat "$BOSH_CA_CERT")"
  # export BOSH_CA_CERT
fi


cat > bosh-env/alias-env.sh << EOF
export INSTANCE_JUMPBOX_PRIVATE="$(jq -r '.bosh.jumpbox_private_key' ${ENVIRONMENT_LOCK_METADATA})"
export INSTANCE_JUMPBOX_USER="$(echo "${BOSH_ALL_PROXY}" | sed -n -E "s/${bosh_all_proxy_pattern}/\1/p")"
export INSTANCE_JUMPBOX_EXTERNAL_IP="$(echo "${BOSH_ALL_PROXY}" | sed -n -E "s/${bosh_all_proxy_pattern}/\2/p")"

JUMPBOX_PRIVATE_KEY="\$(mktemp)"
chmod 0600 "\$JUMPBOX_PRIVATE_KEY"
echo "\$INSTANCE_JUMPBOX_PRIVATE" > "\$JUMPBOX_PRIVATE_KEY"

export BOSH_CLIENT="$(jq -r '.bosh.bosh_client' ${ENVIRONMENT_LOCK_METADATA})"
export BOSH_CLIENT_SECRET="$(jq -r '.bosh.bosh_client_secret' ${ENVIRONMENT_LOCK_METADATA})"
export BOSH_ENVIRONMENT="$(jq -r '.bosh.bosh_environment' ${ENVIRONMENT_LOCK_METADATA})"
export BOSH_CA_CERT="$(jq -r '.bosh.bosh_ca_cert' ${ENVIRONMENT_LOCK_METADATA})"
export BOSH_ALL_PROXY="ssh+socks5://\${INSTANCE_JUMPBOX_USER}@\${INSTANCE_JUMPBOX_EXTERNAL_IP}:22?private-key=\${JUMPBOX_PRIVATE_KEY}"
export BOSH_ENV_NAME="$(jq -r '.name' ${ENVIRONMENT_LOCK_METADATA})"

export CREDHUB_PROXY="\$BOSH_ALL_PROXY"
export CREDHUB_SERVER="$(jq -r '.bosh.credhub_server' ${ENVIRONMENT_LOCK_METADATA})"
export CREDHUB_CLIENT="$(jq -r '.bosh.credhub_client' ${ENVIRONMENT_LOCK_METADATA})"
export CREDHUB_SECRET="$(jq -r '.bosh.credhub_secret' ${ENVIRONMENT_LOCK_METADATA})"
export CREDHUB_CA_CERT="$(jq -r '.bosh.credhub_ca_cert' ${ENVIRONMENT_LOCK_METADATA})"
EOF

cat > bosh-env/metadata.yml << EOF
INSTANCE_JUMPBOX_PRIVATE: |-
$(jq -r '.bosh.jumpbox_private_key' ${ENVIRONMENT_LOCK_METADATA} | sed -E 's/(-+(BEGIN|END) RSA PRIVATE KEY-+) *| +/\1\n/g' |  sed 's/^/  /')
INSTANCE_JUMPBOX_USER: "$(echo "${BOSH_ALL_PROXY}" | sed -n -E "s/${bosh_all_proxy_pattern}/\1/p")"
INSTANCE_JUMPBOX_EXTERNAL_IP: "$(echo "${BOSH_ALL_PROXY}" | sed -n -E "s/${bosh_all_proxy_pattern}/\2/p")"
BOSH_CLIENT: "$(jq -r '.bosh.bosh_client' ${ENVIRONMENT_LOCK_METADATA})"
BOSH_CLIENT_SECRET: "$(jq -r '.bosh.bosh_client_secret' ${ENVIRONMENT_LOCK_METADATA})"
BOSH_ENVIRONMENT: "$(jq -r '.bosh.bosh_environment' ${ENVIRONMENT_LOCK_METADATA})"
BOSH_CA_CERT: |-
$(echo $BOSH_CA_CERT | sed -E 's/(-+(BEGIN|END) CERTIFICATE-+) *| +/\1\n/g' | sed 's/^/  /')
CREDHUB_PROXY: "$BOSH_ALL_PROXY"
CREDHUB_SERVER: "$(jq -r '.bosh.credhub_server' ${ENVIRONMENT_LOCK_METADATA})"
CREDHUB_CLIENT: "$(jq -r '.bosh.credhub_client' ${ENVIRONMENT_LOCK_METADATA})"
CREDHUB_SECRET: "$(jq -r '.bosh.credhub_secret' ${ENVIRONMENT_LOCK_METADATA})"
CREDHUB_CA_CERT: |-
$(echo $BOSH_CA_CERT | sed -E 's/(-+(BEGIN|END) CERTIFICATE-+) *| +/\1\n/g' | sed 's/^/  /')
EOF