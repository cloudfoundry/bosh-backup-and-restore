#!/bin/bash

set -eu

if [ -d version ]; then
  export VERSION=$(cat version/version)
else
  export VERSION=$(date +%s)
fi

BBR_REPO="bosh-backup-and-restore"
pushd "$BBR_REPO"
  make release
popd

echo "BBR successfully built. Copying to release directory..."

cp -r "$BBR_REPO/releases" bbr-release

echo "The release directory now contains the following files:"
ls -R bbr-release

echo "Creating release tarball..."
tar -C bbr-release -cf "bbr-build/bbr-${VERSION}.tar" .
echo "Auto-delivered in
https://s3-eu-west-1.amazonaws.com/bosh-backup-and-restore-builds/bbr-${VERSION}.tar

[Backup and Restore Bot]" > bbr-build/message

echo "Moving linux binaries to the build directory..."

LINUX="bbr-${VERSION}-linux-amd64"
mv "$BBR_REPO"/releases/bbr bbr-build/"$LINUX"
cat "$BBR_REPO"/releases/checksum.sha256 | cut -d' ' -f1  | sed -n '1p' > bbr-build/"$LINUX".sha256

LINUX_ARM64="bbr-${VERSION}-linux-arm64"
mv "$BBR_REPO"/releases/bbr-arm64 bbr-build/"$LINUX_ARM64"
cat "$BBR_REPO"/releases/checksum.sha256 | cut -d' ' -f1  | sed -n '1p' > bbr-build/"$LINUX_ARM64".sha256

echo "Moving mac binaries to the build directory..."

DARWIN="bbr-${VERSION}-darwin-amd64"
mv "$BBR_REPO"/releases/bbr-mac bbr-build/"$DARWIN"
cat "$BBR_REPO"/releases/checksum.sha256 | cut -d' ' -f1  | sed -n '2p' > bbr-build/"$DARWIN".sha256

DARWIN_ARM64="bbr-${VERSION}-darwin-arm64"
mv "$BBR_REPO"/releases/bbr-mac-arm64 bbr-build/"$DARWIN_ARM64"
cat "$BBR_REPO"/releases/checksum.sha256 | cut -d' ' -f1  | sed -n '2p' > bbr-build/"$DARWIN_ARM64".sha256

echo "The build directory now contains the following files:"
ls bbr-build

echo "Done building BBR"
