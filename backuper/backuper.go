package backuper

import "github.com/cloudfoundry/bosh-cli/director"

func New(boshDirector director.Director) Backuper {
	return Backuper{
		Director: boshDirector,
	}
}

type Backuper struct {
	Director director.Director
}

func (b Backuper) Backup(deploymentName string) error {
	deployment, err := b.Director.FindDeployment(deploymentName)
	if err != nil {
		return err
	}
	_, err = deployment.Manifest()
	if err != nil {
		return err
	}

	return nil
}

//go:generate counterfeiter -o fakes/fake_bosh_client.go . BoshClient
type BoshClient interface {
	CheckDeploymentExists(name string) (bool, error)
}
