#!/bin/bash

set -eu

eval "$(ssh-agent)"
chmod 400 bosh-backup-and-restore-meta/keys/github
chmod 400 bosh-backup-and-restore-meta/genesis-bosh/bosh.pem
ssh-add bosh-backup-and-restore-meta/keys/github

export BOSH_GW_HOST=$BOSH_ENVIRONMENT
export GOPATH=$PWD
export PATH=$PATH:$GOPATH/bin
export BOSH_GW_USER=${BOSH_GW_USER:-vcap}
export BOSH_GW_PRIVATE_KEY=$PWD/bosh-backup-and-restore-meta/genesis-bosh/bosh.pem
export SSH_KEY=$PWD/bosh-backup-and-restore-meta/genesis-bosh/bosh.pem
export BOSH_CA_CERT=$PWD/bosh-backup-and-restore-meta/certs/$BOSH_ENVIRONMENT.crt

cd src/github.com/cloudfoundry-incubator/bosh-backup-and-restore
make sys-test-director-ci
