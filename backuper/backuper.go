package backuper

import "fmt"

func New(boshClient BoshClient) Backuper {
	return Backuper{
		BoshClient: boshClient,
	}
}

type Backuper struct {
	BoshClient
}

func (b Backuper) Backup(deploymentName string) error {
	exists, err := b.CheckDeploymentExists(deploymentName)
	if err != nil {
		return err
	}
	if exists == false {
		return fmt.Errorf("Deployment '%s' not found", deploymentName)
	}

	return nil
}

//go:generate counterfeiter -o fakes/fake_bosh_client.go . BoshClient
type BoshClient interface {
	CheckDeploymentExists(name string) (bool, error)
}
