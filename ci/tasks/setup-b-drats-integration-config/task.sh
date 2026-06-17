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
  pushd environment
    eval "$(bbl print-env)"
  popd

  # Modern bbl emits BOSH_ALL_PROXY (ssh+socks5://user@host:22?private-key=...)
  # and JUMPBOX_PRIVATE_KEY instead of the legacy BOSH_GW_HOST / BOSH_GW_PRIVATE_KEY.
  local jumpbox_host
  jumpbox_host="$(echo "${BOSH_ALL_PROXY}" | cut -d"@" -f2 | cut -d":" -f1)"

  # BOSH_ENVIRONMENT is a URL (https://<IP>:25555); BBR needs just the IP for SSH.
  local bosh_host
  bosh_host="$(echo "${BOSH_ENVIRONMENT}" | sed 's|https://||' | cut -d: -f1)"

  local jumpbox_user
  jumpbox_user="${BOSH_GW_USER:-jumpbox}"

  local jumpbox_privkey_path
  jumpbox_privkey_path="${JUMPBOX_PRIVATE_KEY:-${BOSH_GW_PRIVATE_KEY}}"

  local jumpbox_privkey_raw
  jumpbox_privkey_raw="$(cat "${jumpbox_privkey_path}")"

  local jumpbox_pubkey_raw
  jumpbox_pubkey_raw="$(ssh-keygen -y -f "${jumpbox_privkey_path}")"

  # The BOSH director's 'jumpbox' user is provisioned via jumpbox-user.yml with a
  # BOSH-generated key pair (jumpbox_ssh), stored in the bbl state vars-store.
  # This is NOT the same as the jumpbox VM's SSH key.
  local bosh_ssh_private_key_raw
  bosh_ssh_private_key_raw="$(bosh int environment/vars/director-vars-store.yml --path /jumpbox_ssh/private_key)"

  local stemcell_src
  stemcell_src="$(cat stemcell/url)"

  local stemcell_os
  stemcell_os="$(tar --occurrence --to-stdout -xf stemcell/stemcell.tgz stemcell.MF | yq '.operating_system')"

  jq -n \
    --arg bosh_host "${bosh_host}" \
    --arg bosh_client "${BOSH_CLIENT}" \
    --arg bosh_client_secret "${BOSH_CLIENT_SECRET}" \
    --arg bosh_ssh_private_key "${bosh_ssh_private_key_raw}" \
    --arg bosh_ca_cert "${BOSH_CA_CERT}" \
    --arg credhub_client "${CREDHUB_CLIENT}" \
    --arg credhub_client_secret "${CREDHUB_SECRET}" \
    --arg credhub_ca_cert "${BOSH_CA_CERT}" \
    --arg credhub_server "${CREDHUB_SERVER}" \
    --arg stemcell_src "${stemcell_src}" \
    --arg stemcell_os "${stemcell_os}" \
    --argjson include_deployment_testcase "${INCLUDE_DEPLOYMENT_TESTCASE}" \
    --argjson include_truncate_db_blobstore_testcase "${INCLUDE_TRUNCATE_DB_BLOBSTORE_TESTCASE}" \
    --argjson include_credhub_testcase "${INCLUDE_CREDHUB_TESTCASE}" \
    --argjson timeout_in_minutes "${TIMEOUT_IN_MINUTES}" \
    --arg jumpbox_host "${jumpbox_host}" \
    --arg jumpbox_user "${jumpbox_user}" \
    --arg jumpbox_pubkey "${jumpbox_pubkey_raw}" \
    --arg jumpbox_privkey "${jumpbox_privkey_raw}" \
    '{
      bosh_host: $bosh_host,
      bosh_client: $bosh_client,
      bosh_client_secret: $bosh_client_secret,
      bosh_ssh_username: "jumpbox",
      bosh_ssh_private_key: $bosh_ssh_private_key,
      bosh_ca_cert: $bosh_ca_cert,
      credhub_client_secret: $credhub_client_secret,
      credhub_client: $credhub_client,
      credhub_ca_cert: $credhub_ca_cert,
      credhub_server: $credhub_server,
      stemcell_src: $stemcell_src,
      stemcell_os: $stemcell_os,
      include_deployment_testcase: $include_deployment_testcase,
      include_truncate_db_blobstore_testcase: $include_truncate_db_blobstore_testcase,
      include_credhub_testcase: $include_credhub_testcase,
      timeout_in_minutes: $timeout_in_minutes,
      deployment_vm_type: "minimal",
      deployment_network: "default",
      deployment_az: "z1",
      jumpbox_host: $jumpbox_host,
      jumpbox_user: $jumpbox_user,
      jumpbox_pubkey: $jumpbox_pubkey,
      jumpbox_privkey: $jumpbox_privkey
    }' > config/integration_config.json
}

main
