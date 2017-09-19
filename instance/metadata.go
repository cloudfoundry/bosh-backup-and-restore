package instance

import (
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

type LockBefore struct {
	JobName string `yaml:"job_name"`
	Release string `yaml:"release"`
}

type Metadata struct {
	BackupName           string       `yaml:"backup_name"`
	RestoreName          string       `yaml:"restore_name"`
	ShouldBeLockedBefore []LockBefore `yaml:"should_be_locked_before"`
}

func NewJobMetadata(data []byte) (*Metadata, error) {
	metadata := &Metadata{}
	err := yaml.Unmarshal(data, metadata)

	for _, lockBefore := range metadata.ShouldBeLockedBefore {
		if lockBefore.JobName == "" || lockBefore.Release == "" {
			return nil, errors.New(
				"both job name and release should be specified for should be locked before")
		}
	}

	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal job metadata")
	}

	return metadata, nil
}
