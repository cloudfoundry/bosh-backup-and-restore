#!/usr/bin/env bash

set -euo pipefail
set -x

[ -d version ]
[ -d promoted-artefacts ]
[ -d repo ]
[ -d pivnet-artefacts ]

: "${TEMPLATE_PATH:?}"
: "${RELEASE_TYPE:?}"

VERSION=$(cat "version/number")

function main {
  copy_pivnet_files 'promoted-artefacts' 'pivnet-artefacts'
  generate_pivnet_metadata "repo/${TEMPLATE_PATH}" 'pivnet-artefacts'
}

function copy_pivnet_files() {
  local promoted_artefacts="${1:?}"
  local pivnet_artefacts="${2:?}"

  ls "$promoted_artefacts"
  cp "$promoted_artefacts"/* "$pivnet_artefacts" 

  rm "$pivnet_artefacts"/*.sha256
}

function generate_pivnet_metadata() {
  local metadata_template="${1:?}"
  local pivnet_artefacts="${2:?}"

  export BBR_LINUX_BINARY="$pivnet_artefacts/bbr-${VERSION}-linux-amd64"
  export BBR_LINUX_ARM64_BINARY="$pivnet_artefacts/bbr-${VERSION}-linux-arm64"
  export BBR_DARWIN_BINARY="$pivnet_artefacts/bbr-${VERSION}-darwin-amd64"
  export BBR_DARWIN_ARM64_BINARY="$pivnet_artefacts/bbr-${VERSION}-darwin-arm64"
  export RELEASE_TAR="$pivnet_artefacts/bbr-${VERSION}.tar"
  export BBR_S3_VALIDATOR_BINARY="$pivnet_artefacts/bbr-s3-config-validator-${VERSION}-linux-amd64"
  export BBR_S3_VALIDATOR_README="$pivnet_artefacts/bbr-s3-config-validator-${VERSION}.README.md"
  export VERSION
  export RELEASE_TYPE

  erb -T- "$metadata_template" > "$pivnet_artefacts/release.yml"
}

main
