#!/usr/bin/env bash

set -euo pipefail
set -x

VERSION=$(cat version/version)
release_folder='packaged-release'

touch release_metadata/empty-file

function copy_bbr_binaries {
  binary_name="bbr-${VERSION}-linux-amd64"
  cp 'bbr-build/releases/bbr' "$release_folder/${binary_name}"
  grep 'bbr$' 'bbr-build/releases/checksum.sha256'  | cut -d' ' -f1 > "$release_folder/${binary_name}.sha256"

  binary_name="bbr-${VERSION}-linux-arm64"
  cp 'bbr-build/releases/bbr-arm64' "$release_folder/${binary_name}"
  grep 'bbr-arm64$' 'bbr-build/releases/checksum.sha256'  | cut -d' ' -f1 > "$release_folder/${binary_name}.sha256"

  binary_name="bbr-$VERSION-darwin-amd64"
  cp 'bbr-build/releases/bbr-mac' "$release_folder/${binary_name}"
  grep 'bbr-mac$' 'bbr-build/releases/checksum.sha256'  | cut -d' ' -f1 > "$release_folder/${binary_name}.sha256"

  binary_name="bbr-$VERSION-darwin-arm64"
  cp 'bbr-build/releases/bbr-mac-arm64' "$release_folder/${binary_name}"
  grep 'bbr-mac-arm64$' 'bbr-build/releases/checksum.sha256'  | cut -d' ' -f1 > "$release_folder/${binary_name}.sha256"
}

function copy_s3_config_validator {
  cp 'bbr-s3-config-validator-build/README.md' "$release_folder/bbr-s3-config-validator-$VERSION.README.md"
  cp 'bbr-s3-config-validator-build/bbr-s3-config-validator' "$release_folder/bbr-s3-config-validator-$VERSION-linux-amd64"
  cp 'bbr-s3-config-validator-build/bbr-s3-config-validator.sha256' "$release_folder/bbr-s3-config-validator-$VERSION-linux-amd64.sha256"
}

function create_bbr_tarball() {
  bbr_tarball_folder="$(mktemp -d)"

  mkdir -p "$bbr_tarball_folder/releases/"

  cp 'bbr-build/releases/bbr' "$bbr_tarball_folder/releases/"
  cp 'bbr-build/releases/bbr-mac' "$bbr_tarball_folder/releases/"
  cp 'bbr-build/releases/checksum.sha256' "$bbr_tarball_folder/releases/"

  cp 'bbr-s3-config-validator-build/bbr-s3-config-validator' "$bbr_tarball_folder/releases/"
  cp 'bbr-s3-config-validator-build/README.md' "$bbr_tarball_folder/releases/bbr-s3-config-validator.README.md"

  echo "$(cat bbr-s3-config-validator-build/bbr-s3-config-validator.sha256)  bbr-s3-config-validator" >> "$bbr_tarball_folder/releases/checksum.sha256"

  tar -cf "$release_folder/bbr-${VERSION}.tar" -C "$bbr_tarball_folder" .
}

pushd bbr-build
  tar xf bbr-1.9.74.tar
popd
copy_bbr_binaries
copy_s3_config_validator
create_bbr_tarball
