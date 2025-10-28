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

	jsonOutput, _ := json.MarshalIndent(config.Buckets, "  ", "  ")

	fmt.Fprintf(writer, "  %s\n\n", string(jsonOutput))
}
