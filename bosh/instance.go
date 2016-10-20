package bosh

import "github.com/cloudfoundry/bosh-cli/director"
import "github.com/pivotal-cf/pcf-backup-and-restore/backuper"

type DeployedInstance struct {
	director.Deployment
	JobName  string
	JobIndex string
	SSHConnection
}

//go:generate counterfeiter -o fakes/fake_ssh_connection.go . SSHConnection
type SSHConnection interface {
	Run(cmd string) ([]byte, []byte, int, error)
	Cleanup() error
	Username() string
}

func NewBoshInstance(jobName, jobIndex string, connection SSHConnection, deployment director.Deployment) backuper.Instance {
	return DeployedInstance{
		JobIndex:      jobIndex,
		JobName:       jobName,
		SSHConnection: connection,
		Deployment:    deployment,
	}
}

func (d DeployedInstance) IsBackupable() (bool, error) {
	_, _, exitCode, err := d.Run("ls /var/vcap/store/jobs/*/backup")
	return exitCode == 0, err
}

func (d DeployedInstance) Cleanup() error {
	return d.CleanUpSSH(director.NewAllOrPoolOrInstanceSlug(d.JobName, d.JobIndex), director.SSHOpts{Username: d.SSHConnection.Username()})
}
