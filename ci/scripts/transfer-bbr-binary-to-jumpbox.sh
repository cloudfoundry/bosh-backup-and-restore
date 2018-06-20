#!/bin/bash

set -eu
set -o pipefail

chmod 400 bosh-backup-and-restore-meta/genesis-bosh/bosh.pem

bosh \
  -n \
  -t "${BOSH_TARGET}" \
  -u "${BOSH_CLIENT}" \
  -p "${BOSH_CLIENT_SECRET}" \
  -d bosh-backup-and-restore-meta/deployments/acceptance-jump-box.yml \
  ssh \
  --gateway_identity_file bosh-backup-and-restore-meta/genesis-bosh/bosh.pem \
  --gateway_user vcap \
  --gateway_host genesis-bosh.backup-and-restore.cf-app.com \
  jump-box 0 \
  "sudo mkdir -p /var/vcap/store/bbr && \
   sudo chmod 775 /var/vcap/store/bbr && \
   sudo chown vcap:vcap /var/vcap/store/bbr
  "

# shellcheck disable=SC2011
ls rc/bbr* | xargs -INAME bosh -n -t "${BOSH_TARGET}" -u "${BOSH_CLIENT}" -p "${BOSH_CLIENT_SECRET}" \
  -d bosh-backup-and-restore-meta/deployments/acceptance-jump-box.yml \
  scp jump-box 0 NAME /var/vcap/store/bbr/ \
  --upload \
  --gateway_identity_file bosh-backup-and-restore-meta/genesis-bosh/bosh.pem \
  --gateway_user vcap \
  --gateway_host genesis-bosh.backup-and-restore.cf-app.com

# shellcheck disable=SC2011
# list tarballs, remove filename extention, bosh ssh commands to extract tarball
ls rc/bbr* | \
  xargs -INAME basename NAME | rev | cut -d "." -f2- | rev | \
  xargs -INAME bosh \
  -n \
  -t "${BOSH_TARGET}" \
  -u "${BOSH_CLIENT}" \
  -p "${BOSH_CLIENT_SECRET}" \
  -d bosh-backup-and-restore-meta/deployments/acceptance-jump-box.yml \
  ssh \
  --gateway_identity_file bosh-backup-and-restore-meta/genesis-bosh/bosh.pem \
  --gateway_user vcap \
  --gateway_host genesis-bosh.backup-and-restore.cf-app.com \
  jump-box 0 \
  "sudo chpst -u vcap:vcap mkdir -p /var/vcap/store/bbr/NAME && \
   sudo chpst -u vcap:vcap tar xvf /var/vcap/store/bbr/NAME.tar -C /var/vcap/store/bbr/NAME/ --strip-components 1 && \
   sudo rm -f /var/vcap/store/bbr/NAME.tar
  "
