#!/usr/bin/env bash

[ -z "$DEBUG" ] || set -x

set -eu
set -o pipefail

[ -d bbr-release/ ]
[ -d homebrew-tap/ ]

VERSION="$(cat "bbr-release/version")"

SHA256_OSX_AMD64="$(echo -n "$(cat bbr-release/bbr-"${VERSION}"-darwin-amd64.sha256)")"
SHA256_OSX_ARM64="$(echo -n "$(cat bbr-release/bbr-"${VERSION}"-darwin-arm64.sha256)")"
SHA256_LINUX_AMD64="$(echo -n "$(cat bbr-release/bbr-"${VERSION}"-linux-amd64.sha256)")"

pushd homebrew-tap
  cat <<EOF > bbr.rb
#
# This code has been generated automatically. Any changes will be overwritten.
#
class Bbr < Formula
  desc "BOSH Backup and Restore CLI"
  homepage "https://github.com/cloudfoundry/bosh-backup-and-restore"

  if OS.mac?
    if Hardware::CPU.arm?
      url "https://github.com/cloudfoundry/bosh-backup-and-restore/releases/download/v${VERSION}/bbr-${VERSION}-darwin-arm64"
      sha256 "${SHA256_OSX_ARM64}"
    else
      url "https://github.com/cloudfoundry/bosh-backup-and-restore/releases/download/v${VERSION}/bbr-${VERSION}-darwin-amd64"
      sha256 "${SHA256_OSX_AMD64}"
    end
  elsif OS.linux?
    url "https://github.com/cloudfoundry/bosh-backup-and-restore/releases/download/v${VERSION}/bbr-${VERSION}-linux-amd64"
    sha256 "${SHA256_LINUX_AMD64}"
  end

  def install
    binary_name = "bbr"

    if OS.mac?
      if Hardware::CPU.arm?
        bin.install "bbr-${VERSION}-darwin-arm64" => binary_name
      else
        bin.install "bbr-${VERSION}-darwin-amd64" => binary_name
      end
    elsif OS.linux?
      bin.install "bbr-${VERSION}-linux-amd64" => binary_name
    end
  end

  test do
    system "#{bin}/bbr", "version"
  end
end
EOF

  git add bbr.rb
  git config --global user.name "PCF Backup & Restore CI"
  git config --global user.email "cf-lazarus@pivotal.io"
  if git commit -m "Release BBR CLI v${VERSION}"; then
    echo "Updated homebrew formula to bbr v${VERSION}"
  else
    echo "No changes to formula"
  fi
popd

cp -r homebrew-tap/. updated-homebrew-tap/
