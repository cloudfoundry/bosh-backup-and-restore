package artifact

type blobMetadata struct {
	BlobName string            `yaml:"blob_name"`
	Checksum map[string]string `yaml:"checksums"`
}

func (metadata blobMetadata) Name() string {
	return metadata.BlobName
}

func (metadata blobMetadata) Index() string {
	return ""
}

func (metadata blobMetadata) ID() string {
	return ""
}

func (metadata blobMetadata) IsNamed() bool {
	return true
}
