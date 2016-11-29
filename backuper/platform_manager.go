package backuper

import "strings"

type PlatformManager interface {
	Find(query string) (Platform, error)
}

type BoshPlatformManager struct {
	DeploymentManager
	Logger
}

func NewBoshPlatformManager(deploymentManager DeploymentManager, logger Logger) PlatformManager {
	return &BoshPlatformManager{DeploymentManager: deploymentManager, Logger: logger}
}

func (m BoshPlatformManager) Find(query string) (Platform, error) {
	var deployments []Deployment
	for _, deploymentName := range strings.Split(query, ",") {
		deployment, err := m.DeploymentManager.Find(deploymentName)
		if err != nil {
			return nil, err
		}
		deployments = append(deployments, deployment)
	}
	return NewBoshPlatform(deployments), nil
}
