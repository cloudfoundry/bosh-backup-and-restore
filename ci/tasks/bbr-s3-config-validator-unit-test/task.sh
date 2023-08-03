#!/usr/bin/env bash

set -euo pipefail

cd bosh-backup-and-restore/s3-config-validator/src

go run github.com/onsi/ginkgo/v2/ginkgo \
  -r \
  --randomize-all \
  --randomize-suites \
  --fail-on-pending \
  --keep-going \
  --cover \
  --race \
  --progress \
  --skip="unreadable file" \
  internal

