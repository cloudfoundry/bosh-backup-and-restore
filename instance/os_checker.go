package instance

import "github.com/cloudfoundry-incubator/bosh-backup-and-restore/ssh"

//go:generate counterfeiter -o fakes/fake_os_checker.go . OSChecker
type OSChecker interface {
	IsLinux(instanceIdentifier InstanceIdentifier, remoteRunner ssh.RemoteRunner) (bool, error)
}
