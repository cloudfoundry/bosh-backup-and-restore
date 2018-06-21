#!/bin/bash

set -eu

eval "$(ssh-agent)"
chmod 400 bosh-backup-and-restore-meta/keys/github
ssh-add bosh-backup-and-restore-meta/keys/github

if [[ "$USE_BOSH_ALL_PROXY" = true ]]; then
  echo -e "${BOSH_GW_PRIVATE_KEY}" > "${PWD}/ssh.key"
  chmod 0600 "${PWD}/ssh.key"
  export BOSH_GW_PRIVATE_KEY="${PWD}/ssh.key"
  export BOSH_ALL_PROXY="ssh+socks5://${BOSH_GW_USER}@${BOSH_GW_HOST}?private-key=${BOSH_GW_PRIVATE_KEY}"
else
  chmod 400 "${BOSH_GATEWAY_KEY:-bosh-backup-and-restore-meta/genesis-bosh/bosh.pem}"
  export BOSH_GATEWAY_HOST=$BOSH_HOST
  export BOSH_URL=https://$BOSH_HOST
  export BOSH_GATEWAY_USER=${BOSH_GATEWAY_USER:-vcap}
  export BOSH_GATEWAY_KEY=$PWD/${BOSH_GATEWAY_KEY:-bosh-backup-and-restore-meta/genesis-bosh/bosh.pem}
fi

if [[ -z "$BOSH_CA_CERT" ]]; then
  export BOSH_CERT_PATH=$PWD/bosh-backup-and-restore-meta/certs/$BOSH_HOST.crt
fi

export GOPATH=$PWD
export PATH=$PATH:$GOPATH/bin
cd src/github.com/cloudfoundry-incubator/bosh-backup-and-restore
make sys-test-ci
