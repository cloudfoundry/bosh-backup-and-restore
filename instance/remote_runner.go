package instance

import (
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/ssh"
	"fmt"
	"errors"
)

type RemoteRunner struct {
	instanceIdentifier InstanceIdentifier
	logger             Logger
	connection         ssh.SSHConnection
}

func NewRemoteRunner(connection ssh.SSHConnection, instanceId InstanceIdentifier, logger Logger) RemoteRunner {
	return RemoteRunner{
		connection:         connection,
		instanceIdentifier: instanceId,
		logger:             logger,
	}
}

func (r RemoteRunner) directoryExists(dir string) (bool, error) {
	_, _, exitCode, err := r.runOnInstance(
		fmt.Sprintf("stat %s", dir),
		fmt.Sprintf("checking directory '%s' exists", dir),
	)

	return exitCode == 0, err
}

func (r RemoteRunner) removeDirectory(dir string) error {
	_, stdErr, exitCode, err := r.runOnInstance(fmt.Sprintf("sudo rm -rf %s", dir), "remove artifact directory")

	if err != nil {
		return err
	}

	if exitCode != 0 {
		return errors.New(string(stdErr))
	}

	return err
}

func (r RemoteRunner) runOnInstance(cmd, label string) ([]byte, []byte, int, error) {
	r.logger.Debug("bbr", "Running %s on %s", label, r.instanceIdentifier)

	stdout, stderr, exitCode, err := r.connection.Run(cmd)
	r.logger.Debug("bbr", "Stdout: %s", string(stdout))
	r.logger.Debug("bbr", "Stderr: %s", string(stderr))

	if err != nil {
		r.logger.Debug("bbr", "Error running %s on instance %s. Exit code %d, error: %s", label, r.instanceIdentifier, exitCode, err.Error())
	}

	return stdout, stderr, exitCode, err
}
