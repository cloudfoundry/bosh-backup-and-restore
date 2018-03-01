package orchestrator

type ArtifactCopy struct {
	backupArtifact BackupArtifact
	checksum       BackupChecksum
	err            error
}
