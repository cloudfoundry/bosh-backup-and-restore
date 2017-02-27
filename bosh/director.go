package bosh

import (
	"strings"

	"strconv"

	"fmt"

	"github.com/cloudfoundry/bosh-cli/director"
	"github.com/cloudfoundry/bosh-utils/uuid"
	"github.com/pivotal-cf/pcf-backup-and-restore/instance"
	"github.com/pivotal-cf/pcf-backup-and-restore/orchestrator"
)

func New(boshDirector director.Director,
	sshOptsGenerator SSHOptsGenerator,
	connectionFactory SSHConnectionFactory,
	logger Logger,
	jobFinder instance.JobFinder) orchestrator.BoshDirector {
	return client{
		Director:             boshDirector,
		SSHOptsGenerator:     sshOptsGenerator,
		SSHConnectionFactory: connectionFactory,
		Logger:               logger,
		jobFinder:            jobFinder,
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
	jobFinder instance.JobFinder
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

	c.Logger.Info("", "Scripts found:")

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

			hostIdentifier := fmt.Sprintf("%s/%s", instanceGroupName, host.IndexOrID)

			jobs, err := c.jobFinder.FindJobs(hostIdentifier, sshConnection)

			if err != nil {
				cleanupAlreadyMadeConnections(deployment, instanceGroupWithSSHConnections, sshOpts)
				return nil, err
			}

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

func cleanupAlreadyMadeConnections(deployment director.Deployment, instanceGroupWithSSHConnections map[director.AllOrInstanceGroupOrInstanceSlug][]SSHConnection, opts director.SSHOpts) {
	for slug, sshConnections := range instanceGroupWithSSHConnections {
		for _, connection := range sshConnections {
			connection.Cleanup()
		}
		deployment.CleanUpSSH(slug, director.SSHOpts{Username: opts.Username})
	}
}
