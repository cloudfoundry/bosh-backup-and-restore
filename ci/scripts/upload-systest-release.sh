#!/bin/bash

set -eu

if [[ "$USE_BOSH_ALL_PROXY" = true ]]; then
  echo -e "${BOSH_GW_PRIVATE_KEY}" > "${PWD}/ssh.key"
  chmod 0600 "${PWD}/ssh.key"
  export BOSH_GW_PRIVATE_KEY="${PWD}/ssh.key"
  export BOSH_ALL_PROXY="ssh+socks5://${BOSH_GW_USER}@${BOSH_GW_HOST}?private-key=${BOSH_GW_PRIVATE_KEY}"
fi

if [[ ! -z "$BOSH_CA_CERT_PATH" ]]; then
  export BOSH_CA_CERT="$PWD/$BOSH_CA_CERT_PATH"
fi
cd "bbr-systest-releases/${RELEASE_NAME}"

bosh-cli -n create-release --force
bosh-cli upload-release --rebase
