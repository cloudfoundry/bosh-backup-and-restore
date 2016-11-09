#!/bin/bash

set -cxe

eval "$(ssh-agent)"
chmod 400 pcf-backup-and-restore-meta/keys/github
ssh-add pcf-backup-and-restore-meta/keys/github
export GOPATH=$PWD
export PATH=$PATH:$GOPATH/bin
export VERSION=$(cat version/number)
cd src/github.com/pivotal-cf/pcf-backup-and-restore
make release
tar -cvf pbr-$VERSION.tar releases/*
cd -
mv src/github.com/pivotal-cf/pcf-backup-and-restore/pbr-$VERSION.tar pbr-build/