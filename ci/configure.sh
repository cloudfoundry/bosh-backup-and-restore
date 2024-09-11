#!/bin/bash

set -euo pipefail

fly -t bosh-ecosystem set-pipeline \
  -p bbr-cli \
  -c <(ytt -f ci/pipeline.yml --data-values-file ci/pipeline-values.yml)