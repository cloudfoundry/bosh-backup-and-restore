package orchestrator

type ArtifactExecutionStrategy interface {
	Run(backupArtifacts []BackupArtifact, action func(artifact BackupArtifact) error) []error
}