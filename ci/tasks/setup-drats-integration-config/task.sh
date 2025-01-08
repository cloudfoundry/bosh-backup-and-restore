#!/usr/bin/env bash
# shellcheck disable=SC2034

set -euo pipefail

get_password_from_credhub() {
  local variable_name=$1
  credhub find -j -n "${variable_name}" | jq -r .credentials[].name | xargs credhub get -j -n | jq -r .value
}

get_system_domain() {
  jq -r '.cf.api_url | capture("^api\\.(?<system_domain>.*)$") | .system_domain' \
    cf-deployment-env/metadata
}

setup_env_vars() {
  eval "$(bbl print-env --metadata-file cf-deployment-env/metadata)"
  export SYSTEM_DOMAIN="$(get_system_domain)"
  export JUMPBOX_ADDRESS=$(echo $BOSH_ALL_PROXY | cut -d"@" -f2 | cut -d":" -f1)
}

setup_env_vars

cf_deployment_name="${CF_DEPLOYMENT_NAME}"
cf_api_url="https://api.${SYSTEM_DOMAIN}"
cf_admin_username="admin"
cf_admin_password=$(get_password_from_credhub cf_admin_password)
bosh_environment="$BOSH_ENVIRONMENT"
bosh_client="$BOSH_CLIENT"
bosh_client_secret="$BOSH_CLIENT_SECRET"
bosh_ca_cert="$BOSH_CA_CERT"
ssh_proxy_user="jumpbox"
ssh_proxy_host="${JUMPBOX_ADDRESS}"
ssh_proxy_cidr="10.0.0.0/8"
ssh_proxy_private_key="$(cat "$JUMPBOX_PRIVATE_KEY")"
nfs_service_name="nfs"
nfs_plan_name="Existing"
nfs_broker_user="nfs-broker"
nfs_broker_password=$(get_password_from_credhub nfs-broker-password || echo "")
nfs_broker_url="http://nfs-broker.${SYSTEM_DOMAIN}"
smb_service_name="smb"
smb_plan_name="Existing"
smb_broker_user="admin"
smb_broker_password=$(get_password_from_credhub smb-broker-password || echo "")
smb_broker_url="http://smbbroker.${SYSTEM_DOMAIN}"
credhub_client_name="${CREDHUB_CLIENT}"
credhub_client_secret="${CREDHUB_SECRET}"

configs=( cf_deployment_name
        cf_api_url
        cf_admin_username
        cf_admin_password
        bosh_environment
        bosh_client
        bosh_client_secret
        bosh_ca_cert
        ssh_proxy_user
        ssh_proxy_host
        ssh_proxy_cidr
        ssh_proxy_private_key
        nfs_service_name
        nfs_plan_name
        nfs_broker_user
        nfs_broker_password
        nfs_broker_url
        smb_service_name
        smb_plan_name
        smb_broker_user
        smb_broker_password
        smb_broker_url
        credhub_client_name
        credhub_client_secret )

integration_config="$(cat "integration-configs/${INTEGRATION_CONFIG_FILE_PATH}")"

for config in "${configs[@]}"; do
  integration_config=$(echo "${integration_config}" | jq ".${config}=\"${!config}\"")
done

tests_to_disable=( include_cf-credhub
  include_cf-nfsbroker
  include_cf-smbbroker )

for test_to_disable in "${tests_to_disable[@]}"; do
  integration_config=$(echo "${integration_config}" | jq ".\"${test_to_disable}\"=false")
done


echo "${integration_config}" > "integration-configs/${INTEGRATION_CONFIG_FILE_PATH}"

cp -Tr integration-configs updated-integration-configs
