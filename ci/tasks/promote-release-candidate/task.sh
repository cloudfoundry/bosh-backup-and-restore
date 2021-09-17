#!/bin/bash

set -eu
set -x

VERSION=$(cat bbr-final-release-version/number)

function main {
  local release_folder='bbr-final-release-artefacts'

  copy_bbr_binaries_to "$release_folder"
  copy_s3_config_validator_artefacts_to "$release_folder"
  create_bbr_tarball_in "$release_folder"
}

function create_bbr_tarball_in() {
  local release_folder="${1:?}"

  local bbr_tarball_folder
  bbr_tarball_folder="$(mktemp -d)"

  ls bbr-rc-artefacts/releases

  cp 'bbr-rc-artefacts/releases/bbr' "$bbr_tarball_folder"
  cp 'bbr-rc-artefacts/releases/bbr-mac' "$bbr_tarball_folder"
  cp 'bbr-rc-artefacts/releases/checksum.sha256' "$bbr_tarball_folder"

  cp 's3-config-validator-rc-artefacts/bbr-s3-config-validator' "$bbr_tarball_folder"
  cp 's3-config-validator-rc-artefacts/README.md' "$bbr_tarball_folder/bbr-s3-config-validator.README.md"

  echo "$(cat s3-config-validator-rc-artefacts/bbr-s3-config-validator.sha256)  bbr-s3-config-validator" >> "$bbr_tarball_folder/checksum.sha256"

  tar -cf "$release_folder/bbr-${VERSION}.tar" -C "$bbr_tarball_folder" .
}

function copy_bbr_binaries_to {
  local release_folder="${1:?}"
  local binary_name

  binary_name="bbr-${VERSION}-linux-amd64"
  cp 'bbr-rc-artefacts/releases/bbr' "$release_folder/${binary_name}"
  grep 'bbr$' 'bbr-rc-artefacts/releases/checksum.sha256'  | cut -d' ' -f1 > "$release_folder/${binary_name}.sha256"

  binary_name="bbr-$VERSION-darwin-amd64"
  cp 'bbr-rc-artefacts/releases/bbr-mac' "$release_folder/${binary_name}"
  grep 'bbr-mac$' 'bbr-rc-artefacts/releases/checksum.sha256'  | cut -d' ' -f1 > "$release_folder/${binary_name}.sha256"
}

function copy_s3_config_validator_artefacts_to {
  local release_folder="${1:?}"

  cp 's3-config-validator-rc-artefacts/README.md' "$release_folder/bbr-s3-config-validator-$VERSION.README.md"
  cp 's3-config-validator-rc-artefacts/bbr-s3-config-validator' "$release_folder/bbr-s3-config-validator-$VERSION-linux-amd64"
  cp 's3-config-validator-rc-artefacts/bbr-s3-config-validator.sha256' "$release_folder/bbr-s3-config-validator-$VERSION-linux-amd64.sha256"
}

main

