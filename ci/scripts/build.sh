#!/bin/bash

set -ex

eval "$(ssh-agent)"
chmod 400 pcf-backup-and-restore-meta/keys/github
ssh-add pcf-backup-and-restore-meta/keys/github
export GOPATH=$PWD
export PATH=$PATH:$GOPATH/bin
export VERSION=$(cat version/number)

pushd src/github.com/pivotal-cf/pcf-backup-and-restore
  make release
  tar -cvf pbr-"$VERSION".tar releases/*
popd

mv src/github.com/pivotal-cf/pcf-backup-and-restore/pbr-"$VERSION".tar pbr-build/
