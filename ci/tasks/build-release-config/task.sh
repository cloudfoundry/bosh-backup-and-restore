#!/usr/bin/env bash
set -euo pipefail

root_dir=$(pwd)

today="$(date '+%m/%d/%Y')"
one_year_from_now="$(date -v +1y '+%m/%d/%Y')"
docs_link="https://docs.vmware.com/en/Compliance-Scanner-for-VMware-Tanzu/1.3/addon-compliance-tools/GUID-index.html"

# copy all the stuff to release-files dir

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
docs_link: "${docs_link}"
ga_date_mm/dd/yyyy: ${today}
published_date_mm/dd/yyyy: ${today}
end_of_support_date_mm/dd/yyyy: ${one_year_from_now}
export_control_status: SCREENING_REQUIRED
files:
- file: ../release-files/bbr-${RELEASE_VERSION}-darwin-amd64
- file: ../release-files/bbr-${RELEASE_VERSION}-linux-amd64
- file: ../release-files/bbr-${RELEASE_VERSION}.tar
- file: ../release-files/bbr-s3-config-validator-${RELEASE_VERSION}-linux-amd64
- file: ../release-files/bbr-s3-config-validator-${RELEASE_VERSION}.README.md
RELEASE_METADATA

echo "Release config:"
cat "$root_dir/release-config/release.yml"