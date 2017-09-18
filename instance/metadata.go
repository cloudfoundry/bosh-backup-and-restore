package instance

import (
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

type Metadata struct {
	Backup  ActionConfig `yaml:"backup"`
	Restore ActionConfig `yaml:"restore"`
}

type ActionConfig struct {
	Name                 string       `yaml:"name"`
	ShouldBeLockedBefore []LockBefore `yaml:"should_be_locked_before"`
}

type LockBefore struct {
	JobName string `yaml:"job_name"`
	Release string `yaml:"release"`
}

func NewJobMetadata(data []byte) (*Metadata, error) {
	metadata := &Metadata{}
	err := yaml.Unmarshal(data, metadata)

	for _, lockBefore := range metadata.Backup.ShouldBeLockedBefore {
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
