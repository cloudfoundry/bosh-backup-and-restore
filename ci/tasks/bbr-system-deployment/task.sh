#!/bin/bash

set -eu

eval "$(ssh-agent)"
echo -e "${GITHUB_SDK_PRIVATE_KEY}" > "${PWD}/github-sdk.key"
chmod 400 "${PWD}/github-sdk.key"
ssh-add "${PWD}/github-sdk.key"

echo -e "${BOSH_GW_PRIVATE_KEY}" > "${PWD}/ssh.key"
chmod 0600 "${PWD}/ssh.key"
export BOSH_GW_PRIVATE_KEY="${PWD}/ssh.key"
export BOSH_ALL_PROXY="ssh+socks5://${BOSH_GW_USER}@${BOSH_GW_HOST}?private-key=${BOSH_GW_PRIVATE_KEY}"

GOBIN=/usr/local/bin go install github.com/onsi/ginkgo/ginkgo@latest
ginkgo version

ln -s "$( which bosh )" /usr/local/bin/bosh-cli

cd bosh-backup-and-restore
make sys-test-deployment-ci
