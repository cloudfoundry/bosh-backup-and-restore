#!/usr/bin/env bash

set -euo pipefail
set -x

cd bosh-backup-and-restore-ci/s3-config-validator/

make test-unit
