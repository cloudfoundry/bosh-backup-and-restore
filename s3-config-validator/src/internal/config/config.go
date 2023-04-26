package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
)

var errEmptyJSON = errors.New("invalid config: json was empty")

type Config struct {
	Buckets map[string]LiveBucket
}

type LiveBucket struct {
	Name     string       `json:"name"`
	Region   string       `json:"region"`
	ID       string       `json:"aws_access_key_id"`
	Secret   string       `json:"aws_secret_access_key"`
	Endpoint string       `json:"endpoint"`
	Backup   *BackupBucket `json:"backup,omitempty"`
	UseIAMProfile bool `json:"use_iam_profile"`
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
		} else {
			if liveBucket.ID == "" {
				emptyFieldNames = append(emptyFieldNames, liveBucketName+".aws_access_key_id")
			}

			if liveBucket.Secret == "" {
				emptyFieldNames = append(emptyFieldNames, liveBucketName+".aws_secret_access_key")
			}
		}

		if !versioned {
			if liveBucket.Backup.Name == "" {
				emptyFieldNames = append(emptyFieldNames, liveBucketName+".backup.name")
			}

			if liveBucket.Backup.Region == "" {
				emptyFieldNames = append(emptyFieldNames, liveBucketName+".backup.region")
			}
		}

	}

	if len(emptyFieldNames) > 0 {
		return fmt.Errorf("invalid config: fields %v are empty", emptyFieldNames)
	}
	if len(bucketsWithTooManyCreds) > 0 {
		explanation := ""
		for _, bucket := range bucketsWithTooManyCreds {
			explanation += fmt.Sprintf(" if %[1]s.use_iam_profile is true, then %[1]s.aws_access_key_id and %[1]s.aws_secret_access_key should be empty", bucket)
		}
		return fmt.Errorf("invalid config:%s", explanation)
	}

	return nil
}
