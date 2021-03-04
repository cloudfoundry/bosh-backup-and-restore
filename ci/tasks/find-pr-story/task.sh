#!/bin/bash

# find-pr-story
# Script that retrieves a single story from tracker relating to a specific pull request
# Needs an input named "pr", generated with the Concourse PR Resource:
# https://github.com/telia-oss/github-pr-resource

set -euo pipefail

API_TOKEN=${TRACKER_API_TOKEN}
PROJECT_ID=${TRACKER_PROJECT_ID}
GIT_REPOSITORY=${GIT_REPOSITORY}
GIT_PR_ID=$(cat bosh-backup-and-restore/.git/resource/pr)

echo "Fetching tracker story for $GIT_REPOSITORY PR #$GIT_PR_ID..."

res=$(
    curl \
        -H "X-TrackerToken: $API_TOKEN" \
        "https://www.pivotaltracker.com/services/v5/projects/$PROJECT_ID/stories?with_label=github-pull-request&limit=9999" \
        -s \
        | jq -r '.[] | select(.description | contains("https://github.com/'$GIT_REPOSITORY'/pull/'$GIT_PR_ID'")) | .id'
    )

if [ -z "$res" ]
then
    echo "No story found"
    exit 1
fi

echo $res > ./tracker-story/id
echo $res