#!/usr/bin/env bash

set -euo pipefail
set -x

if [ -d version ]; then
  export VERSION=$(cat version/version)
else
  export VERSION=$(date +%s)
fi

ROOT_DIR=$PWD

cd bosh-backup-and-restore/s3-config-validator

make artifact

cp -r \
  build/* \
  "${ROOT_DIR}/bbr-s3-config-validator-build/"
