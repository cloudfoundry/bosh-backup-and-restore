#!/usr/bin/env bash

set -eu

pool_metadata="env-pool/metadata"

OPS_MAN_URL=$(< "$pool_metadata" jq -r .ops_manager.url)
OPS_MAN_USER=$(< "$pool_metadata" jq -r .ops_manager.username)
OPS_MAN_PASS=$(< "$pool_metadata" jq -r .ops_manager.password)

OM_CMD="om --target ${OPS_MAN_URL} --username ${OPS_MAN_USER} --password ${OPS_MAN_PASS} -k"

if [[ ! -z "${BACKUP_BUCKET}"  ]]; then
    UNVERSIONSED_PROPERTIES="
    \".properties.system_blobstore.external.versioning\": {
      \"value\": false
    },
    \".properties.system_blobstore.external.backup_region\": {
      \"value\": \"${BACKUP_REGION}\"
    },
    \".properties.system_blobstore.external.buildpacks_backup_bucket\": {
      \"value\": \"${BACKUP_BUCKET}\"
    },
    \".properties.system_blobstore.external.droplets_backup_bucket\": {
      \"value\": \"${BACKUP_BUCKET}\"
    },
    \".properties.system_blobstore.external.packages_backup_bucket\": {
      \"value\": \"${BACKUP_BUCKET}\"
    }
"
else
    UNVERSIONSED_PROPERTIES="
    \".properties.system_blobstore.external.versioning\": {
      \"value\": true
    }"
fi

$OM_CMD configure-product --product-name cf --product-properties "{
    \".properties.system_blobstore\": {
      \"value\": \"external\"
    },
    \".properties.system_blobstore.external.endpoint\": {
      \"value\": \"${ENDPOINT}\"
    },
    \".properties.system_blobstore.external.buildpacks_bucket\": {
      \"value\": \"${BUILDPACKS_BUCKET}\"
    },
    \".properties.system_blobstore.external.droplets_bucket\": {
      \"value\": \"${DROPLETS_BUCKET}\"
    },
    \".properties.system_blobstore.external.packages_bucket\": {
      \"value\": \"${PACKAGES_BUCKET}\"
    },
    \".properties.system_blobstore.external.resources_bucket\": {
      \"value\": \"${RESOURCES_BUCKET}\"
    },
    \".properties.system_blobstore.external.access_key\": {
      \"value\": \"${ACCESS_KEY}\"
    },
    \".properties.system_blobstore.external.secret_key\": {
      \"value\": {\"secret\": \"${SECRET_KEY}\"}
    },
    \".properties.system_blobstore.external.signature_version\": {
      \"value\": \"4\"
    },
    \".properties.system_blobstore.external.region\": {
      \"value\": \"${REGION}\"
    },
    \".properties.system_blobstore.external.encryption\": {
      \"value\": false
    }, ${UNVERSIONSED_PROPERTIES}
}"

if [ "$SKIP_APPLY_CHANGES" = false ] ; then
    $OM_CMD apply-changes
fi
