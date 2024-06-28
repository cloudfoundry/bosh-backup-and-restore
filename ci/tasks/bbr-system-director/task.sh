#!/bin/bash

set -eu
set -o pipefail

# Add GitHub SSH key to avoid rate-limit
eval "$(ssh-agent)"

# # Write Jumpbox SSH key to file
echo -e "${BOSH_GW_PRIVATE_KEY}" > "${PWD}/ssh.key"
chmod 0600 "${PWD}/ssh.key"
export BOSH_GW_PRIVATE_KEY="${PWD}/ssh.key"

DIRECTOR_SSH_KEY_PATH="$(mktemp)"
echo -e "${DIRECTOR_SSH_KEY}" > "$DIRECTOR_SSH_KEY_PATH"
chmod 0600 "$DIRECTOR_SSH_KEY_PATH"
export DIRECTOR_SSH_KEY_PATH

# Create tunnel to Director via Jumpbox
ssh-add "$BOSH_GW_PRIVATE_KEY"

if [[ "${USE_SHUTTLE}" == "true" ]]; then
  sshuttle -r "${BOSH_GW_USER}@${BOSH_GW_HOST}" "$DIRECTOR_HOST/32" \
    --daemon \
    -e 'ssh -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no -o ServerAliveInterval=600'
  echo "Establishing tunnel to Director via Jumpbox..."
  sleep 5

  if ! stat sshuttle.pid > /dev/null 2>&1; then
    echo "Failed to start sshuttle daemon"
    exit 1
  fi
  export CREDHUB_PROXY="ssh+socks5://${BOSH_GW_USER}@${BOSH_GW_HOST}?private-key=${BOSH_GW_PRIVATE_KEY}"
else
  cat << EOF > ~/.ssh/config
Host jumphost
  HostName $(echo "${BOSH_GW_HOST}" | cut -f1 -d: )
  User ${BOSH_GW_USER}
  IdentityFile ${BOSH_GW_PRIVATE_KEY}

  StrictHostKeyChecking no
### Second jumphost. Only reachable via jumphost1.example.org
Host ${DIRECTOR_HOST}
  HostName ${DIRECTOR_HOST}
  ProxyJump jumphost

EOF

fi

export BOSH_ALL_PROXY="ssh+socks5://${BOSH_GW_USER}@${BOSH_GW_HOST}?private-key=${BOSH_GW_PRIVATE_KEY}"

cd bosh-backup-and-restore
make sys-test-director-ci
