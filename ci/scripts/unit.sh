#!/bin/bash

set -ex

source /bin/docker-lib.sh
start_docker

# pre-load testing ssh node for speed
ls -l backup-and-restore-node-with-ssh/
docker load < backup-and-restore-node-with-ssh/image

eval "$(ssh-agent)"
chmod 400 pcf-backup-and-restore-meta/keys/github
ssh-add pcf-backup-and-restore-meta/keys/github
export GOPATH=$PWD
export PATH=$PATH:$GOPATH/bin
cd src/github.com/pivotal-cf/pcf-backup-and-restore
make test-ci
