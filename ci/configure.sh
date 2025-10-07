#!/usr/bin/env bash
set -euo pipefail

function errcho() {
  # echo to STDERR
  # generally used for human-facing logging to preserve STDOUT as a channel for data that might be piped elsewhere.
   >&2 echo -e "$@"
}


REPO_ROOT="$( cd "$( dirname "${0}" )/.." && pwd )"

function main() {

  local rendered_pipeline
  if ! rendered_pipeline=$(ytt -f "${REPO_ROOT}/ci/pipeline.yml"); then
    errcho "\n\nytt render failed, please check for template errors above."
    exit 1
  fi

  #getting a yaml error w/ a line number? uncomment the below to find it more quickly.
  #echo "$rendered_pipeline" > ./render.yaml

  fly -t "${CONCOURSE_TARGET:=bosh-ecosystem}" set-pipeline \
    -p bbr-cli \
    -c <(echo "$rendered_pipeline")

}

main "$@"
