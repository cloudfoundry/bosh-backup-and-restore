package artifact

import "gopkg.in/yaml.v2"
import "io/ioutil"

type instanceMetadata struct {
	InstanceName string            `yaml:"instance_name"`
	InstanceID   string            `yaml:"instance_id"`
	Checksum     map[string]string `yaml:"checksums"`
}

func (m instanceMetadata) Name() string {
	return m.InstanceName
}

func (m instanceMetadata) ID() string {
	return m.InstanceID
}

type metadata struct {
	MetadataForEachInstance []instanceMetadata `yaml:"instances"`
}

func readMetadata(filename string) (metadata, error) {
	metadata := metadata{}

	contents, err := ioutil.ReadFile(filename)
	if err != nil {
		return metadata, err
	}

	if err := yaml.Unmarshal(contents, &metadata); err != nil {
		return metadata, err
	}
	return metadata, nil
}

func (data *metadata) save(filename string) error {
	contents, err := yaml.Marshal(data)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(filename, contents, 0666)
}
