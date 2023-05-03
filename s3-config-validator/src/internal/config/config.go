package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"sort"
)

var errEmptyJSON = errors.New("invalid config: json was empty")

type Config struct {
	Buckets map[string]LiveBucket
}

type LiveBucket struct {
	Name          string        `json:"name"`
	Region        string        `json:"region"`
	ID            string        `json:"aws_access_key_id"`
	Secret        string        `json:"aws_secret_access_key"`
	Endpoint      string        `json:"endpoint"`
	Backup        *BackupBucket `json:"backup,omitempty"`
	UseIAMProfile bool          `json:"use_iam_profile"`
}

type BackupBucket struct {
	Name   string `json:"name"`
	Region string `json:"region"`
}

func Read(filePath string, versioned bool) (Config, error) {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return Config{}, err
	}

	return readConfig(data, versioned)
}

func readConfig(jsonFile []byte, versioned bool) (Config, error) {
	var buckets map[string]LiveBucket

	if err := json.Unmarshal(jsonFile, &buckets); err != nil {
		return Config{}, err
	}

	config := Config{Buckets: buckets}

	if err := validateConfig(config, versioned); err != nil {
		return Config{}, err
	}

	return config, nil
}

func validateConfig(config Config, versioned bool) error {
	if len(config.Buckets) == 0 {
		return errEmptyJSON
	}

	var emptyFieldNames []string
	var bucketsWithTooManyCreds []string
	var missingUnversionedBackupBuckets []string
	var bucketsWithEndpointThatUseIAM []string

	for liveBucketName, liveBucket := range config.Buckets {
		if liveBucket.Name == "" {
			emptyFieldNames = append(emptyFieldNames, liveBucketName+".name")
		}

		if liveBucket.Region == "" {
			emptyFieldNames = append(emptyFieldNames, liveBucketName+".region")
		}

		if liveBucket.UseIAMProfile {
			if liveBucket.ID != "" || liveBucket.Secret != "" {
				bucketsWithTooManyCreds = append(bucketsWithTooManyCreds, liveBucketName)
			}
			if liveBucket.Endpoint != "" {
				bucketsWithEndpointThatUseIAM = append(bucketsWithEndpointThatUseIAM, liveBucketName)
			}
		} else {
			if liveBucket.ID == "" {
				emptyFieldNames = append(emptyFieldNames, liveBucketName+".aws_access_key_id")
			}

			if liveBucket.Secret == "" {
				emptyFieldNames = append(emptyFieldNames, liveBucketName+".aws_secret_access_key")
			}
		}

		if !versioned {
			if liveBucket.Backup == nil {
				missingUnversionedBackupBuckets = append(missingUnversionedBackupBuckets, liveBucketName)
			} else {
				if liveBucket.Backup.Name == "" {
					emptyFieldNames = append(emptyFieldNames, liveBucketName+".backup.name")
				}

				if liveBucket.Backup.Region == "" {
					emptyFieldNames = append(emptyFieldNames, liveBucketName+".backup.region")
				}
			}
		}

	}

	errorMessage := ""
	if len(emptyFieldNames) > 0 {
		sort.Sort(sort.StringSlice(emptyFieldNames))
		errorMessage += fmt.Sprintf("invalid config: fields %v are empty\n", emptyFieldNames)
	}
	if len(bucketsWithTooManyCreds) > 0 {
		sort.Sort(sort.StringSlice(bucketsWithTooManyCreds))
		errorMessage += fmt.Sprintf("invalid config: because use_iam_profile is set to true, there should be no aws_access_key_id or aws_secret_access_key in the following buckets: %v\n", bucketsWithTooManyCreds)
	}
	if len(missingUnversionedBackupBuckets) > 0 {
		sort.Sort(sort.StringSlice(missingUnversionedBackupBuckets))
		errorMessage += fmt.Sprintf("invalid config: backup buckets must be specified when taking unversioned backups. The following buckets are missing backup buckets: %v\n", missingUnversionedBackupBuckets)
	}

	if len(bucketsWithEndpointThatUseIAM) > 0 {
		sort.Sort(sort.StringSlice(bucketsWithEndpointThatUseIAM))
		errorMessage += fmt.Sprintf("invalid config: because use_iam_profile is set to true, the endpoint field must not be set in the following buckets: %v\n", bucketsWithEndpointThatUseIAM)
	}

	if errorMessage != "" {
		return fmt.Errorf(errorMessage)
	}

	return nil
}
