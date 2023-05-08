#!/bin/bash

set -euo pipefail
gcloud -q auth activate-service-account --key-file=<(echo "$GCP_SERVICE_ACCOUNT_KEY")
env_name="$(cat additional-environment/name)"
echo 'Y' | gcloud compute firewall-rules delete "${env_name}-custom-jumpbox-to-director-ingress-allow" || true
gcloud compute firewall-rules create "${env_name}-custom-jumpbox-to-director-ingress-allow" \
       --network="${env_name}-network"     \
       --direction=ingress \
       --target-tags="${env_name}-bosh-director" \
       --action=allow \
       --rules=tcp:22,tcp:6868,tcp:8443,tcp:8844,tcp:25555 \
       --source-tags=jumpbox \
       --priority=999