package ssh

import "golang.org/x/crypto/ssh"

//go:generate counterfeiter -o fakes/fake_remote_runner_factory.go . RemoteRunnerFactory
type RemoteRunnerFactory func(host, user, privateKey string, publicKeyCallback ssh.HostKeyCallback, publicKeyAlgorithm []string, logger Logger) (RemoteRunner, error)
