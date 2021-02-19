#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR=$PWD

cd bbr-s3-config-validator/s3-config-validator

make artifact

cp -r \
  build/artifact.tgz \
  "${ROOT_DIR}/bbr-s3-config-validator-test-artifacts/bbr-s3-config-validator.$(cat "${ROOT_DIR}"/s3-config-validator-version/number).tgz"
