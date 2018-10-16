package command

import (
	"errors"
	"fmt"

	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/bosh"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator"
	"github.com/cloudfoundry/bosh-cli/director"
	"github.com/urfave/cli"
)

type deploymentError struct {
	deployment string
	errs       orchestrator.Error
}

type allDeploymentsError struct {
	summary        string
	deploymentErrs []deploymentError
}

func ContainsUnlockOrCleanup(deploymentErrs []deploymentError) bool {
	for _, errs := range deploymentErrs {
		if errs.errs.ContainsUnlockOrCleanup() {
			return true
		}
	}
	return false
}

func ContainsArtifactDir(deploymentErrs []deploymentError) bool {
	for _, errs := range deploymentErrs {
		if errs.errs.ContainsArtifactDirError() {
			return true
		}
	}
	return false
}

func (a allDeploymentsError) Error() string {
	return ""
}

func (a allDeploymentsError) Process() error {
	return a.ProcessWithFooter("")
}

func (a allDeploymentsError) ProcessWithFooter(footer string) error {
	msg := fmt.Sprintln(a.summary)
	msgWithStackTrace := msg

	for _, err := range a.deploymentErrs {
		msg = msg + fmt.Sprintf("Deployment '%s': %s\n", err.deployment, err.errs.Error())
		msgWithStackTrace = msgWithStackTrace + fmt.Sprintf("Deployment %s: %s\n", err.deployment, err.errs.PrettyError(true))
	}

	if writeStackTrace(msgWithStackTrace) != nil {
		msg = msgWithStackTrace
	}

	if footer != "" {
		msg = msg + "\n" + footer
	}

	return cli.NewExitError(msg, 1)
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

type actionFunc func(string) orchestrator.Error
type errorHandleFunc func(deploymentsError allDeploymentsError) error

func runForAllDeployments(action actionFunc, boshClient bosh.Client, summaryErrorMsg, summarySuccessMsg string, errorHandler errorHandleFunc) error {
	deployments, err := getAllDeployments(boshClient)
	if err != nil {
		return processError(orchestrator.NewError(err))
	}

	if len(deployments) == 0 {
		return processError(orchestrator.NewError(errors.New("Failed to find any deployments")))
	}

	var errs []deploymentError

	for _, deployment := range deployments {
		err := action(deployment.Name())
		if err != nil {
			errs = append(errs, deploymentError{deployment: deployment.Name(), errs: err})
		}
		fmt.Println("-------------------------")
	}

	if len(errs) != 0 {
		errMsg := fmt.Sprintf("%d out of %d deployments %s:\n", len(errs), len(deployments), summaryErrorMsg)
		for _, cleanupErr := range errs {
			errMsg = errMsg + cleanupErr.deployment + "\n"
		}

		return errorHandler(allDeploymentsError{summary: errMsg, deploymentErrs: errs})
	}

	fmt.Printf("All %d deployments %s.\n", len(deployments), summarySuccessMsg)
	return cli.NewExitError("", 0)

}
