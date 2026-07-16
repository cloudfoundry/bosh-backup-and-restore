#!/usr/bin/env bash
set -euo pipefail
[ -n "${DEBUG:-}" ] && set -x

bbl_up() {
  # Generate a unique environment name with the bbr-cli- prefix.
  # bbl's --name flag sets the *full* env ID (no random suffix is appended by bbl
  # itself), so we generate the unique portion here: a timestamp (minute resolution)
  # combined with a 4-char random hex suffix to avoid collisions between concurrent
  # jobs that start within the same minute.
  local env_name
  env_name="bbr-cli-$(date -u +%Y-%m-%dt%H-%Mz)-$(openssl rand -hex 2)"
  bbl plan --name "${env_name}"
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

  # The acceptance-tests task routes sshuttle THROUGH the director (via SSH
  # ProxyJump through the jumpbox) so the director's local warden bridge reaches
  # containers directly — no GCP VPC next-hop IP-forwarding is needed.
  # This pre-start script therefore exists as a secondary safeguard: it enables
  # ip_forward and sets permissive FORWARD rules so guardian container networking
  # continues to work if anything else on the director resets the policy.
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
        # Ensure ip_forward is on (belt)
        sysctl -w net.ipv4.ip_forward=1 || echo 1 > /proc/sys/net/ipv4/ip_forward || true
        # Load connection-tracking module so state-based iptables rules work on Noble
        modprobe nf_conntrack 2>/dev/null || true
        # Set FORWARD policy to ACCEPT (belt)
        iptables -P FORWARD ACCEPT || true
        # Add explicit ACCEPT rules (suspenders). Use -C to check first so the
        # script is idempotent if run more than once (e.g., monit restart).
        # Rule 1: jumpbox subnet (10.0.0.0/24) → warden containers (10.244.0.0/16)
        iptables -C FORWARD -s 10.0.0.0/24 -d 10.244.0.0/16 -j ACCEPT 2>/dev/null || \
          iptables -I FORWARD 1 -s 10.0.0.0/24 -d 10.244.0.0/16 -j ACCEPT || true
        # Rule 2: established/related return packets from containers back to jumpbox
        iptables -C FORWARD -s 10.244.0.0/16 -d 10.0.0.0/24 \
          -m state --state ESTABLISHED,RELATED -j ACCEPT 2>/dev/null || \
          iptables -I FORWARD 2 -s 10.244.0.0/16 -d 10.0.0.0/24 \
          -m state --state ESTABLISHED,RELATED -j ACCEPT || true
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
  # firewall rule names. Our generated name (e.g. bbr-cli-2026-07-16t22-58z-a3f9)
  # is ~31 chars; increasing the truncation to 32 ensures the full name (including
  # the random suffix) is used, guaranteeing uniqueness between concurrent runs
  # while staying within GCP's 63-char resource name limit (32 + 31-char longest
  # suffix "-bosh-director-lite-tcp-routing" = 63).
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
