#!/usr/bin/env bash

set -eu
set -o pipefail

VERSION="$(cat "bbr-release/version")"
SHA256="$(shasum -a 256 "bbr-release/bbr-${VERSION}.tar" | cut -d ' ' -f 1)"

pushd homebrew-tap
  cat <<EOF > bbr.rb
class Bbr < Formula
  desc "BOSH Backup and Restore CLI"
  homepage "https://github.com/cloudfoundry-incubator/bosh-backup-and-restore"
  url "https://github.com/cloudfoundry-incubator/bosh-backup-and-restore/releases/download/v${VERSION}/bbr-${VERSION}.tar"
  sha256 "${SHA256}"

  depends_on :arch => :x86_64

  def install
    bin.install "bbr-mac" => "bbr"
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
