#!/usr/bin/env bash

set -euo pipefail

target=${1:?"usage: $0 TARGET"}

fly --target $target set-pipeline \
  --pipeline bbr \
  --config pipeline.yml


