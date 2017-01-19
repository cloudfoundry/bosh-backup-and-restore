package bosh

import (
	"strings"

	"strconv"

	"github.com/cloudfoundry/bosh-cli/director"
	"github.com/cloudfoundry/bosh-utils/uuid"
	"github.com/pivotal-cf/pcf-backup-and-restore/backuper"
	"fmt"
)

func New(boshDirector director.Director,
	sshOptsGenerator SSHOptsGenerator,
	connectionFactory SSHConnectionFactory,
	logger Logger) backuper.BoshDirector {
	return client{
		Director:             boshDirector,
		SSHOptsGenerator:     sshOptsGenerator,
		SSHConnectionFactory: connectionFactory,
		Logger:               logger,
	}
}

//go:generate counterfeiter -o fakes/fake_opts_generator.go . SSHOptsGenerator
type SSHOptsGenerator func(uuidGen uuid.Generator) (director.SSHOpts, string, error)

//go:generate counterfeiter -o fakes/fake_ssh_connection_factory.go . SSHConnectionFactory
type SSHConnectionFactory func(host, user, privateKey string) (SSHConnection, error)

type client struct {
	director.Director
	SSHOptsGenerator
	SSHConnectionFactory
	Logger
}

type Logger interface {
	Debug(tag, msg string, args ...interface{})
	Info(tag, msg string, args ...interface{})
	Error(tag, msg string, args ...interface{})
}

func (c client) FindInstances(deploymentName string) ([]backuper.Instance, error) {
	deployment, err := c.Director.FindDeployment(deploymentName)
	if err != nil {
		return nil, err
	}

	c.Logger.Debug("", "Finding VMs...")
	vms, err := deployment.VMInfos()
	if err != nil {
		return nil, err
	}
	sshOpts, privateKey, err := c.SSHOptsGenerator(uuid.NewGenerator())
	if err != nil {
		return nil, err
	}
	c.Logger.Debug("", "SSH user generated: %s", sshOpts.Username)

	instances := []backuper.Instance{}

	for _, instanceGroupName := range uniqueInstanceGroupNamesFromVMs(vms) {
		c.Logger.Debug("", "Setting up SSH for job %s", instanceGroupName)

		allVmInstances, err := director.NewAllOrInstanceGroupOrInstanceSlugFromString(instanceGroupName)
		if err != nil {
			return nil, err
		}

		sshRes, err := deployment.SetUpSSH(allVmInstances, sshOpts)
		if err != nil {
			return nil, err
		}

		for index, host := range sshRes.Hosts {
			var sshConnection SSHConnection
			var err error

			c.Logger.Debug("", "Attempting to SSH onto %s, %s", host.Host, host.IndexOrID)
			sshConnection, err = c.SSHConnectionFactory(defaultToSSHPort(host.Host), host.Username, privateKey)

			if err != nil {
				return nil, err
			}

			scripts, err := c.findScripts(host, sshConnection)

			if err != nil {
				return nil, err
			}

			instances = append(instances, NewBoshInstance(instanceGroupName, strconv.Itoa(index), host.IndexOrID, sshConnection, deployment, c.Logger, NewBackupAndRestoreScripts(scripts)))
		}
	}

	return instances, nil
}
func (c client) GetManifest(deploymentName string) (string, error) {
	deployment, err := c.Director.FindDeployment(deploymentName)
	if err != nil {
		return "", err
	}
	return deployment.Manifest()
}

func defaultToSSHPort(host string) string {
	parts := strings.Split(host, ":")
	if len(parts) == 2 {
		return host
	} else {
		return host + ":22"
	}
}

func uniqueInstanceGroupNamesFromVMs(vms []director.VMInfo) []string {
	var jobs []string
	for _, vm := range vms {
		if !contains(jobs, vm.JobName) {
			jobs = append(jobs, vm.JobName)
		}
	}
	return jobs
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func (c client) findScripts(host director.Host, sshConnection SSHConnection) ([]string, error) {
	c.Logger.Debug("", "Attempting to find scripts on %s/%s", host.Host, host.IndexOrID)

	stdout, stderr, exitCode, err := sshConnection.Run("find /var/vcap/jobs/*/bin/* -type f")
	if err != nil {
		c.Logger.Error(
			"",
			"Failed to run find on %s/%s. Error: %s'nStdout: %s\nStderr%s",
			host.Host,
			host.IndexOrID,
			err,
			stdout,
			stderr,
		)
		return nil, err
	}

	if exitCode != 0 {
		if strings.Contains(string(stderr), "No such file or directory") {
			c.Logger.Debug(
				"",
				"Running find failed on %s/%s.\nStdout: %s\nStderr: %s",
				host.Host,
				host.IndexOrID,
				stdout,
				stderr,
			)
		} else {
			c.Logger.Error(
				"",
				"Running find failed on %s/%s.\nStdout: %s\nStderr: %s",
				host.Host,
				host.IndexOrID,
				stdout,
				stderr,
			)
			return nil, fmt.Errorf(
				"Running find failed on %s/%s.\nStdout: %s\nStderr: %s",
				host.Host,
				host.IndexOrID,
				stdout,
				stderr,
			)
		}
	}
	return strings.Split(string(stdout), "\n"), nil
}
