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

mkdir -p $GOPATH/src/github.com/cloudfoundry
cp src/github.com/pivotal-cf/pcf-backup-and-restore/vendor/github.com/cloudfoundry/bosh-cli $GOPATH/src/github.com/cloudfoundry
go install github.com/cloudfoundry/bosh-cli

cd src/github.com/pivotal-cf/pcf-backup-and-restore
make sys-test-ci
