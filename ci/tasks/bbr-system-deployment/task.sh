#!/bin/bash

set -eu

eval "$(ssh-agent)"
echo -e "${GITHUB_SDK_PRIVATE_KEY}" > "${PWD}/github-sdk.key"
chmod 400 "${PWD}/github-sdk.key"
ssh-add "${PWD}/github-sdk.key"

echo -e "${BOSH_GW_PRIVATE_KEY}" > "${PWD}/ssh.key"
chmod 0600 "${PWD}/ssh.key"
export BOSH_GW_PRIVATE_KEY="${PWD}/ssh.key"

ssh-add "$BOSH_GW_PRIVATE_KEY"

export DIRECTOR_HOST="$(echo "$BOSH_ENVIRONMENT" | sed -E 's/(https:\/\/)?([^:]*)(:.*)?/\2/g')"
export DIRECTOR_PORT="$(echo "$BOSH_ENVIRONMENT" | sed -E 's/(https:\/\/)?([^:]*)(:.*)?/\3/g')"

sshuttle -r "${BOSH_GW_USER}@${BOSH_GW_HOST}" "$DIRECTOR_HOST/32$DIRECTOR_PORT" \
  --daemon \
  -e 'ssh -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no -o ServerAliveInterval=600'
echo "Establishing tunnel to Director via Jumpbox..."
sleep 5

if ! stat sshuttle.pid > /dev/null 2>&1; then
  echo "Failed to start sshuttle daemon"
  exit 1
fi

export BOSH_ALL_PROXY="ssh+socks5://${BOSH_GW_USER}@${BOSH_GW_HOST}?private-key=${BOSH_GW_PRIVATE_KEY}"

cd bosh-backup-and-restore
make sys-test-deployment-ci
