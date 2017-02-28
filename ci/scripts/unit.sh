#!/bin/bash

set -ex

eval "$(ssh-agent)"
./pcf-backup-and-restore-meta/unlock-ci.sh
chmod 400 pcf-backup-and-restore-meta/keys/github
ssh-add pcf-backup-and-restore-meta/keys/github
export GOPATH=$PWD
export PATH=$PATH:$GOPATH/bin
cd src/github.com/pivotal-cf/bosh-backup-and-restore
make test-ci
make clean-docker || true