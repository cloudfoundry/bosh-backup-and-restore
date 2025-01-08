#!/usr/bin/env bash

set -euo pipefail
mkdir -p /var/vcap/jobs/s3-unversioned-blobstore-backup-restorer/config
cat << EOF > /var/vcap/jobs/s3-unversioned-blobstore-backup-restorer/config/buckets.json
{
  "buildpacks": {
    "aws_access_key_id": "${ACCESS_KEY}",
    "aws_secret_access_key": "${SECRET_KEY}",
    "aws_assumed_role_arn": "${ROLE_ARN}",
    "backup": {
      "name": "bbr-s3-validator-unversioned-bucket-backup",
      "region": "eu-west-1"
    },
    "endpoint": "https://s3.eu-west-1.amazonaws.com",
    "name": "bbr-s3-validator-unversioned-bucket-live",
    "region": "eu-west-1",
    "force_path_style": true
  },
  "droplets": {
    "aws_access_key_id": "${ACCESS_KEY}",
    "aws_secret_access_key": "${SECRET_KEY}",
    "aws_assumed_role_arn": "${ROLE_ARN}",
    "backup": {
      "name": "bbr-s3-validator-unversioned-bucket-backup",
      "region": "eu-west-1"
    },
    "endpoint": "https://s3.eu-west-1.amazonaws.com",
    "name": "bbr-s3-validator-unversioned-bucket-live",
    "region": "eu-west-1",
    "force_path_style": true
  },
  "packages": {
    "aws_access_key_id": "${ACCESS_KEY}",
    "aws_secret_access_key": "${SECRET_KEY}",
    "aws_assumed_role_arn": "${ROLE_ARN}",
    "backup": {
      "name": "bbr-s3-validator-unversioned-bucket-backup",
      "region": "eu-west-1"
    },
    "endpoint": "https://s3.eu-west-1.amazonaws.com",
    "name": "bbr-s3-validator-unversioned-bucket-live",
    "region": "eu-west-1",
    "force_path_style": true
  }
}
EOF

chmod +x ./s3-config-validator-dev-release/bbr-s3-config-validator
./s3-config-validator-dev-release/bbr-s3-config-validator --unversioned --validate-put-object \
  | sed 's/"\(aws_.*\)"\: "\(.*\)"/"\1": "<redacted>"/g'

