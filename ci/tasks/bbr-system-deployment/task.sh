#!/bin/bash
set -euo pipefail

printf '%s' "${BOSH_GW_PRIVATE_KEY}" > "${PWD}/ssh.key"
chmod 0600 "${PWD}/ssh.key"
export BOSH_GW_PRIVATE_KEY="${PWD}/ssh.key"
export BOSH_ALL_PROXY="ssh+socks5://${BOSH_GW_USER}@${BOSH_GW_HOST}?private-key=${BOSH_GW_PRIVATE_KEY}"

FIXTURES_DIR="$(realpath "${FIXTURES_DIR}")"
export FIXTURES_DIR

cd bosh-backup-and-restore
make sys-test-deployment-ci
