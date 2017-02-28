#!/bin/bash

set -ex

eval "$(ssh-agent)"
./pcf-backup-and-restore-meta/unlock-ci.sh
chmod 400 pcf-backup-and-restore-meta/keys/github
chmod 400 pcf-backup-and-restore-meta/genesis-bosh/bosh.pem
ssh-add pcf-backup-and-restore-meta/keys/github
export GOPATH=$PWD
export PATH=$PATH:$GOPATH/bin
export BOSH_CERT_PATH=`pwd`/pcf-backup-and-restore-meta/certs/lite-bosh.backup-and-restore.cf-app.com.crt
export BOSH_GATEWAY_KEY=`pwd`/pcf-backup-and-restore-meta/genesis-bosh/bosh.pem

cd src/github.com/pivotal-cf/bosh-backup-and-restore
make sys-test-ci
