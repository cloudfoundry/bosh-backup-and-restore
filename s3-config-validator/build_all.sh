#!/usr/bin/env bash

set -euo pipefail

cd "$(dirname "$0")"

main() {
  rm -rf build
  mkdir -p build

  build linux amd64
  create_hash
}

build() {
  local os=$1
  local arch=$2

  echo "building bbr-s3-config-validator with os=${os},arch=${arch} ..."

  pushd src/ > /dev/null
    GOOS=$os GOARCH=$arch go build -o "../build/bbr-s3-config-validator" cmd/main.go
  popd > /dev/null

}

create_hash() {
  pushd build
    shasum -a 256 bbr-s3-config-validator | awk '{print $1}' > bbr-s3-config-validator.sha256
  popd
}

main

