#!/usr/bin/env bash

set -euo pipefail

fly --target cryo-bbr set-pipeline \
  --pipeline test-github-prs \
  --config pipeline.yml


