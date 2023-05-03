package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/s3-config-validator/src/internal/config"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/s3-config-validator/src/internal/configPrinter"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/s3-config-validator/src/internal/flags"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/s3-config-validator/src/internal/runner"
)

const (
	ConfigPathEnv     = "BBR_S3_BUCKETS_CONFIG"
	UnversionedConfig = "/var/vcap/jobs/s3-unversioned-blobstore-backup-restorer/config/buckets.json"
	VersionedConfig   = "/var/vcap/jobs/s3-versioned-blobstore-backup-restorer/config/buckets.json"
)

type CommandParams struct {
	ReadOnlyValidation bool
	Versioned          bool
	ConfigPath         string
}

func main() {
	commandParams := parseParams()

	validatedConfig, err := config.Read(commandParams.ConfigPath, commandParams.Versioned)

	printHeader(commandParams, commandParams.Versioned)

	configPrinter.PrintConfig(os.Stdout, validatedConfig)

	if err != nil {
		fmt.Printf("%v\n", err.Error())
		fmt.Println("Bad config")
		printHints(commandParams)
		os.Exit(1)
	}

	if !isValid(validatedConfig, commandParams.ReadOnlyValidation, commandParams.Versioned) {
		fmt.Println("Bad config")
		printHints(commandParams)
		os.Exit(1)
	}

	fmt.Println("Good config")
	printHints(commandParams)
}

func printHeader(commandParams CommandParams, versioned bool) {
	fmt.Printf("\n%s\n\n", flags.RunLocationHint)

	if versioned {
		fmt.Printf("Validating versioned S3 buckets configuration at:\n\n  %s\n\n", commandParams.ConfigPath)
	} else {
		fmt.Printf("Validating unversioned S3 buckets configuration at:\n\n  %s\n\n", commandParams.ConfigPath)
	}

}

func printHints(commandParams CommandParams) {
	if commandParams.ReadOnlyValidation {
		fmt.Printf("\n%s\n\n", flags.ReadOnlyValidationHint)
	}
}

func parseParams() CommandParams {
	var (
		validatePutObject bool
		unversioned       bool
	)
	flags.OverrideDefaultHelpFlag(flags.HelpMessage)
	flag.BoolVar(&validatePutObject, "validate-put-object", false, "Test writing objects to the buckets. Disclaimer: This will write test files to the buckets!")
	flag.BoolVar(&unversioned, "unversioned", false, "Validate unversioned bucket configuration.")
	flag.Parse()

	return CommandParams{
		ReadOnlyValidation: !validatePutObject,
		Versioned:          !unversioned,
		ConfigPath:         getConfigPath(!unversioned),
	}
}

func getConfigPath(versioned bool) string {
	configPath := os.Getenv(ConfigPathEnv)
	if configPath == "" {
		if versioned {
			configPath = VersionedConfig
		} else {
			configPath = UnversionedConfig
		}
	}
	return configPath
}

func isValid(config config.Config, readOnly, versioned bool) (isValidConfig bool) {
	isValidConfig = true

	var probeRunners []runner.ProbeRunner

	for resource, bucket := range config.Buckets {
		probeRunners = append(probeRunners, runner.NewProbeRunners(resource, bucket, readOnly, versioned)...)
	}

	for _, probeRunner := range probeRunners {
		if !probeRunner.Run() {
			isValidConfig = false
		}
	}

	return isValidConfig
}
