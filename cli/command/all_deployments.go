package command

import (
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

func (a allDeploymentsError) Error() string {
	return ""
}

func (a allDeploymentsError) Process() error {
	msg := fmt.Sprintln(a.summary)
	msgWithStackTrace := msg

	for _, err := range a.deploymentErrs {
		msg = msg + fmt.Sprintf("Deployment '%s': %s\n", err.deployment, err.errs.Error())
		msgWithStackTrace = msgWithStackTrace + fmt.Sprintf("%s: %s\n", err.deployment, err.errs.PrettyError(true))
	}

	if writeStackTrace(msgWithStackTrace) != nil {
		return cli.NewExitError(msgWithStackTrace, 1)
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
