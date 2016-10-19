package backuper

import "fmt"

func New(bosh BoshDirector) *Backuper {
	return &Backuper{
		BoshDirector: bosh,
	}
}

type Backuper struct {
	BoshDirector
}

func (b Backuper) Backup(deploymentName string) error {
	instances, err := b.FindInstances(deploymentName)
	if err != nil {
		return err
	}
	defer instances.Cleanup()

	if backupable, err := instances.AreAnyBackupable(); err != nil {
		return err
	} else if !backupable {
		return fmt.Errorf("Deployment '%s' has no backup scripts", deploymentName)
	}
	return nil
}

//go:generate counterfeiter -o fakes/fake_bosh_director.go . BoshDirector
type BoshDirector interface {
	FindInstances(deploymentName string) (Instances, error)
}
