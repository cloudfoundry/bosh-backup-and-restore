#!/bin/bash

set -ex

source /bin/docker-lib.sh
start_docker

# pre-load testing ssh node for speed
ls -l backup-and-restore-node-with-ssh/
docker load < backup-and-restore-node-with-ssh/image
ID=$(cat backup-and-restore-node-with-ssh/image-id  |cut -d":" -f2)
REPO=$(cat backup-and-restore-node-with-ssh/repository)
TAG=$(cat backup-and-restore-node-with-ssh/tag)
docker tag $ID $REPO:$TAG

eval "$(ssh-agent)"
./pcf-backup-and-restore-meta/unlock-ci.sh
chmod 400 pcf-backup-and-restore-meta/keys/github
ssh-add pcf-backup-and-restore-meta/keys/github
export GOPATH=$PWD
export PATH=$PATH:$GOPATH/bin
cd src/github.com/pivotal-cf/pcf-backup-and-restore
make test-ci
