package instance

//go:generate counterfeiter -o fakes/fake_manifest_querier_creator.go . ManifestQuerierCreator
type ManifestQuerierCreator func(manifest string) (ManifestQuerier, error)

//go:generate counterfeiter -o fakes/fake_manifest_querier.go . ManifestQuerier
type ManifestQuerier interface {
	FindReleaseName(instanceGroupName, jobName string) (string, error)
	IsJobBackupOneRestoreAll(instanceGroupName, jobName string) (bool, error)
}

type noopManifestQuerier struct{}

func NewNoopManifestQuerier() ManifestQuerier {
	return noopManifestQuerier{}
}

func (mq noopManifestQuerier) FindReleaseName(instanceGroupName, jobName string) (string, error) {
	return "", nil
}

func (mq noopManifestQuerier) IsJobBackupOneRestoreAll(instanceGroupName, jobName string) (bool, error) {
	return false, nil
}
