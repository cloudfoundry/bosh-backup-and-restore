package ssh

//go:generate counterfeiter -o fakes/fake_ssh_connection_factory.go . SSHConnectionFactory
type SSHConnectionFactory func(host, user, privateKey string) (SSHConnection, error)
