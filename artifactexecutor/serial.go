package artifactexecutor

import "github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator"

type SerialExecutionStrategy struct{}

func NewSerialExecutionStrategy() SerialExecutionStrategy {
	return SerialExecutionStrategy{}
}

func (s SerialExecutionStrategy) Run(backupArtifacts []orchestrator.BackupArtifact, action func(artifact orchestrator.BackupArtifact) error) []error {
	var outputErrs []error

	for _, backupArtifact := range backupArtifacts {
		err := action(backupArtifact)
		outputErrs = append(outputErrs, err)
	}

	return outputErrs
}
