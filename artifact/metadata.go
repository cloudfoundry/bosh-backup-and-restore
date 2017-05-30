package artifact

import "gopkg.in/yaml.v2"
import "io/ioutil"

type backupActivityMetadata struct {
	StartTime  string `yaml:"start_time"`
	FinishTime string `yaml:"finish_time,omitempty"`
}

type metadata struct {
	MetadataForEachInstance   []instanceMetadata     `yaml:"instances,omitempty"`
	MetadataForEachBlob       []blobMetadata         `yaml:"blobs,omitempty"`
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
