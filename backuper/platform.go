package backuper

type Platform interface {
	CheckIfBackupable() error
	Backup() error
	Cleanup() error
}

func NewBoshPlatform(deployments []Deployment) Platform {
	return &BoshPlatform{Deployments: deployments}
}

type BoshPlatform struct {
	Deployments []Deployment
}

func (p *BoshPlatform) CheckIfBackupable() error {
	return nil
}

func (p *BoshPlatform) Backup() error {
	return nil
}

func (p *BoshPlatform) Cleanup() error {
	return nil
}
