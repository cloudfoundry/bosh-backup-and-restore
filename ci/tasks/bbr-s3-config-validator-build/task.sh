#!/usr/bin/env bash

set -euo pipefail
set -x

[ -d version ]
[ -d repo ]

VERSION="$(cat version/number)"

ROOT_DIR=$PWD

cd repo/s3-config-validator

make artifact

cp -r \
  build/artifact.tgz \
  "${ROOT_DIR}/bbr-s3-config-validator-test-artifacts/bbr-s3-config-validator.$VERSION.tgz"
