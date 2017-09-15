package orchestrator

type Session struct {
	deploymentName      string
	deployment          Deployment
	currentArtifact     Backup
	currentArtifactPath string
}

func NewSession(deploymentName string) *Session {
	return &Session{deploymentName: deploymentName}
}

func (session *Session) SetCurrentArtifact(artifact Backup) {
	session.currentArtifact = artifact
}

func (session *Session) DeploymentName() string {
	return session.deploymentName
}

func (session *Session) CurrentDeployment() Deployment {
	return session.deployment
}

func (session *Session) SetCurrentDeployment(deployment Deployment) {
	session.deployment = deployment
}

func (session *Session) CurrentArtifact() Backup {
	return session.currentArtifact
}

func (session *Session) SetCurrentArtifactPath(artifactPath string) {
	session.currentArtifactPath = artifactPath
}

func (session *Session) CurrentArtifactPath() string {
	return session.currentArtifactPath
}
