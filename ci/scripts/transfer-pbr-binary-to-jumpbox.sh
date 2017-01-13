#!/bin/bash

set -eu

ls release | xargs -I{} bosh -n -t ${BOSH_TARGET} -u ${BOSH_CLIENT} -p ${BOSH_CLIENT_SECRET} \
  -d pcf-backup-and-restore-meta/deployments/acceptance-jump-box.yml \
  scp jump-box 0 release/{} /var/vcap/store/ \
  --upload --gateway_identity_file pcf-backup-and-restore-meta/genesis-bosh/bosh.pem \
  --gateway-user vcap --gateway-host lite-bosh.backup-and-restore.cf-app.com
