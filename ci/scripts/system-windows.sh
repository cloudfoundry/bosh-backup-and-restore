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
fi

export GOPATH=$PWD
export PATH=$PATH:$GOPATH/bin
cd src/github.com/cloudfoundry-incubator/bosh-backup-and-restore
make sys-test-windows-ci
