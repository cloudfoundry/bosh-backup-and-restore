#!/usr/bin/env bash

set -euo pipefail
set -x

cd bosh-backup-and-restore/s3-config-validator/

make test-unit
