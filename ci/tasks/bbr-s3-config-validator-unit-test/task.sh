#!/usr/bin/env bash

set -euo pipefail

cd bosh-backup-and-restore/s3-config-validator/src

ginkgo \
  -r \
  --randomizeAllSpecs \
  --randomizeSuites \
  --failOnPending \
  --keepGoing \
  --cover \
  --race \
  --progress \
  --skip="unreadable file" \
  internal

