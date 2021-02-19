#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$PWD"
export VERSION=$(cat "version-folder/${VERSION_PATH}")

pushd repo
  latest_tag=$(git describe --abbrev=0 --tags)
  export COMMITS=$(git log ${latest_tag}..HEAD --oneline | grep -v "Merge" | cut -d ' ' -f2-| uniq -u)
popd

erb -r date -T- template-folder/${TEMPLATE_PATH} > "${ROOT_DIR}/generated-release-notes.txt"

pushd docs-repo
  sed -i "/Releases/ r ${ROOT_DIR}/generated-release-notes.txt" bbr-rn.html.md.erb
  echo -e "\n > Generated Release Notes:"
  cat bbr-rn.html.md.erb

  git add bbr-rn.html.md.erb
  git config --global user.name "Cryogenics CI"
  git config --global user.email "cf-lazarus@pivotal.io"
  git commit -m "Release Notes for version: ${VERSION}"
popd

cp -r docs-repo/. updated-docs-repo/
