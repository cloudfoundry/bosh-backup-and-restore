package instance

import (
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

type LockBefore struct {
	JobName string `yaml:"job_name"`
}

type Metadata struct {
	BackupName           string       `yaml:"backup_name"`
	RestoreName          string       `yaml:"restore_name"`
	ShouldBeLockedBefore []LockBefore `yaml:"should_be_locked_before"`
}

func NewJobMetadata(data []byte) (*Metadata, error) {
	metadata := &Metadata{}
	err := yaml.Unmarshal(data, metadata)

	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal job metadata")
	}

	return metadata, nil
}
