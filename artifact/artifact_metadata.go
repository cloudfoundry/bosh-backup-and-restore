package artifact

type artifactMetadata struct {
	ArtifactName string            `yaml:"artifact_name"`
	Checksum     map[string]string `yaml:"checksums"`
}

func (metadata artifactMetadata) Name() string {
	return metadata.ArtifactName
}

func (metadata artifactMetadata) Index() string {
	return ""
}

func (metadata artifactMetadata) ID() string {
	return ""
}

func (metadata artifactMetadata) IsNamed() bool {
	return true
}
