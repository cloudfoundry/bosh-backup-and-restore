package orchestrator

import (
	"github.com/cloudfoundry/bosh-backup-and-restore/executor"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -generate
//counterfeiter:generate -o fakes/fake_artifact_copier.go . ArtifactCopier
type ArtifactCopier interface {
	DownloadBackupFromDeployment(Backup, Deployment) error
	UploadBackupToDeployment(Backup, Deployment) error
}

type artifactCopier struct {
	Logger
	executor executor.Executor
}

func NewArtifactCopier(executor executor.Executor, logger Logger) ArtifactCopier {
	return artifactCopier{
		Logger:   logger,
		executor: executor,
	}
}

func (c artifactCopier) DownloadBackupFromDeployment(localBackup Backup, deployment Deployment) error {
	instances := deployment.BackupableInstances()

	var executables []executor.Executable
	for _, instance := range instances {
		for _, remoteBackupArtifact := range instance.ArtifactsToBackup() {
			executables = append(executables, NewBackupDownloadExecutable(localBackup, remoteBackupArtifact, c.Logger))
		}
	}

	errs := c.executor.Run([][]executor.Executable{executables})

	return ConvertErrors(errs)
}

func (c artifactCopier) UploadBackupToDeployment(localBackup Backup, deployment Deployment) error {
	instances := deployment.RestorableInstances()

	var executables []executor.Executable
	for _, instance := range instances {
		for _, remoteBackupArtifact := range instance.ArtifactsToRestore() {
			executables = append(executables, NewBackupUploadExecutable(localBackup, remoteBackupArtifact, instance, c.Logger))
		}
	}

	errs := c.executor.Run([][]executor.Executable{executables})

	return ConvertErrors(errs)
}
