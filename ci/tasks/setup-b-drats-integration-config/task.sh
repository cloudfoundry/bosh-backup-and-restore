#!/usr/bin/env bash

set -euo pipefail

[[ -d environment ]]
[[ -d config ]]
[[ -d stemcell ]]

: "${INCLUDE_DEPLOYMENT_TESTCASE:?}"
: "${INCLUDE_TRUNCATE_DB_BLOBSTORE_TESTCASE:?}"
: "${INCLUDE_CREDHUB_TESTCASE:?}"
: "${TIMEOUT_IN_MINUTES:?}"

main() {
  check_binary_exists \
    yq \
    om

  if [ -f "environment/pcf.yml" ]; then
      OM_TARGET="$(yq -r '.target' 'environment/pcf.yml')"
      OM_USERNAME="$(yq -r '.username' 'environment/pcf.yml')"
      OM_PASSWORD="$(yq -r '.password' 'environment/pcf.yml')"
  else
      OM_TARGET="$(yq -r '.ops_manager.url' 'environment/metadata')"
      OM_USERNAME="$(yq -r '.ops_manager.username' 'environment/metadata')"
      OM_PASSWORD="$(yq -r '.ops_manager.password' 'environment/metadata')"
  fi

  export \
    OM_TARGET \
    OM_USERNAME \
    OM_PASSWORD \
    OM_SKIP_SSL_VALIDATION=true


  local jumpbox_host
  jumpbox_host=$( jq .ops_manager_dns -r < environment/metadata  )

  local jumpbox_user
  jumpbox_user=ubuntu

  local jumpbox_pubkey
  jumpbox_pubkey=$( jq .ops_manager_public_key < environment/metadata )

  local jumpbox_privkey
  jumpbox_privkey=$( jq .ops_manager_private_key < environment/metadata )

  local bosh_commandline_credentials
  bosh_commandline_credentials="$(om curl --silent --path /api/v0/deployed/director/credentials/bosh_commandline_credentials)"

  local \
    bosh_environment \
    bosh_client \
    bosh_client_secret \

  bosh_environment=$(get_bosh_environment "$bosh_commandline_credentials")
  bosh_client=$(get_bosh_client "$bosh_commandline_credentials")
  bosh_client_secret=$(get_bosh_client_secret "$bosh_commandline_credentials")

  local bbr_ssh_credentials
  bbr_ssh_credentials="$(om curl --silent --path /api/v0/deployed/director/credentials/bbr_ssh_credentials)"

  local bosh_ssh_private_key
  bosh_ssh_private_key="$(jq '.credential.value.private_key_pem' <(echo "$bbr_ssh_credentials"))"

  local bosh_ca_cert
  bosh_ca_cert="$(get_bosh_ca_cert)"

  local stemcell_src
  stemcell_src="$(cat stemcell/url)"

  local az
  az="$(jq -r '.azs[0]' environment/metadata)"

  local env_name
  if [ -f "environment/name" ]; then
      env_name="$(cat environment/name)"
  else
      env_name="$( jq -r '.name' < environment/metadata )"
  fi

  cat << EOF > config/integration_config.json
{
  "bosh_host": "$bosh_environment",
  "bosh_client": "$bosh_client",
  "bosh_client_secret": "$bosh_client_secret",
  "bosh_ssh_username": "bbr",
  "bosh_ssh_private_key": $bosh_ssh_private_key,
  "bosh_ca_cert": "$bosh_ca_cert",
  "credhub_client_secret": "$bosh_client_secret",
  "credhub_client": "$bosh_client",
  "credhub_ca_cert": "$bosh_ca_cert",
  "credhub_server": "https://${bosh_environment}:8844",
  "stemcell_src": "$stemcell_src",
  "include_deployment_testcase": $INCLUDE_DEPLOYMENT_TESTCASE,
  "include_truncate_db_blobstore_testcase": $INCLUDE_TRUNCATE_DB_BLOBSTORE_TESTCASE,
  "include_credhub_testcase": $INCLUDE_CREDHUB_TESTCASE,
  "timeout_in_minutes": $TIMEOUT_IN_MINUTES,
  "deployment_vm_type": "default",
  "deployment_network": "network",
  "deployment_az": "null",
  "jumpbox_host": "$jumpbox_host",
  "jumpbox_user": "$jumpbox_user",
  "jumpbox_pubkey": $jumpbox_pubkey,
  "jumpbox_privkey": $jumpbox_privkey
}
EOF
}

check_binary_exists() {
  local binaries
  binaries=("$@")

  for binary in "${binaries[@]}"
  do
    if ! command -v "$binary" > /dev/null; then \
      echo "'$binary' required, but not found in PATH."; \
      exit 1; \
    fi;
  done
}

get_bosh_environment() {
  local bosh_commandline_credentials="${1:?}"

  jq -r '.credential' <(echo "$bosh_commandline_credentials") | sed -n -E 's/.*BOSH_ENVIRONMENT=(([0-9]{1,3}\.){3}[0-9]{1,3}).*/\1/p'
}

get_bosh_client(){
  local bosh_commandline_credentials="${1:?}"

  jq -r '.credential' <(echo "$bosh_commandline_credentials") | sed -n -E 's/.*BOSH_CLIENT=([^[:space:]]+).*/\1/p'
}

get_bosh_client_secret() {
  local bosh_commandline_credentials="${1:?}"

  jq -r '.credential' <(echo "$bosh_commandline_credentials") | sed -n -E 's/.*BOSH_CLIENT_SECRET=([^[:space:]]+).*/\1/p'
}

get_bosh_ca_cert() {
  local ops_manager_private_key
  ops_manager_private_key="$(mktemp)"
  chmod 0600 "$ops_manager_private_key"
  jq -r '.ops_manager_private_key' environment/metadata > "$ops_manager_private_key"

  local ops_manager_dns
  ops_manager_dns="$(jq -r '.ops_manager_dns' environment/metadata)"

  local bosh_ca_cert_path
  bosh_ca_cert_path="$(mktemp)"

  scp \
    -o UserKnownHostsFile=/dev/null \
    -o StrictHostKeyChecking=no \
    -i "$ops_manager_private_key" \
    "ubuntu@${ops_manager_dns}:/var/tempest/workspaces/default/root_ca_certificate" "$bosh_ca_cert_path"

  awk '{printf "%s\\n", $0}' "$bosh_ca_cert_path"
}

main
