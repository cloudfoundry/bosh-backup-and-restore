#!/usr/bin/env bash
set -euo pipefail
[ -n "${DEBUG:-}" ] && set -x

bbl_up() {
  bbl plan
  rm -rf bosh-deployment
  cp -rfp "${bosh_deployment}" .
  cp -R "${bosh_bootloader}/plan-patches/bosh-lite-gcp/." .

  # GCP labels applied to the BOSH director VM (created by bosh create-env).
  # Using individual key paths (labels?/key?) so that any labels already present
  # in cloud_properties are preserved rather than replaced.
  cat > gcp-labels-director.yml << OPSEOF
---
- type: replace
  path: /resource_pools/name=vms/cloud_properties/labels?/pipeline?
  value: bbr-cli
- type: replace
  path: /resource_pools/name=vms/cloud_properties/labels?/pipeline-job?
  value: ${PIPELINE_JOB}
OPSEOF

  # GCP labels applied to the jumpbox VM (created by bosh create-env).
  cat > gcp-labels-jumpbox.yml << OPSEOF
---
- type: replace
  path: /resource_pools/name=vms/cloud_properties/labels?/pipeline?
  value: bbr-cli
- type: replace
  path: /resource_pools/name=vms/cloud_properties/labels?/pipeline-job?
  value: ${PIPELINE_JOB}
OPSEOF

  # Inject ops-file flags into create-director-override.sh.
  sed '$ s/$/ \\/' create-director-override.sh > /tmp/create-director-override.sh
  printf ' -o ${BBL_STATE_DIR}/bosh-deployment/bbr.yml \\\n' >> /tmp/create-director-override.sh
  if [[ "${WARDEN_CONTAINERS_USE_SYSTEMD:-true}" == "false" ]]; then
    cat > bosh-lite-jammy-containers.yml << OPSEOF
---
- type: replace
  path: /instance_groups/name=bosh/properties/warden_cpi/start_containers_with_systemd?
  value: false
OPSEOF
    printf ' -o ${BBL_STATE_DIR}/bosh-lite-jammy-containers.yml \\\n' >> /tmp/create-director-override.sh
  fi
  if [[ "${DISABLE_HM_RESURRECTOR:-false}" == "true" ]]; then
    cat > bosh-disable-resurrector.yml << OPSEOF
---
- type: replace
  path: /instance_groups/name=bosh/properties/hm/resurrector_enabled
  value: false
OPSEOF
    printf ' -o ${BBL_STATE_DIR}/bosh-disable-resurrector.yml \\\n' >> /tmp/create-director-override.sh
  fi

  # bosh-lite.yml pins os-conf v18 which only has basic jobs (sysctl, disable_agent, etc.).
  # Upgrade to v23 so we can use pre-start-script and the iptables job, both of which
  # were added in later releases and are required for the director VM configuration below.
  cat > bosh-os-conf-upgrade.yml << 'OPSEOF'
---
- path: /releases/name=os-conf
  type: replace
  value:
    name: os-conf
    sha1: "sha256:efcf30754ce4c5f308aedab3329d8d679f5967b2a4c3c453204c7cb10c7c5ed9"
    url: https://bosh.io/d/github.com/cloudfoundry/os-conf-release?v=23.0.0
    version: "23.0.0"
OPSEOF
  printf ' -o ${BBL_STATE_DIR}/bosh-os-conf-upgrade.yml \\\n' >> /tmp/create-director-override.sh

  # Noble warden containers share the host kernel's inotify limits. With 15+
  # simultaneous warden containers each running systemd (and apps running Envoy),
  # the default fs.inotify.max_user_instances=128 is quickly exhausted, causing
  # systemd and Envoy to abort with "inotify_fd_ >= 0 assert failure". Setting a
  # high limit ensures every process can create inotify watches without contention.
  cat > bosh-inotify-limits.yml << OPSEOF
---
- path: /instance_groups/name=bosh/jobs/-
  type: replace
  value:
    name: sysctl
    release: os-conf
    properties:
      sysctl:
      - fs.inotify.max_user_instances=65536
      - fs.inotify.max_user_watches=1048576
OPSEOF
  printf ' -o ${BBL_STATE_DIR}/bosh-inotify-limits.yml \\\n' >> /tmp/create-director-override.sh

  # The GCP VPC has a static route for 10.244.0.0/16 pointing to the director
  # VM as next-hop. The director must forward packets from the jumpbox (10.0.0.x)
  # to warden containers (10.244.x.x). When the Linux kernel's default iptables
  # FORWARD policy is DROP (common on GCP VMs), these packets are silently dropped,
  # making container IPs unreachable from the jumpbox and breaking sshuttle.
  # The pre-start-script job from os-conf runs on the director VM at BOSH startup
  # and sets the FORWARD policy to ACCEPT before the BOSH director starts, so that
  # the route works from the moment the director is reachable.
  cat > bosh-forward-iptables.yml << 'OPSEOF'
---
- path: /instance_groups/name=bosh/jobs/-
  type: replace
  value:
    name: pre-start-script
    release: os-conf
    properties:
      script: |
        #!/bin/bash
        iptables -P FORWARD ACCEPT || true
        echo 1 > /proc/sys/net/ipv4/ip_forward || true
OPSEOF
  printf ' -o ${BBL_STATE_DIR}/bosh-forward-iptables.yml \\\n' >> /tmp/create-director-override.sh
  printf ' -o ${BBL_STATE_DIR}/gcp-labels-director.yml\n' >> /tmp/create-director-override.sh
  cp /tmp/create-director-override.sh create-director-override.sh
  chmod +x create-director-override.sh

  # Inject labels ops-file flag into create-jumpbox-override.sh.
  sed '$ s/$/ \\/' create-jumpbox-override.sh > /tmp/create-jumpbox-override.sh
  printf ' -o ${BBL_STATE_DIR}/gcp-labels-jumpbox.yml\n' >> /tmp/create-jumpbox-override.sh
  cp /tmp/create-jumpbox-override.sh create-jumpbox-override.sh
  chmod +x create-jumpbox-override.sh

  # Add GCP labels to the Terraform-managed external IP resources.
  # google_compute_address supports labels; other bbl resources (firewall rules,
  # network, subnet, NAT router) do not.
  cat > terraform/labels_override.tf << EOF
resource "google_compute_address" "jumpbox-ip" {
  labels = {
    pipeline     = "bbr-cli"
    pipeline-job = "${PIPELINE_JOB}"
  }
}

resource "google_compute_address" "bosh-director-ip" {
  labels = {
    pipeline     = "bbr-cli"
    pipeline-job = "${PIPELINE_JOB}"
  }
}
EOF

  # The bosh-lite-gcp plan-patch uses short_env_id (first 20 chars of env_id) for
  # firewall rule names. Two concurrent environments that share the same lake name
  # in the same year (e.g. bbl-env-qinghai-2026-...) collide because both truncate
  # to "bbl-env-qinghai-2026". Increasing to 32 chars includes the full date+hour,
  # making the name unique per run while staying within GCP's 63-char resource name
  # limit (32-char prefix + 31-char longest suffix "-bosh-director-lite-tcp-routing").
  sed -i 's/min(20, length(var.env_id))/min(32, length(var.env_id))/' terraform/bosh-lite.tf

  bbl --debug up
}

