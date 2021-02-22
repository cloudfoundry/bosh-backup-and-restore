#!/usr/bin/env bash

set -euo pipefail

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

