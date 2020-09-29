# BBR S3 bucket configuration validator

The [BBR](https://docs.cloudfoundry.org/bbr/index.html) S3 bucket configuration
validator is a tool to validate and troubleshoot your TAS & BBR external
blobstore configuration. It will look at your bucket configuration file and
validate that all credentials, buckets and policies are in order. If it
succeeds, so should your backups and restores with BBR. If it does not, it will
help you debug the issue.

The tool is geared towards bucket configuration files that are produced by _Ops
Manager_ (see [Bucket configuration files](#bucket-configuration-files)).
Both
[versioned](https://docs.aws.amazon.com/AmazonS3/latest/dev/Versioning.html)
and unversioned S3-compatible blobstores are supported.

Get an idea by looking at the [sample output](#sample-output).

## Installation

**The tool should be used on the VM that you run your backups and restores from.**
This is to provide realistic network conditions and because that's where your
bucket configuration file will be put by _Ops Manager_.

You can use the [BOSH CLI](https://github.com/cloudfoundry/bosh-cli) to get
the tool onto that VM assuming that it's pointing to your environment:
 1. Find the deployment
    ```shell script
    bosh deployments
    ```
 1. Copy the binary onto the backup and restore VM
    ```shell script
    bosh --deployment <deployment> scp bbr-s3-config-validator-linux-amd64 backup_restore:/tmp
    ```
 1. Ssh onto the backup and restore VM
    ```shell script
    bosh --deployment <deployment> ssh backup_restore
    ```
 1. Move the binary into your homedir to be able to execute it
    ```shell script
    mv /tmp/bbr-s3-config-validator-linux-amd64 .
    ```

## Usage

By default the tool validates a versioned buckets configuration. It expects to
find this configuration at
`/var/vcap/jobs/s3-versioned-blobstore-backup-restorer/config/buckets.json`.
You can also use the tool to validate unversioned buckets by using the
`--unversioned` flag.

1. To learn more about the tool usage
   ```shell script
   ./bbr-s3-config-validator-linux-amd64 --help
   ```
1. To run it
   ```shell script
   ./bbr-s3-config-validator-linux-amd64
   ```

The tool will run a series of tests:
- Verify it can reach the blobstore and bucket
- Verify that the bucket is versioned or unversioned
- Verify it can get objects and objects metadata
- Verify it can write an object to the bucket (if you use the `--validate-put-object` flag)

You can override the default configuration location with the `BBR_S3_BUCKETS_CONFIG`
environment variable. This allows you to validate a configuration that you wish
to apply without overriding the current configuration.

## Bucket configuration files

A BBR bucket configuration file is expected to look like this:

```json
{
    "some-resource-to-backup": {
        "aws_access_key_id": "<the buckets' s3-compatible blobstore's access key>",
        "aws_secret_access_key": "<the buckets' s3-compatible blobstore's secret key>",
        "endpoint": "<the s3-compatible blobstore's endpoint>",
        "name": "<the live bucket's name>",
        "region": "<the live bucket's region>",
        "backup": {
            "name": "<the backup bucket's name>",
            "region": "<the backup bucket's region>"
        }
    },
    "another-resource-to-backup": {
        ...
    },
    ...
}
```

## Sample output
```shell script
$ ./bbr-s3-config-validator --validate-put-object

Make sure to run this on your 'backup & restore' VM.

Validating unversioned S3 buckets configuration at:

  /var/vcap/jobs/s3-unversioned-blobstore-backup-restorer/config/buckets.json

Configuration:

  {
    "packages": {
      "name": "packages-live",
      "region": "eu-west-1",
      "aws_access_key_id": "<redacted>",
      "aws_secret_access_key": "<redacted>",
      "endpoint": "https://s3.eu-west-1.amazonaws.com",
      "backup": {
        "name": "packages-backup",
        "region": "eu-west-1"
      }
    },
    "buildpacks": {
      "name": "buildpacks-live",
      "region": "eu-west-1",
      "aws_access_key_id": "<redacted>",
      "aws_secret_access_key": "<redacted>",
      "endpoint": "https://s3.eu-west-1.amazonaws.com",
      "backup": {
        "name": "buildpacks-backup",
        "region": "eu-west-1"
      }
    }
  }

Validating packages' live bucket packages-live ...
 * Bucket is not versioned ... Yes
 * Can list objects ... Yes
 * Can get objects ... Yes
 * Can put objects ... Yes

Validating packages' backup bucket packages-backup ...
 * Bucket is not versioned ... Yes
 * Can list objects ... Yes
 * Can get objects ... Yes
 * Can put objects ... Yes

Validating buildpacks' live bucket buildpacks-live ...
 * Bucket is not versioned ... Yes
 * Can list objects ... Yes
 * Can get objects ... Yes
 * Can put objects ... Yes

Validating buildpacks' backup bucket buildpacks-backup ...
 * Bucket is not versioned ... No [reason: bucket buildpacks-backup is versioned]
 * Can list objects ... Yes
 * Can get objects ... Yes
 * Can put objects ... Yes

Bad config
exit 1
```
