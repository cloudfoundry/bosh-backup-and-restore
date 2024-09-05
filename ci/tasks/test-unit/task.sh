#!/usr/bin/env bash

set -euo pipefail
set -x

cd bosh-backup-and-restore/

make test-unit
