#!/usr/bin/env bash
# shellcheck disable=SC2034

set -euo pipefail

get_password_from_credhub() {
  local variable_name=$1
  credhub find -j -n "${variable_name}" | jq -r .credentials[].name | xargs credhub get -j -n | jq -r .value
}

setup_env_vars() {
  pushd cf-deployment-env
    eval "$(bbl print-env)"
    # The credhub CLI connects to two TLS endpoints that use different CAs:
    #   - CredHub server (8844): signed by credhub_ca (credhubServerCa)
    #   - UAA server (8443): signed by default_ca (= BOSH_CA_CERT)
    # CREDHUB_CA_CERT must contain both CAs (PEM concatenation is supported).
    CREDHUB_TLS_CA=$(bosh interpolate vars/director-vars-store.yml --path /credhub_tls/ca)
    export CREDHUB_CA_CERT="${CREDHUB_TLS_CA}
${BOSH_CA_CERT}"
    # Extract the director VM's jumpbox SSH private key for diagnostic SSH access.
    # bosh create-env stores this in director-vars-store.yml under jumpbox_ssh.
    export DIRECTOR_SSH_PRIVATE_KEY
    DIRECTOR_SSH_PRIVATE_KEY=$(bosh interpolate vars/director-vars-store.yml --path /jumpbox_ssh/private_key)
  popd
  # SYSTEM_DOMAIN is passed as a pipeline param (e.g. bosh-lite.com)
  # Prefer BOSH_GW_HOST (set by bbl print-env) over parsing BOSH_ALL_PROXY.
  export JUMPBOX_ADDRESS="${BOSH_GW_HOST:-$(echo "$BOSH_ALL_PROXY" | cut -d"@" -f2 | cut -d":" -f1)}"
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
# JUMPBOX_PRIVATE_KEY is set by bbl print-env as a path to a temp file
ssh_proxy_private_key="$(cat "${JUMPBOX_PRIVATE_KEY:-${BOSH_GW_PRIVATE_KEY}}")"
# Director VM SSH key (vcap user) for diagnostic access via the jumpbox.
# Extracted from director-vars-store.yml which bosh create-env populates.
director_ssh_private_key="${DIRECTOR_SSH_PRIVATE_KEY}"
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
        director_ssh_private_key
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
  integration_config=$(echo "${integration_config}" | jq --arg val "${!config}" ".${config}=\$val")
done

tests_to_disable=( include_cf-credhub
  include_cf-nfsbroker
  include_cf-smbbroker )

for test_to_disable in "${tests_to_disable[@]}"; do
  integration_config=$(echo "${integration_config}" | jq ".\"${test_to_disable}\"=false")
done


echo "${integration_config}" > "integration-configs/${INTEGRATION_CONFIG_FILE_PATH}"

cp -Tr integration-configs updated-integration-configs
