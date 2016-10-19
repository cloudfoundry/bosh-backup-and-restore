package bosh

import "github.com/cloudfoundry/bosh-cli/director"
import "github.com/pivotal-cf/pcf-backup-and-restore/backuper"

type DeployedInstance struct {
	director.Director
	JobName  string
	JobIndex string
	SSHConnection
}

//go:generate counterfeiter -o fakes/fake_ssh_connection.go . SSHConnection
type SSHConnection interface {
	Run(cmd string) ([]byte, []byte, error)
	Cleanup() error
}

func NewBoshInstance(jobName, jobIndex string, connection SSHConnection) backuper.Instance {
	return DeployedInstance{
		JobIndex:      jobIndex,
		JobName:       jobName,
		SSHConnection: connection,
	}
}

func (DeployedInstance) IsBackupable() (bool, error) {
	return false, nil
}
func (DeployedInstance) Cleanup() error {
	return nil
}
