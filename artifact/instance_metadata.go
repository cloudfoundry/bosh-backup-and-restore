package artifact

type instanceMetadata struct {
	InstanceName  string            `yaml:"instance_name"`
	InstanceIndex string            `yaml:"instance_index"`
	Checksum      map[string]string `yaml:"checksums"`
}

func (m instanceMetadata) Name() string {
	return m.InstanceName
}

func (m instanceMetadata) Index() string {
	return m.InstanceIndex
}

func (m instanceMetadata) ID() string {
	return m.Index()
}
