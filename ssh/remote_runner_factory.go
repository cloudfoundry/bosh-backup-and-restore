package ssh

import "golang.org/x/crypto/ssh"

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -generate
//counterfeiter:generate -o fakes/fake_remote_runner_factory.go . RemoteRunnerFactory
type RemoteRunnerFactory func(host, user, privateKey string, publicKeyCallback ssh.HostKeyCallback, publicKeyAlgorithm []string, logger Logger) (RemoteRunner, error)
