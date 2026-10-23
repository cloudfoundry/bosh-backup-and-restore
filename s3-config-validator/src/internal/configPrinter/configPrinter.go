package configPrinter

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/cloudfoundry/bosh-backup-and-restore/s3-config-validator/src/internal/config"
)

func PrintConfig(writer io.Writer, config config.Config) {
	fmt.Fprintf(writer, "Configuration:\n\n")

	if config.Buckets == nil {
		fmt.Fprintf(writer, "  {}\n\n")
	}

	var displayBuckets map[string]interface{}
	if config.Buckets != nil {
		displayBuckets = make(map[string]interface{}, len(config.Buckets))
	}
	for name, bucket := range config.Buckets {
		bucket.ID = redact(bucket.ID)
		bucket.Secret = redact(bucket.Secret)
		displayBuckets[name] = bucket
	}
	jsonOutput, _ := json.MarshalIndent(displayBuckets, "  ", "  ")

	fmt.Fprintf(writer, "  %s\n\n", string(jsonOutput))
}

func redact(value string) string {
	if value == "" {
		return ""
	}
	return "REDACTED"
}
