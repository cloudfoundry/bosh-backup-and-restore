#!/bin/bash

set -eu

VERSION=$(cat bbr-final-release-version/number)
BBR_REPO="$PWD/bosh-backup-and-restore"
BBR_BUILD="$PWD/bbr-build"
BBR_RELEASE="$PWD/bbr-release/releases"

function main {
  pushd "$BBR_REPO"
    make release
  popd

  copy_release_files "$BBR_REPO/releases/." "${BBR_RELEASE}"
  generate_build_dir
  display_files "$BBR_BUILD"
  add_s3_config_files "bbr-build"
  display_files "$BBR_BUILD"
  display_files "$BBR_RELEASE"
}

function copy_release_files {
  local from_dir=$1
  local to_dir=$2

  echo -e "\nProduct has been successfully built. Copying to ${to_dir} directory..."
  cp -r "${from_dir}" "${to_dir}"
}

function display_files {
  local dir=$1

  echo -e "\nThe directory '${dir}' now contains the following files:"
  ls "${dir}"
}

function generate_build_dir {
  echo -e "\nMoving linux binary to the build directory..."
  binary_name="bbr-${VERSION}-linux-amd64"
  mv "${BBR_REPO}/releases/bbr" "bbr-build/${binary_name}"
  cat "${BBR_REPO}/releases/checksum.sha256" | cut -d' ' -f1  | sed -n '1p' > "bbr-build/${binary_name}.sha256"

  echo -e "Moving mac binary to the build directory..."
  binary_name="bbr-$VERSION-darwin-amd64"
  mv "${BBR_REPO}/releases/bbr-mac" "bbr-build/${binary_name}"
  cat "${BBR_REPO}/releases/checksum.sha256" | cut -d' ' -f1  | sed -n '2p' > "bbr-build/${binary_name}.sha256"
}

function add_s3_config_files {
  echo "Adding s3-config-validator release files"
  pushd "bbr-s3-config-validator-artifact"
    tar -xf ./*.tgz

    echo -e "\nMoving s3-config-validator binary to the build directory..."
    cp README.md "$BBR_BUILD/bbr-s3-config-validator-$VERSION.README.md"
    cp bbr-s3-config-validator "$BBR_BUILD/bbr-s3-config-validator-$VERSION-linux-amd64"
    cp bbr-s3-config-validator.sha256 "$BBR_BUILD/bbr-s3-config-validator-$VERSION-linux-amd64.sha256"

    echo "$(cat bbr-s3-config-validator.sha256)  bbr-s3-config-validator" >> "$BBR_RELEASE/checksum.sha256"
    cp bbr-s3-config-validator "$BBR_RELEASE"
    cp README.md "$BBR_RELEASE/bbr-s3-config-validator.README.md"
  popd
}

main

