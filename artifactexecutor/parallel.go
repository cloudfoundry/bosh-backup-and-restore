package artifactexecutor

import "github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator"

type ParallelExecutionStrategy struct{}

func NewParallelExecutionStrategy() ParallelExecutionStrategy {
	return ParallelExecutionStrategy{}
}

func (s ParallelExecutionStrategy) Run(backupArtifacts []orchestrator.BackupArtifact, action func(artifact orchestrator.BackupArtifact) error) []error {
	errs := make(chan error, len(backupArtifacts))

	for _, backupArtifact := range backupArtifacts {
		go func(ba orchestrator.BackupArtifact) {
			errs <- action(ba)
		}(backupArtifact)
	}

	var outputErrs []error
	for i := 0; i < len(backupArtifacts); i++ {
		err := <-errs
		if err != nil {
			outputErrs = append(outputErrs, err)
		}
	}

	return outputErrs
}
