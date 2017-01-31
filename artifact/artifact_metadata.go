package artifact

type artifactMetadata struct {
	ArtifactName  string            `yaml:"artifact_name"`
	Checksum      map[string]string `yaml:"checksums"`
}

func (metadata artifactMetadata) Name() string {
	return metadata.ArtifactName
}
