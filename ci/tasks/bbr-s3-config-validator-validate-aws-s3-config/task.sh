#!/usr/bin/env bash

set -euo pipefail
mkdir -p /var/vcap/jobs/s3-unversioned-blobstore-backup-restorer/config
cat << EOF > /var/vcap/jobs/s3-unversioned-blobstore-backup-restorer/config/buckets.json
{
  "buildpacks": {
    "aws_access_key_id": "${AWS_ACCESS_KEY}",
    "aws_secret_access_key": "${AWS_SECRET_KEY}",
    "aws_assumed_role_arn": "${AWS_ASSUMED_ROLE_ARN}",
    "backup": {
      "name": "${AWS_UNVERSIONED_BUCKET_BACKUP}",
      "region": "${AWS_REGION}"
    },
    "endpoint": "https://s3.${AWS_REGION}.amazonaws.com",
    "name": "${AWS_UNVERSIONED_BUCKET_LIVE}",
    "region": "${AWS_REGION}",
    "force_path_style": true
  },
  "droplets": {
    "aws_access_key_id": "${AWS_ACCESS_KEY}",
    "aws_secret_access_key": "${AWS_SECRET_KEY}",
    "aws_assumed_role_arn": "${AWS_ASSUMED_ROLE_ARN}",
    "backup": {
      "name": "${AWS_UNVERSIONED_BUCKET_BACKUP}",
      "region": "${AWS_REGION}"
    },
    "endpoint": "https://s3.${AWS_REGION}.amazonaws.com",
    "name": "${AWS_UNVERSIONED_BUCKET_LIVE}",
    "region": "${AWS_REGION}",
    "force_path_style": true
  },
  "packages": {
    "aws_access_key_id": "${AWS_ACCESS_KEY}",
    "aws_secret_access_key": "${AWS_SECRET_KEY}",
    "aws_assumed_role_arn": "${AWS_ASSUMED_ROLE_ARN}",
    "backup": {
      "name": "${AWS_UNVERSIONED_BUCKET_BACKUP}",
      "region": "${AWS_REGION}"
    },
    "endpoint": "https://s3.${AWS_REGION}.amazonaws.com",
    "name": "${AWS_UNVERSIONED_BUCKET_LIVE}",
    "region": "${AWS_REGION}",
    "force_path_style": true
  }
}
EOF

chmod +x ./s3-config-validator-dev-release/bbr-s3-config-validator
./s3-config-validator-dev-release/bbr-s3-config-validator --unversioned --validate-put-object \
  | sed 's/"\(aws_.*\)"\: "\(.*\)"/"\1": "<redacted>"/g'

