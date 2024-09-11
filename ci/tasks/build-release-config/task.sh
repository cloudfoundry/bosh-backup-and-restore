#!/usr/bin/env bash
set -euo pipefail
set -x

root_dir=$(pwd)

release_version=$(cat scanner-version/version)
today="$(date '+%m/%d/%Y')"
docs_link="https://docs.vmware.com/en/Compliance-Scanner-for-VMware-Tanzu/1.3/addon-compliance-tools/GUID-index.html"

tar xf scanner-tile-tarball/compliance-scanner-*.tar.gz --directory=release-config

cat > "$root_dir/release-config/release.yml" <<RELEASE_METADATA
---
business_unit: Tanzu and Cloud Health
contact: ${RELEASE_CONTACT}
title: ${RELEASE_TITLE}
product_name: ${RELEASE_PRODUCT_NAME}
display_group: ${RELEASE_DISPLAY_GROUP}
version: ${release_version}
type: ${RELEASE_TYPE}
status: ${RELEASE_STATUS}
lang: EN
docs_link: "${docs_link}"
ga_date_mm/dd/yyyy: ${today}
published_date_mm/dd/yyyy: ${today}
end_of_support_date_mm/dd/yyyy: ${RELEASE_END_OF_SUPPORT}
export_control_status: SCREENING_REQUIRED
files:
- file: ../release-config/CIS-Kubernetes-${release_version}.pdf
  description: Compliance Scanner Benchmark  - CIS-Kubernetes-${release_version}
- file: ../release-config/Jammy-CIS-Level-1-${release_version}.pdf
  description: Compliance Scanner Benchmark  - Jammy-CIS-Level-1-${release_version}
- file: ../release-config/Jammy-CIS-Level-2-${release_version}.pdf
  description: Compliance Scanner Benchmark  - Jammy-CIS-Level-2-${release_version}
- file: ../release-config/STIG-jammy-${release_version}.pdf
  description: Compliance Scanner Benchmark  - STIG-jammy-${release_version}
- file: ../release-config/STIG-Kubernetes-${release_version}.pdf
  description: Compliance Scanner Benchmark  - STIG-Kubernetes-${release_version}
- file: ../release-config/STIG-xenial-${release_version}.pdf
  description: Compliance Scanner Benchmark  - STIG-xenial-${release_version}
- file: ../release-config/Xenial-CIS-Level-1-${release_version}.pdf
  description: Compliance Scanner Benchmark  - Xenial-CIS-Level-1-${release_version}
- file: ../release-config/Xenial-CIS-Level-2-${release_version}.pdf
  description: Compliance Scanner Benchmark  - Xenial-CIS-Level-2-${release_version}
RELEASE_METADATA

if [ -n "$RELEASE_UPGRADE_SPECIFIERS" ]; then
  cat >> "$root_dir/release-config/release.yml" <<EOF
upgrade_specifiers:
EOF
  IFS=',' ;for i in ${RELEASE_UPGRADE_SPECIFIERS}; do
  cat >> "$root_dir/release-config/release.yml" <<EOF
- specifier: ${i}
EOF
  done
fi
