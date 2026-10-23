package configPrinter_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/cloudfoundry/bosh-backup-and-restore/s3-config-validator/src/internal/config"
	"github.com/cloudfoundry/bosh-backup-and-restore/s3-config-validator/src/internal/configPrinter"
)

func TestDisplayedConfigDoesNotChangeCredentials(t *testing.T) {
	cfg := config.Config{Buckets: map[string]config.LiveBucket{"bucket": {Name: "bucket-name", ID: "private-id", Secret: "private-secret"}}}
	var out bytes.Buffer
	configPrinter.PrintConfig(&out, cfg)
	if strings.Contains(out.String(), "private-id") || strings.Contains(out.String(), "private-secret") {
		t.Fatal("display includes credentials")
	}
	if !strings.Contains(out.String(), "bucket-name") {
		t.Fatal("display lost bucket metadata")
	}
	if cfg.Buckets["bucket"].ID != "private-id" || cfg.Buckets["bucket"].Secret != "private-secret" {
		t.Fatal("printing changed credentials")
	}
}

func TestDisplayedConfigKeepsEmptyCredentialsEmpty(t *testing.T) {
	cfg := config.Config{Buckets: map[string]config.LiveBucket{"bucket": {Name: "bucket-name", UseIAMProfile: true}}}
	var out bytes.Buffer
	configPrinter.PrintConfig(&out, cfg)
	if strings.Contains(out.String(), "REDACTED") {
		t.Fatal("display marks unset credentials as redacted")
	}
	if !strings.Contains(out.String(), `"aws_access_key_id": ""`) || !strings.Contains(out.String(), `"aws_secret_access_key": ""`) {
		t.Fatal("display lost empty credential fields")
	}
}
