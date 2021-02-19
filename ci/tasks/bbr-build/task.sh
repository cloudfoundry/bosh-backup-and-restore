#!/bin/bash

set -eu

eval "$(ssh-agent)"
chmod 400 bosh-backup-and-restore-meta/keys/github
ssh-add bosh-backup-and-restore-meta/keys/github

VERSION=$(cat bbr-final-release-version/number)
export VERSION

BBR_REPO="bosh-backup-and-restore"
pushd "$BBR_REPO"
  make release
popd

echo "BBR successfully built. Copying to release directory..."

cp -r "$BBR_REPO/releases" bbr-release

echo "The release directory now contains the following files:"
ls -R bbr-release

echo "Creating release tarball..."
tar -C bbr-release -cf "bbr-build/bbr-$VERSION.tar" .

echo "Auto-delivered in
https://s3-eu-west-1.amazonaws.com/bosh-backup-and-restore-builds/bbr-$VERSION.tar

[Backup and Restore Bot]" > bbr-build/message

echo "Moving linux binary to the build directory..."

LINUX="bbr-$VERSION-linux-amd64"
mv "$BBR_REPO"/releases/bbr bbr-build/"$LINUX"
cat "$BBR_REPO"/releases/checksum.sha256 | cut -d' ' -f1  | sed -n '1p' > bbr-build/"$LINUX".sha256

echo "Moving mac binary to the build directory..."

DARWIN="bbr-$VERSION-darwin-amd64"
mv "$BBR_REPO"/releases/bbr-mac bbr-build/"$DARWIN"
cat "$BBR_REPO"/releases/checksum.sha256 | cut -d' ' -f1  | sed -n '2p' > bbr-build/"$DARWIN".sha256

echo "The build directory now contains the following files:"
ls bbr-build

echo "Done building BBR"
