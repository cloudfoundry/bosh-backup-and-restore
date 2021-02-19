#!/usr/bin/env bash

set -eu

pool_metadata="env-pool/metadata"

OPS_MAN_URL=$(< "$pool_metadata" jq -r .ops_manager.url)
OPS_MAN_USER=$(< "$pool_metadata" jq -r .ops_manager.username)
OPS_MAN_PASS=$(< "$pool_metadata" jq -r .ops_manager.password)

OM_CMD="om-6 --target ${OPS_MAN_URL} --username ${OPS_MAN_USER} --password ${OPS_MAN_PASS} -k"

# Set BOSH env vars
eval "$($OM_CMD bosh-env)"

# Set OPS_MAN_PRIVATE_KEY
export OPS_MAN_PRIVATE_KEY=$(mktemp)
cat env-pool/metadata | jq -r .ops_manager_private_key > $OPS_MAN_PRIVATE_KEY
chmod 0600 $OPS_MAN_PRIVATE_KEY

# Set BOSH_CA_CERT
export BOSH_CA_PATH=$(mktemp)
printf %s "$BOSH_CA_CERT" > $BOSH_CA_PATH
unset BOSH_CA_CERT
export BOSH_CA_CERT=$BOSH_CA_PATH
chmod 0600 $BOSH_CA_CERT

export OPS_MAN_IP="$(cat env-pool/metadata | jq -r .ops_manager_public_ip)"
export BOSH_ALL_PROXY=ssh+socks5://ubuntu@${OPS_MAN_IP}:22?private-key=${OPS_MAN_PRIVATE_KEY}

# untar validator
tar -xf bbr-s3-config-validator-test-artifacts/bbr-s3-config-validator.*.tgz

CF_DEPLOYMENT=$(bosh deps --json | jq -r .Tables[0].Rows[0].name)

bosh -d ${CF_DEPLOYMENT} scp bbr-s3-config-validator backup_restore:/tmp

bosh -d ${CF_DEPLOYMENT} ssh backup_restore -c '
  mv /tmp/bbr-s3-config-validator .
  chmod +x bbr-s3-config-validator
  ./bbr-s3-config-validator --unversioned --validate-put-object' \
    | sed 's/"\(aws_.*\)"\: "\(.*\)"/"\1": "<redacted>"/g
'

