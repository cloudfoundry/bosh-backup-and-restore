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

//Backup checks if a deployment has backupable instances and backs them up.
func (b Backuper) Backup(deploymentName string) error {
	instances, err := b.FindInstances(deploymentName)
	if err != nil {
		return err
	}
	defer instances.Cleanup()

	backupable, err := instances.AllBackupable()
	if err != nil {
		return err
	}
	if len(backupable) == 0 {
		return fmt.Errorf("Deployment '%s' has no backup scripts", deploymentName)
	}

	return backupable.Backup()
}

//go:generate counterfeiter -o fakes/fake_bosh_director.go . BoshDirector
type BoshDirector interface {
	FindInstances(deploymentName string) (Instances, error)
}
