#!/usr/bin/env bash

set -euo pipefail

cd bosh-backup-and-restore/s3-config-validator/src
go mod vendor
go run github.com/onsi/ginkgo/v2/ginkgo \
  -r \
  --randomize-all \
  --keep-going \
  --fail-on-pending \
  --cover \
  --race \
  --show-node-events \
  test \
  | sed 's/"\(aws_.*\)"\: "\(.*\)"/"\1": "<redacted>"/g'

