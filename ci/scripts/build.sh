#!/bin/bash

set -ex

eval "$(ssh-agent)"
./pcf-backup-and-restore-meta/unlock-ci.sh
chmod 400 pcf-backup-and-restore-meta/keys/github
ssh-add pcf-backup-and-restore-meta/keys/github
export GOPATH=$PWD
export PATH=$PATH:$GOPATH/bin
export VERSION=$(cat version/number)

pushd src/github.com/pivotal-cf/bosh-backup-and-restore
  make release
  tar -cvf bbr-"$VERSION".tar releases/*
popd

mv src/github.com/pivotal-cf/bosh-backup-and-restore/bbr-"$VERSION".tar bbr-build/

echo "Auto-delivered in
https://s3-eu-west-1.amazonaws.com/bosh-backup-and-restore-builds/bbr-$VERSION.tar

[Backup and Restore Bot]" > bbr-build/message
