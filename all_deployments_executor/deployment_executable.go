package all_deployments_executor

import (
	"fmt"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator"
	"github.com/urfave/cli"
	"io/ioutil"
	"time"
)

type DeploymentExecutable struct {
	action ActionFunc
	name   string
}

type ActionFunc func(string) orchestrator.Error
type ErrorHandleFunc func(deploymentsError AllDeploymentsError) error

type DeploymentError struct {
	Deployment string
	Errs       orchestrator.Error
}

type AllDeploymentsError struct {
	Summary        string
	DeploymentErrs []DeploymentError
}

func (a AllDeploymentsError) Error() string {
	return ""
}

func (a AllDeploymentsError) Process() error {
	return a.ProcessWithFooter("")
}

func (a AllDeploymentsError) ProcessWithFooter(footer string) error {
	msg := fmt.Sprintln(a.Summary)
	msgWithStackTrace := msg

	for _, err := range a.DeploymentErrs {
		msg = msg + fmt.Sprintf("Deployment '%s': %s\n", err.Deployment, err.Errs.Error())
		msgWithStackTrace = msgWithStackTrace + fmt.Sprintf("Deployment %s: %s\n", err.Deployment, err.Errs.PrettyError(true))
	}

	if writeStackTrace(msgWithStackTrace) != nil {
		msg = msgWithStackTrace
	}

	if footer != "" {
		msg = msg + "\n" + footer
	}

	return cli.NewExitError(msg, 1)
}

func writeStackTrace(errorWithStackTrace string) error {
	if errorWithStackTrace != "" {
		err := ioutil.WriteFile(fmt.Sprintf("bbr-%s.err.log", time.Now().UTC().Format(time.RFC3339)), []byte(errorWithStackTrace), 0644)
		if err != nil {
			return err
		}
	}
	return nil
}

func NewDeploymentExecutable(action ActionFunc, name string) DeploymentExecutable {
	return DeploymentExecutable{
		action: action,
		name:   name,
	}
}

func (d DeploymentExecutable) Execute() DeploymentError {
	err := d.action(d.name)
	return DeploymentError{Deployment: d.name, Errs: err}
}
