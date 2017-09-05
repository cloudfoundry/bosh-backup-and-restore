package instance

//go:generate counterfeiter -o fakes/fake_release_mapping_finder.go . ReleaseMappingFinder
type ReleaseMappingFinder func(manifest string, instanceNames []string) ReleaseMapping

//go:generate counterfeiter -o fakes/fake_release_mapping.go . ReleaseMapping
type ReleaseMapping interface {
	FindReleaseName(instanceGroupName, jobName string) (string, error)
}

type noopReleaseMapping struct{}

func NoopReleaseMapping() ReleaseMapping {
	return noopReleaseMapping{}
}

func (rm noopReleaseMapping) FindReleaseName(instanceGroupName, jobName string) (string, error) {
	return "", nil
}