bosh_deployment="$PWD/bosh-deployment"
bosh_bootloader="$PWD/bosh-bootloader"

pushd "${PWD}/bbl-state"
  bbl_up

  # After bbl up, configure BOSH DNS to use 8.8.8.8 as its upstream recursor.
  # In BOSH-lite on GCP, CF VMs (warden containers) run on the 10.244.x.x
  # network. GCP's link-local metadata DNS (169.254.169.254) is not routable
  # from within warden containers; BOSH DNS therefore cannot forward external
  # queries and returns SERVFAIL (visible as "server misbehaving"). This causes
  # CF app staging to fail when buildpacks download dependencies from the
  # internet (e.g. nginx from buildpacks.cloudfoundry.org).
  #
  # We patch the bbl-created "dns" runtime config in place rather than creating
  # a separate named config. A separate config causes BOSH to try to add
  # bosh-dns twice to every instance group, producing a "job already added"
  # error at CF deploy time.
  eval "$(bbl print-env)"
  cat > /tmp/dns-recursors-ops.yml << 'OPSEOF'
- type: replace
  path: /addons/name=bosh-dns/jobs/name=bosh-dns/properties/recursors?
  value:
  - 8.8.8.8
  - 8.8.4.4
OPSEOF
  bosh runtime-config --name dns > /tmp/current-dns-rc.yml
  bosh int /tmp/current-dns-rc.yml -o /tmp/dns-recursors-ops.yml > /tmp/modified-dns-rc.yml
  bosh update-runtime-config /tmp/modified-dns-rc.yml --name dns --non-interactive

popd
