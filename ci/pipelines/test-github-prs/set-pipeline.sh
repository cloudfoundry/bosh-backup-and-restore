#!/usr/bin/env bash

set -euo pipefail

GITHUB_ACCESS_TOKEN=$(lpass show "Shared-Cryogenics/github-ci-access-token" --field="password")
GITHUB_CI_SSH_KEY=$(lpass show "Shared-Cryogenics/github-ci-ssh-key" --field="Private Key")
DOCKER_HOST_SSH_KEY=$(lpass show "Shared-Cryogenics/docker-host-ssh-key" --notes)
TRACKER_API_TOKEN=$(lpass show "Shared-Cryogenics/tracker-api-token" --notes)
TRACKER_PROJECT_ID=2475702

fly --target cryo-bbr set-pipeline \
  --pipeline test-github-prs \
  --config pipeline.yml \
  --var github-access-token="${GITHUB_ACCESS_TOKEN}" \
  --var github-ci-ssh-key="${GITHUB_CI_SSH_KEY}" \
  --var docker-host-ssh-key="${DOCKER_HOST_SSH_KEY}" \
  --var tracker-api-token="${TRACKER_API_TOKEN}" \
  --var tracker-project-id="${TRACKER_PROJECT_ID}"


