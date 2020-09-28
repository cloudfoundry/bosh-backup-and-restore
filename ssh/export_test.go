package ssh

func InjectBuildSSHSession(builder SSHSessionBuilder) {
	buildSSHSession = builder
}

func ResetBuildSSHSession() {
	buildSSHSession = buildSSHSessionImpl
}
