#!/bin/bash

set -eu

[ -d 'bosh-backup-and-restore' ]
[ -d 'release-candidate-version' ]
[ -d 'release-version' ]

eval "$(ssh-agent)"
echo -e "${GITHUB_SDK_PRIVATE_KEY}" > "${PWD}/github-sdk.key"
chmod 400 "${PWD}/github-sdk.key"
ssh-add "${PWD}/github-sdk.key"

RELEASE_CANDIDATE_VERSION=$(cat release-candidate-version/number)
VERSION=$(cat release-version/number)
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
tar -C bbr-release -cf "bbr-build/bbr-$RELEASE_CANDIDATE_VERSION.tar" .
echo "Auto-delivered in
https://s3-eu-west-1.amazonaws.com/bosh-backup-and-restore-builds/bbr-$RELEASE_CANDIDATE_VERSION.tar

[Backup and Restore Bot]" > bbr-build/message

echo "Moving linux binary to the build directory..."

LINUX="bbr-$RELEASE_CANDIDATE_VERSION-linux-amd64"
mv "$BBR_REPO"/releases/bbr bbr-build/"$LINUX"
cat "$BBR_REPO"/releases/checksum.sha256 | cut -d' ' -f1  | sed -n '1p' > bbr-build/"$LINUX".sha256

echo "Moving mac binary to the build directory..."

DARWIN="bbr-$RELEASE_CANDIDATE_VERSION-darwin-amd64"
mv "$BBR_REPO"/releases/bbr-mac bbr-build/"$DARWIN"
cat "$BBR_REPO"/releases/checksum.sha256 | cut -d' ' -f1  | sed -n '2p' > bbr-build/"$DARWIN".sha256

DARWIN_ARM64="bbr-$RELEASE_CANDIDATE_VERSION-darwin-arm64"
mv "$BBR_REPO"/releases/bbr-mac-arm64 bbr-build/"$DARWIN_ARM64"
cat "$BBR_REPO"/releases/checksum.sha256 | cut -d' ' -f1  | sed -n '2p' > bbr-build/"$DARWIN_ARM64".sha256

echo "The build directory now contains the following files:"
ls bbr-build

echo "Done building BBR"
