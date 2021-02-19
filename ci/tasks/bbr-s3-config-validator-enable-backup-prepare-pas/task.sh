#!/usr/bin/env bash

set -eu

pool_metadata="env-pool/metadata"

OPS_MAN_URL=$(< "$pool_metadata" jq -r .ops_manager.url)
OPS_MAN_USER=$(< "$pool_metadata" jq -r .ops_manager.username)
OPS_MAN_PASS=$(< "$pool_metadata" jq -r .ops_manager.password)

OM_CMD="om --target ${OPS_MAN_URL} --username ${OPS_MAN_USER} --password ${OPS_MAN_PASS} -k"

$OM_CMD curl -p /api/v0/deployed/products > deployed_products.json
CF_DEPLOYMENT_NAME=$(jq -r '.[] | select( .type | contains("cf")) | .guid' "deployed_products.json")

is_srt=$($OM_CMD curl -p "/api/v0/staged/products/$CF_DEPLOYMENT_NAME/resources" | jq '.resources | map(select(.identifier=="compute")) | length')

if [[ $is_srt -gt 0 ]]; then
  $OM_CMD configure-product --product-name cf --product-resources '{
    "compute": {
      "instances": 2
    }
  }'
fi
backup_restore_resource="$($OM_CMD curl --path "/api/v0/staged/products/${CF_DEPLOYMENT_NAME}/resources" | jq '.resources[] | select(.identifier=="backup_restore" )')"

export backup_restore_product_resource_name="backup_restore"

if [[ ${backup_restore_resource} = "" ]]; then
  export backup_restore_product_resource_name="backup-prepare"
fi

$OM_CMD configure-product --product-name cf --product-resources "$(printf '{"%s": {"instances": 1}}' $backup_restore_product_resource_name)"

if [ "$SKIP_APPLY_CHANGES" = false ] ; then
  $OM_CMD apply-changes
fi
