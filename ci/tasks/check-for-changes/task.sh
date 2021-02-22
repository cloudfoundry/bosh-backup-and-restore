#!/usr/bin/env bash

set -euo pipefail

pushd repo
  latest_tag=$(git describe --abbrev=0 --tags)
  has_changes=$(git log "${latest_tag}"..HEAD)

  echo "Running: git log ${latest_tag}..HEAD"
  echo "Changes are:"
  echo "$has_changes"

  if [[ -z "$has_changes" ]]; then
    echo "There are no changes to publish in a new release!"
    exit 1
  fi
popd
