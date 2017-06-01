package artifact

import "gopkg.in/yaml.v2"
import "io/ioutil"

type backupActivityMetadata struct {
	StartTime  string `yaml:"start_time"`
	FinishTime string `yaml:"finish_time,omitempty"`
}

type instanceMetadata struct {
	Name      string             `yaml:"name"`
	Index     string             `yaml:"index"`
	Artifacts []artifactMetadata `yaml:"artifacts"`
}

type artifactMetadata struct {
	Name     string            `yaml:"name"`
	Checksum map[string]string `yaml:"checksums"`
}

type metadata struct {
	MetadataForEachInstance   []*instanceMetadata    `yaml:"instances,omitempty"`
	MetadataForEachBlob       []artifactMetadata     `yaml:"custom_artifacts,omitempty"`
	MetadataForBackupActivity backupActivityMetadata `yaml:"backup_activity"`
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

func (data *metadata) findOrCreateInstanceMetadata(name, index string) *instanceMetadata {
	for _, instanceMetadata := range data.MetadataForEachInstance {
		if instanceMetadata.Name == name && instanceMetadata.Index == index {
			return instanceMetadata
		}
	}
	newInstanceMetadata := &instanceMetadata{
		Name:  name,
		Index: index,
	}
	data.MetadataForEachInstance = append(data.MetadataForEachInstance, newInstanceMetadata)
	return newInstanceMetadata
}
