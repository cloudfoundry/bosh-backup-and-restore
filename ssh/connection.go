package ssh

func ConnectionCreator(hostName, username, privateKey string) (Connection, error) {
	return Connection{}, nil
}

type Connection struct {
	host string
	user string
}

func (c Connection) Run(cmd string) ([]byte, []byte, int, error) {
	// parsedPrivateKey, err := ssh.ParsePrivateKey([]byte(privateKey))
	// if err != nil {
	// 	return nil,err
	// }
	//
	// sshConfig := &ssh.ClientConfig{
	// 	User: host.Username,
	// 	Auth: []ssh.AuthMethod{
	// 		ssh.PublicKeys(parsedPrivateKey),
	// 	},
	// }
	//
	// connection, err := ssh.Dial("tcp", host.Host+":22", sshConfig)
	// if err != nil {
	// 	return nil,err
	// }
	//
	// session, err := connection.NewSession()
	// if err != nil {
	// 	return nil,err
	// }
	//
	// output, err := session.Output("ls /tmp")
	// if err != nil {
	// 	return nil,err
	//
	// }
	//
	// session.Wait()

	return nil, nil, 0, nil
}

func (c Connection) Close() error {
	return nil
}
