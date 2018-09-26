package command

import (
	"fmt"

	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/bosh"

	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/factory"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator"
	"github.com/urfave/cli"
)

type DeploymentBackupCommand struct {
}

func NewDeploymentBackupCommand() DeploymentBackupCommand {
	return DeploymentBackupCommand{}
}

func (d DeploymentBackupCommand) Cli() cli.Command {
	return cli.Command{
		Name:    "backup",
		Aliases: []string{"b"},
		Usage:   "Backup a deployment",
		Action:  d.Action,
		Flags: []cli.Flag{
			cli.BoolFlag{
				Name:  "with-manifest",
				Usage: "Download the deployment manifest",
			},
			cli.StringFlag{
				Name:  "artifact-path",
				Usage: "Specify an optional path to save the backup artifacts to",
			},
		},
	}
}

func (d DeploymentBackupCommand) Action(c *cli.Context) error {
	trapSigint(true)

	username, password, target, caCert, debug, deployment, allDeployments := getDeploymentParams(c)
	withManifest := c.Bool("with-manifest")
	artifactPath := c.String("artifact-path")

	logger := factory.BuildLogger(debug)
	boshClient, err := factory.BuildBoshClient(target, username, password, caCert, logger)
	if err != nil {
		return processError(orchestrator.NewError(err))
	}
	backuper := factory.BuildDeploymentBackuper(withManifest, boshClient, logger)

	if !allDeployments {
		backupErr := backuper.Backup(deployment, artifactPath)

		if backupErr.ContainsUnlockOrCleanup() {
			return processErrorWithFooter(backupErr, backupCleanupAdvisedNotice)
		} else {
			return processError(backupErr)
		}
	}

	return backupAll(backuper, boshClient, artifactPath)
}

func backupAll(backuper *orchestrator.Backuper, boshClient bosh.Client, artifactPath string) error {
	deployments, err := getAllDeployments(boshClient)
	if err != nil {
		return processError(orchestrator.NewError(err))
	}

	var unbackupableDeploymentsErrors []deploymentError
	for _, deployment := range deployments {
		errs := backuper.Backup(deployment.Name(), artifactPath)
		if errs != nil {
			unbackupableDeploymentsErrors = append(unbackupableDeploymentsErrors, deploymentError{deployment: deployment.Name(), errs: errs})
		}
		fmt.Println("-------------------------")
	}
	if unbackupableDeploymentsErrors != nil {
		errMsg := fmt.Sprintf("%d out of %d deployments cannot be backed up:\n", len(unbackupableDeploymentsErrors), len(deployments))

		//if errors.ContainsUnlockOrCleanup() {
		//return processErrorWithFooter(errors, backupCleanupAdvisedNotice)
		//}
		return allDeploymentsError{summary: errMsg, deploymentErrs: unbackupableDeploymentsErrors}.Process()
	}

	fmt.Printf("All %d deployments backed up.\n", len(deployments))

	return cli.NewExitError("", 0)
}
