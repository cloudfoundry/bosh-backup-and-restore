#!/usr/bin/env bash
set -euo pipefail

root_dir=$(pwd)

today="$(date '+%m/%d/%Y')"
one_year_from_now="$(date --date='1 year hence' '+%m/%d/%Y')"

cp bbr-artefacts/bbr-${RELEASE_VERSION}*.tar release-files/bbr-${RELEASE_VERSION}.tar
tar -x -f bbr-artefacts/bbr-${RELEASE_VERSION}*.tar --to-stdout ./releases/bbr-mac > release-files/bbr-${RELEASE_VERSION}-darwin-amd64
tar -x -f bbr-artefacts/bbr-${RELEASE_VERSION}*.tar --to-stdout ./releases/bbr > release-files/bbr-${RELEASE_VERSION}-linux-amd64

cp s3-config-validator-artefacts/bbr-s3-config-validator release-files/bbr-s3-config-validator-${RELEASE_VERSION}-linux-amd64
cp s3-config-validator-artefacts/README.md release-files/bbr-s3-config-validator-${RELEASE_VERSION}.README.md

cat > "$root_dir/release-config/release.yml" <<RELEASE_METADATA
---
business_unit: Tanzu and Cloud Health
contact: ${RELEASE_CONTACT}
title: ${RELEASE_TITLE}
product_name: ${RELEASE_PRODUCT_NAME}
display_group: ${RELEASE_DISPLAY_GROUP}
version: ${RELEASE_VERSION}
type: ${RELEASE_TYPE}
status: ${RELEASE_STATUS}
lang: EN
ga_date_mm/dd/yyyy: ${today}
published_date_mm/dd/yyyy: ${today}
end_of_support_date_mm/dd/yyyy: ${one_year_from_now}
export_control_status: SCREENING_REQUIRED
files:
- file: ../release-files/bbr-${RELEASE_VERSION}-darwin-amd64
  description: BOSH Backup and Restore Darwin
- file: ../release-files/bbr-${RELEASE_VERSION}-linux-amd64
  description: BOSH Backup and Restore Linux
- file: ../release-files/bbr-${RELEASE_VERSION}.tar
  description: BOSH Backup and Restore
- file: ../release-files/bbr-s3-config-validator-${RELEASE_VERSION}-linux-amd64
  description: BOSH Backup and Restore S3 Configuration Validator Linux
- file: ../release-files/bbr-s3-config-validator-${RELEASE_VERSION}.README.md
  description: BOSH Backup and Restore S3 Configuration Validator README
RELEASE_METADATA

echo "Release config:"
cat "$root_dir/release-config/release.yml"