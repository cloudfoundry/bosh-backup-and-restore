package artifact

import "gopkg.in/yaml.v2"
import "io/ioutil"

type metadata struct {
	MetadataForEachInstance []instanceMetadata `yaml:"instances"`
	MetadataForEachBlob     []blobMetadata     `yaml:"blobs,omitempty"`
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
