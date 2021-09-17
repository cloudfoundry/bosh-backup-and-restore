#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$PWD"
GITHUB_FOLDER="${ROOT_DIR}/github-release-with-metadata"
RELEASE_FOLDER="${ROOT_DIR}/release"
RELEASE_TAR_FOLDER="${ROOT_DIR}/release-tar"
VERSION=$(cat "version-folder/${VERSION_PATH}")

function main {
  create_tarball
  copy_tarball_to_folder "${GITHUB_FOLDER}"
  copy_release_files_to_folder "${GITHUB_FOLDER}"
  delete_sha256_files
  export_release_metadata_variables
}

function create_tarball {
  echo "Creating release tarball..."
  export TAR_NAME="bbr-${VERSION}.tar"
  tar -cf "${TAR_NAME}" -C "${RELEASE_TAR_FOLDER}" .
}

function copy_tarball_to_folder {
  echo "Adding tarball to: $1"
  cp "${TAR_NAME}" "$1"
}

function copy_release_files_to_folder {
  cp -r "${RELEASE_FOLDER}/." "$1"
}

main
