package bosh

import (
	"strings"

	"strconv"

	"errors"
	"fmt"

	"github.com/cloudfoundry/bosh-cli/director"
	"github.com/cloudfoundry/bosh-utils/uuid"
	"github.com/pivotal-cf/pcf-backup-and-restore/instance"
	"github.com/pivotal-cf/pcf-backup-and-restore/orchestrator"
)

func New(boshDirector director.Director,
	sshOptsGenerator SSHOptsGenerator,
	connectionFactory SSHConnectionFactory,
	logger Logger) orchestrator.BoshDirector {
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

func (c client) FindInstances(deploymentName string) ([]orchestrator.Instance, error) {
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

	instances := []orchestrator.Instance{}
	instanceGroupWithSSHConnections := map[director.AllOrInstanceGroupOrInstanceSlug][]SSHConnection{}

	for _, instanceGroupName := range uniqueInstanceGroupNamesFromVMs(vms) {
		c.Logger.Debug("", "Setting up SSH for job %s", instanceGroupName)

		allVmInstances, err := director.NewAllOrInstanceGroupOrInstanceSlugFromString(instanceGroupName)
		if err != nil {
			cleanupAlreadyMadeConnections(deployment, instanceGroupWithSSHConnections, sshOpts)
			return nil, err
		}

		sshRes, err := deployment.SetUpSSH(allVmInstances, sshOpts)
		if err != nil {
			cleanupAlreadyMadeConnections(deployment, instanceGroupWithSSHConnections, sshOpts)
			return nil, err
		}
		instanceGroupWithSSHConnections[allVmInstances] = []SSHConnection{}

		for index, host := range sshRes.Hosts {
			var sshConnection SSHConnection
			var err error

			c.Logger.Debug("", "Attempting to SSH onto %s, %s", host.Host, host.IndexOrID)
			sshConnection, err = c.SSHConnectionFactory(defaultToSSHPort(host.Host), host.Username, privateKey)

			if err != nil {
				cleanupAlreadyMadeConnections(deployment, instanceGroupWithSSHConnections, sshOpts)
				return nil, err
			}
			instanceGroupWithSSHConnections[allVmInstances] = append(instanceGroupWithSSHConnections[allVmInstances], sshConnection)

			scripts, err := c.findScripts(host, sshConnection)

			if err != nil {
				cleanupAlreadyMadeConnections(deployment, instanceGroupWithSSHConnections, sshOpts)
				return nil, err
			}

			metadata, err := c.getMetadata(host, sshConnection)

			if err != nil {
				cleanupAlreadyMadeConnections(deployment, instanceGroupWithSSHConnections, sshOpts)
				return nil, err
			}

			jobs := instance.NewJobs(instance.NewBackupAndRestoreScripts(scripts), metadata)

			instances = append(instances,
				NewBoshInstance(
					instanceGroupName,
					strconv.Itoa(index),
					host.IndexOrID,
					sshConnection,
					deployment,
					c.Logger,
					jobs,
				),
			)
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

func (c client) getMetadata(host director.Host, sshConnection SSHConnection) (map[string]instance.Metadata, error) {
	c.Logger.Debug("", "Attempting to fetch metadata on %s/%s", host.Host, host.IndexOrID)

	metadata := map[string]instance.Metadata{}

	stdout, stderr, exitCode, err := sshConnection.Run("ls -1 /var/vcap/jobs/*/bin/p-metadata")

	if exitCode != 0 && !strings.Contains(string(stderr), "No such file or directory") {
		errorString := fmt.Sprintf(
			"Failed to check for job metadata scripts on %s/%s.\nStdout: %s\nStderr: %s",
			host.Host,
			host.IndexOrID,
			stdout,
			stderr,
		)
		return map[string]instance.Metadata{}, errors.New(errorString)
	}

	if err != nil {
		errorString := fmt.Sprintf(
			"An error occurred while checking for job metadata scripts on %s/%s: %s",
			host.Host,
			host.IndexOrID,
			err,
		)
		c.Logger.Error("", errorString)
		return map[string]instance.Metadata{}, errors.New(errorString)
	}

	files := strings.Split(string(stdout), "\n")

	for _, file := range files {
		jobName, _ := instance.Script(file).JobName()
		metadataContent, stderr, exitCode, err := sshConnection.Run(file)

		if exitCode != 0 && !strings.Contains(string(stderr), "No such file or directory") {
			errorString := fmt.Sprintf(
				"Failed to run job metadata scripts on %s/%s.\nStdout: %s\nStderr: %s",
				host.Host,
				host.IndexOrID,
				stdout,
				stderr,
			)
			return map[string]instance.Metadata{}, errors.New(errorString)
		}

		if err != nil {
			errorString := fmt.Sprintf(
				"An error occurred while running job metadata scripts on %s/%s: %s",
				host.Host,
				host.IndexOrID,
				err,
			)
			c.Logger.Error("", errorString)
			return map[string]instance.Metadata{}, errors.New(errorString)
		}

		jobMetadata, err := instance.NewJobMetadata(metadataContent)

		if err != nil {
			errorString := fmt.Sprintf(
				"Reading job metadata for %s/%s failed: %s",
				host.Host,
				host.IndexOrID,
				err.Error(),
			)
			c.Logger.Error("", errorString)
			return map[string]instance.Metadata{}, errors.New(errorString)
		}

		metadata[jobName] = *jobMetadata
	}

	return metadata, nil
}

func (c client) findScripts(host director.Host, sshConnection SSHConnection) ([]string, error) {
	c.Logger.Debug("", "Attempting to find scripts on %s/%s", host.Host, host.IndexOrID)

	stdout, stderr, exitCode, err := sshConnection.Run("find /var/vcap/jobs/*/bin/* -type f")
	if err != nil {
		c.Logger.Error(
			"",
			"Failed to run find on %s/%s. Error: %s\nStdout: %s\nStderr%s",
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
func cleanupAlreadyMadeConnections(deployment director.Deployment, instanceGroupWithSSHConnections map[director.AllOrInstanceGroupOrInstanceSlug][]SSHConnection, opts director.SSHOpts) {
	for slug, sshConnections := range instanceGroupWithSSHConnections {
		for _, connection := range sshConnections {
			connection.Cleanup()
		}
		deployment.CleanUpSSH(slug, director.SSHOpts{Username: opts.Username})
	}
}
