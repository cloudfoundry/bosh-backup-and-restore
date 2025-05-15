#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="$( cd "$( dirname "${0}" )/.." && pwd )"

fly -t "${CONCOURSE_TARGET:=bosh-ecosystem}" set-pipeline \
  -p bbr-cli \
  -c <(ytt -f "${REPO_ROOT}/ci/pipeline.yml" --data-values-file "${REPO_ROOT}ci/pipeline-values.yml")
