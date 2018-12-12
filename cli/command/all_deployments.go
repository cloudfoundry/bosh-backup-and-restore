package command

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudfoundry/bosh-utils/logger"

	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/bosh"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/executor/deployment"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/factory"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator"
	"github.com/urfave/cli"
)

func runForAllDeployments(action ActionFunc, boshClient bosh.Client, summaryErrorMsg, summarySuccessMsg string, errorHandler deployment.ErrorHandleFunc, executor deployment.DeploymentExecutor) error {
	deployments, err := getAllDeployments(boshClient)
	if err != nil {
		return processError(orchestrator.NewError(err))
	}

	printPending(deployments)

	executables := createExecutables(deployments, action)
	errs := executor.Run(executables)
	successfulDeployments, failedDeployments := getDeploymentStates(deployments, errs)

	printSuccess(summarySuccessMsg, successfulDeployments)

	if len(errs) != 0 {
		printFailed(failedDeployments)
		errMsg := summaryError(errs, deployments, summaryErrorMsg)
		return errorHandler(deployment.AllDeploymentsError{Summary: errMsg, DeploymentErrs: errs})
	}

	return cli.NewExitError("", 0)

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

	if len(deploymentNames) == 0 {
		return nil, processError(orchestrator.NewError(errors.New("Failed to find any deployments")))
	}

	return deploymentNames, nil
}

func printFailed(failedDeployments []string) {
	printlnWithTimestamp(fmt.Sprintf("FAILED: %s", strings.Join(failedDeployments, ", ")))
}

func printSuccess(summarySuccessMsg string, successfulDeployments []string) {
	printlnWithTimestamp("-------------------------")
	printlnWithTimestamp(fmt.Sprintf("Successfully %s: %s", summarySuccessMsg, strings.Join(successfulDeployments, ", ")))
}

func printPending(deployments []string) {
	printlnWithTimestamp(fmt.Sprintf("Pending: %s", strings.Join(deployments, ", ")))
	printlnWithTimestamp("-------------------------")
}

func summaryError(errs []deployment.DeploymentError, deployments []string, summaryErrorMsg string) string {
	errMsg := fmt.Sprintf("%d out of %d deployments %s:\n", len(errs), len(deployments), summaryErrorMsg)
	for _, depErr := range errs {
		errMsg = errMsg + "  " + depErr.Deployment + "\n"
	}
	return errMsg
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

func createLogger(timestamp string, artifactPath string, deploymentName string, debug bool) (string, *bytes.Buffer, logger.Logger) {
	logFilePath := filepath.Join(artifactPath, fmt.Sprintf("%s_%s.log", deploymentName, timestamp))
	logFile, _ := os.OpenFile(logFilePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, defaultLogfilePermissions)
	buffer := new(bytes.Buffer)
	multiWriter := io.MultiWriter(buffer, logFile)
	logger := factory.BuildBoshLoggerWithCustomWriter(multiWriter, debug)
	return logFilePath, buffer, logger
}

func (d DeploymentExecutable) Execute() deployment.DeploymentError {
	err := d.action(d.name)
	return deployment.DeploymentError{Deployment: d.name, Errs: err}
}
