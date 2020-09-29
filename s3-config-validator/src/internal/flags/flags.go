package flags

import (
	"flag"
	"fmt"
	"os"
)

const HelpMessage = `
Validates a BOSH backup and restore bucket configuration.
By default it will assume versioned buckets unless specified otherwise.

The default config file locations are:

 * versioned: /var/vcap/jobs/s3-versioned-blobstore-backup-restorer/config/buckets.json
 * unversioned: /var/vcap/jobs/s3-unversioned-blobstore-backup-restorer/config/buckets.json

Make sure to run this on the ‘backup_restore’ VM.

USAGE:
  bbr-s3-config-validator [--validate-put-object]

OPTIONS:
  --help                        Show usage.
  --unversioned                 Validate unversioned bucket configuration.
  --validate-put-object         Test writing objects to the buckets. Disclaimer: This will write test files to the buckets.

ENVIRONMENT VARIABLES:
  BBR_S3_BUCKETS_CONFIG=<path>  Override the default bucket configuration file location
`

const RunLocationHint = `Make sure to run this on your 'backup & restore' VM.`

const ReadOnlyValidationHint = `Run with --validate-put-object to test writing objects to the buckets. Disclaimer: This will write test files to the buckets.`

func OverrideDefaultHelpFlag(message string) {
	flag.Usage = func() {
		fmt.Fprintf(os.Stdout, "%v", message)
		os.Exit(0)
	}
}
