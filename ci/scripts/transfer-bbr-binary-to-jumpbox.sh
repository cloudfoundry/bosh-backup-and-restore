#!/bin/bash

set -eu
set -o pipefail

chmod 400 bosh-backup-and-restore-meta/genesis-bosh/bosh.pem

bosh-cli \
  --non-interactive \
  --ca-cert="bosh-backup-and-restore-meta/certs/genesis-bosh.backup-and-restore.cf-app.com.crt" \
  --environment="${BOSH_TARGET}" \
  --client="${BOSH_CLIENT}" \
  --client-secret="${BOSH_CLIENT_SECRET}" \
  --deployment="acceptance-jump-box" \
  ssh \
  --gw-private-key="bosh-backup-and-restore-meta/genesis-bosh/bosh.pem" \
  --gw-user="vcap" \
  --gw-host="genesis-bosh.backup-and-restore.cf-app.com" \
  --command="sudo mkdir -p /var/vcap/store/bbr && sudo chmod 775 /var/vcap/store/bbr && sudo chown vcap:vcap /var/vcap/store/bbr" \
  jump-box

# shellcheck disable=SC2011
ls rc/bbr* | xargs -INAME bosh-cli \
  --non-interactive \
  --ca-cert="bosh-backup-and-restore-meta/certs/genesis-bosh.backup-and-restore.cf-app.com.crt" \
  --environment="${BOSH_TARGET}" \
  --client="${BOSH_CLIENT}" \
  --client-secret="${BOSH_CLIENT_SECRET}" \
  --deployment="acceptance-jump-box" \
  scp NAME jump-box:/var/vcap/store/bbr/ \
  --gw-private-key="bosh-backup-and-restore-meta/genesis-bosh/bosh.pem" \
  --gw-user="vcap" \
  --gw-host="genesis-bosh.backup-and-restore.cf-app.com"

# shellcheck disable=SC2011
# list tarballs, remove filename extension, bosh ssh commands to extract tarball
ls rc/bbr* | \
  xargs -INAME basename NAME | rev | cut -d "." -f2- | rev | \
  xargs -INAME bosh-cli \
  --non-interactive \
  --ca-cert="bosh-backup-and-restore-meta/certs/genesis-bosh.backup-and-restore.cf-app.com.crt" \
  --environment="${BOSH_TARGET}" \
  --client="${BOSH_CLIENT}" \
  --client-secret="${BOSH_CLIENT_SECRET}" \
  --deployment="acceptance-jump-box" \
  ssh \
  --gw-private-key="bosh-backup-and-restore-meta/genesis-bosh/bosh.pem" \
  --gw-user="vcap" \
  --gw-host="genesis-bosh.backup-and-restore.cf-app.com" \
  --command="sudo chpst -u vcap:vcap mkdir -p /var/vcap/store/bbr/NAME && \
    sudo chpst -u vcap:vcap tar xvf /var/vcap/store/bbr/NAME.tar -C /var/vcap/store/bbr/NAME/ --strip-components 1 && \
    sudo rm -f /var/vcap/store/bbr/NAME.tar" \
  jump-box