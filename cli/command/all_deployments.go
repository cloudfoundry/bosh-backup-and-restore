package command

import (
	"errors"
	"fmt"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/bosh"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/executor/deployment"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator"
	"github.com/urfave/cli"
	"strings"
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

func getAllDeployments(boshClient bosh.Client) ([]string, error) {
	allDeployments, err := boshClient.Director.Deployments()
	if err != nil {
		return nil, orchestrator.NewError(err)
	}

	deploymentNames := []string{}
	for _, dep := range allDeployments {
		deploymentNames = append(deploymentNames, dep.Name())
	}

	return deploymentNames, nil
}

func runForAllDeployments(action ActionFunc, boshClient bosh.Client, summaryErrorMsg, summarySuccessMsg string, errorHandler deployment.ErrorHandleFunc, exec deployment.DeploymentExecutor) error {
	deployments, err := getAllDeployments(boshClient)
	if err != nil {
		return processError(orchestrator.NewError(err))
	}

	if len(deployments) == 0 {
		return processError(orchestrator.NewError(errors.New("Failed to find any deployments")))
	}

	fmt.Printf("Pending: %s\n", strings.Join(deployments, ", "))
	fmt.Println("-------------------------")

	executables := createExecutables(deployments, action)
	errs := exec.Run(executables)

	successfullDeployments, failedDeployments := getDeploymentStates(deployments, errs)

	fmt.Println("-------------------------")
	fmt.Printf("Successfully %s: %s\n", summarySuccessMsg, strings.Join(successfullDeployments, ", "))

	if len(errs) != 0 {

		fmt.Printf("FAILED: %s\n", strings.Join(failedDeployments, ", "))
		errMsg := fmt.Sprintf("%d out of %d deployments %s:\n", len(errs), len(deployments), summaryErrorMsg)
		for _, cleanupErr := range errs {
			errMsg = errMsg + "  " + cleanupErr.Deployment + "\n"
		}

		return errorHandler(deployment.AllDeploymentsError{Summary: errMsg, DeploymentErrs: errs})
	}
	return cli.NewExitError("", 0)

}

func createExecutables(deployments []string, action ActionFunc) []deployment.Executable {
	var executables []deployment.Executable
	for _, deploymentName := range deployments {
		executables = append(executables, NewDeploymentExecutable(action, deploymentName))
	}
	return executables
}

func getDeploymentStates(allDeployments []string, errs []deployment.DeploymentError) ([]string, []string) {
	failedDeployments := []string{}
	for _, depErr := range errs {
		failedDeployments = append(failedDeployments, depErr.Deployment)
	}

	successfullDeployments := []string{}
	for _, dep := range allDeployments {
		if !contains(failedDeployments, dep) {
			successfullDeployments = append(successfullDeployments, dep)
		}
	}

	return successfullDeployments, failedDeployments
}

func contains(list []string, item string) bool {
	for _, str := range list {
		if str == item {
			return true
		}
	}
	return false
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
