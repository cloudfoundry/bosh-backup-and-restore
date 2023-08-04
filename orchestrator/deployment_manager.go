package orchestrator

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -generate
//counterfeiter:generate -o fakes/fake_deployment_manager.go . DeploymentManager
type DeploymentManager interface {
	Find(deploymentName string) (Deployment, error)
	SaveManifest(deploymentName string, artifact Backup) error
}
