package command

import (
	"errors"
	"fmt"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/bosh"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/executor/deployment"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator"
	"github.com/cloudfoundry/bosh-cli/director"
	"github.com/urfave/cli"
)

func ContainsUnlockOrCleanup(deploymentErrs []deployment.DeploymentError) bool {
	for _, errs := range deploymentErrs {
		if errs.Errs.ContainsUnlockOrCleanup() {
			return true
		}
	}
	return false
}

func ContainsArtifactDir(deploymentErrs []deployment.DeploymentError) bool {
	for _, errs := range deploymentErrs {
		if errs.Errs.ContainsArtifactDirError() {
			return true
		}
	}
	return false
}

func getDeploymentParams(c *cli.Context) (string, string, string, string, bool, string, bool) {
	username := c.Parent().String("username")
	password := c.Parent().String("password")
	target := c.Parent().String("target")
	caCert := c.Parent().String("ca-cert")
	debug := c.GlobalBool("debug")
	deployment := c.Parent().String("deployment")
	allDeployments := c.Parent().Bool("all-deployments")

	return username, password, target, caCert, debug, deployment, allDeployments
}

func getAllDeployments(boshClient bosh.Client) ([]director.Deployment, error) {
	allDeployments, err := boshClient.Director.Deployments()
	if err != nil {
		return nil, orchestrator.NewError(err)
	}

	fmt.Printf("Found %d deployments:\n", len(allDeployments))
	for _, deployment := range allDeployments {
		fmt.Printf("  %s\n", deployment.Name())
	}
	fmt.Println("-------------------------")

	return allDeployments, nil
}

func runForAllDeployments(action ActionFunc, boshClient bosh.Client, summaryErrorMsg, summarySuccessMsg string, errorHandler deployment.ErrorHandleFunc, exec deployment.DeploymentExecutor) error {
	deployments, err := getAllDeployments(boshClient)
	if err != nil {
		return processError(orchestrator.NewError(err))
	}

	if len(deployments) == 0 {
		return processError(orchestrator.NewError(errors.New("Failed to find any deployments")))
	}

	var executables []deployment.Executable
	for _, deployment := range deployments {
		executables = append(executables, NewDeploymentExecutable(action, deployment.Name()))
	}
	errs := exec.Run(executables)

	fmt.Println("-------------------------")
	if len(errs) != 0 {
		errMsg := fmt.Sprintf("%d out of %d deployments %s:\n", len(errs), len(deployments), summaryErrorMsg)
		for _, cleanupErr := range errs {
			errMsg = errMsg + "  " + cleanupErr.Deployment + "\n"
		}

		return errorHandler(deployment.AllDeploymentsError{Summary: errMsg, DeploymentErrs: errs})
	}

	fmt.Printf("All %d deployments %s.\n", len(deployments), summarySuccessMsg)
	return cli.NewExitError("", 0)

}

type DeploymentExecutable struct {
	action ActionFunc
	name   string
}

type ActionFunc func(string) orchestrator.Error

func NewDeploymentExecutable(action ActionFunc, name string) DeploymentExecutable {
	return DeploymentExecutable{
		action: action,
		name:   name,
	}
}

func (d DeploymentExecutable) Execute() deployment.DeploymentError {
	err := d.action(d.name)
	return deployment.DeploymentError{Deployment: d.name, Errs: err}
}
