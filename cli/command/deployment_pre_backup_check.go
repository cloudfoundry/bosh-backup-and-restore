package command

import (
	"fmt"

	"github.com/cloudfoundry/bosh-cli/director"

	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/factory"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator"
	"github.com/urfave/cli"
)

type DeploymentPreBackupCheck struct{}

func NewDeploymentPreBackupCheckCommand() DeploymentPreBackupCheck {
	return DeploymentPreBackupCheck{}
}

func (d DeploymentPreBackupCheck) Cli() cli.Command {
	return cli.Command{
		Name:    "pre-backup-check",
		Aliases: []string{"c"},
		Usage:   "Check a deployment can be backed up",
		Action:  d.Action,
		Flags:   []cli.Flag{},
	}
}

func (d DeploymentPreBackupCheck) Action(c *cli.Context) error {
	username, password, target, caCert, debug, deployment, allDeployments := getDeploymentParams(c)

	logger := factory.BuildLogger(debug)
	boshClient, err := factory.BuildBoshClient(target, username, password, caCert, logger)
	if err != nil {
		return processError(orchestrator.NewError(err))
	}

	backupChecker := factory.BuildDeploymentBackupChecker(boshClient, logger, false)

	if allDeployments {
		allDeployments, err := getAllDeployments(boshClient)
		if err != nil {
			return processError(orchestrator.NewError(err))
		}

		errs := allDeploymentsBackupCheck(allDeployments, backupChecker)
		if errs != nil {
			return errs
		}
	} else {
		errs := backupableCheck(backupChecker, deployment)
		if errs != nil {
			return processError(errs)
		}
	}

	return cli.NewExitError("", 0)
}

func backupableCheck(backupChecker *orchestrator.BackupChecker, deployment string) orchestrator.Error {
	err := backupChecker.Check(deployment)
	if err != nil {
		fmt.Printf("Deployment '%s' cannot be backed up.\n", deployment)
		fmt.Println(err.Error())
		return err
	}

	fmt.Printf("Deployment '%s' can be backed up.\n", deployment)
	return nil
}

func allDeploymentsBackupCheck(deployments []director.Deployment, backupChecker *orchestrator.BackupChecker) error {
	var unbackupableDeploymentsErrors []deploymentError
	for _, deployment := range deployments {
		errs := backupableCheck(backupChecker, deployment.Name())
		if errs != nil {
			unbackupableDeploymentsErrors = append(unbackupableDeploymentsErrors, deploymentError{deployment: deployment.Name(), errs: errs})
		}
		fmt.Println("-------------------------")
	}

	if len(unbackupableDeploymentsErrors) != 0 {
		errMsg := fmt.Sprintf("%d out of %d deployments cannot be backed up:\n", len(unbackupableDeploymentsErrors), len(deployments))
		for _, deploymentErr := range unbackupableDeploymentsErrors {
			errMsg = errMsg + fmt.Sprintln(deploymentErr.deployment)
		}
		return allDeploymentsError{summary: errMsg, deploymentErrs: unbackupableDeploymentsErrors}.Process()
	}

	fmt.Printf("All %d deployments can be backed up.\n", len(deployments))
	return nil
}
