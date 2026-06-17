#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="$( cd "$( dirname "${0}" )/.." && pwd )"

function err_echo() {
  # echo to STDERR
  # generally used for human-facing logging to preserve STDOUT as a channel for data that might be piped elsewhere.
   >&2 echo -e "$@"
}

if ! rendered_pipeline=$(ytt -f "${REPO_ROOT}/ci/pipeline.yml"); then
  err_echo "\n\n ytt render failed, please check for template errors above."
  exit 1
fi

if [ -n "${DEBUG:-}" ]; then
  rendered_pipeline="${REPO_ROOT}/ci/pipeline-rendered.yml"
  err_echo "DEBUG: Writing rendered YTT pipeline.yml to\n => '${rendered_pipeline}'"
  echo "${rendered_pipeline}" > "${rendered_pipeline}"
fi

fly -t "${CONCOURSE_TARGET:=bosh}" set-pipeline \
  -p bbr-cli \
  -c <(echo "$rendered_pipeline")
