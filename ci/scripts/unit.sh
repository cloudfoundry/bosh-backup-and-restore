#!/bin/bash

set -eu

eval "$(ssh-agent)"
github_ssh_key=$(mktemp)
# shellcheck disable=2153
echo "$GITHUB_SSH_KEY" > "$github_ssh_key"
chmod 400 "$github_ssh_key"
ssh-add "$github_ssh_key"

export GOPATH=$PWD
export PATH=$PATH:$GOPATH/bin

cd src/github.com/cloudfoundry-incubator/bosh-backup-and-restore
make test-ci
make clean-docker || true
