package orchestrator

import (
	"fmt"

	"strings"

	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/executor"
	"github.com/pkg/errors"
)

const ArtifactDirectory = "/var/vcap/store/bbr-backup"

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -generate
//counterfeiter:generate -o fakes/fake_deployment.go . Deployment
type Deployment interface {
	IsBackupable() bool
	BackupableInstances() []Instance
	CheckArtifactDir() error
	IsRestorable() bool
	RestorableInstances() []Instance
	PreBackupLock(LockOrderer, executor.Executor) error
	Backup(executor.Executor) error
	PostBackupUnlock(bool, LockOrderer, executor.Executor) error
	Restore() error
	Cleanup() error
	CleanupPrevious() error
	Instances() []Instance
	PreRestoreLock(LockOrderer, executor.Executor) error
	PostRestoreUnlock(LockOrderer, executor.Executor) error
	ValidateLockingDependencies(orderer LockOrderer) error
}

//counterfeiter:generate -o fakes/fake_lock_orderer.go . LockOrderer
type LockOrderer interface {
	Order(jobs []Job) ([][]Job, error)
}

type deployment struct {
	Logger
	instances instances
}

func NewDeployment(logger Logger, instancesArray []Instance) Deployment {
	return &deployment{Logger: logger, instances: instances(instancesArray)}
}

func (bd *deployment) IsBackupable() bool {
	backupableInstances := bd.instances.AllBackupable()
	return !backupableInstances.IsEmpty()
}

func (bd *deployment) BackupableInstances() []Instance {
	return bd.instances.AllBackupable()
}

func (bd *deployment) CheckArtifactDir() error {
	var errs []string

	for _, inst := range bd.instances {
		exists, err := inst.ArtifactDirExists()
		if err != nil {
			errs = append(errs, fmt.Sprintf("Error checking %s on instance %s/%s", ArtifactDirectory, inst.Name(), inst.ID()))
		} else if exists {
			errs = append(errs, fmt.Sprintf("Directory %s already exists on instance %s/%s", ArtifactDirectory, inst.Name(), inst.ID()))
		}
	}

	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "\n"))
	}

	return nil
}

func (bd *deployment) ValidateLockingDependencies(lockOrderer LockOrderer) error {
	jobs := bd.instances.Jobs()
	_, err := lockOrderer.Order(jobs)
	return err
}

func (bd *deployment) PreBackupLock(lockOrderer LockOrderer, executor executor.Executor) error {
	bd.Logger.Info("bbr", "Running pre-backup-lock scripts...") //nolint:staticcheck

	jobs := bd.instances.Jobs()

	orderedJobs, err := lockOrderer.Order(jobs)
	if err != nil {
		return err
	}

	preBackupLockErrors := executor.Run(newJobExecutables(orderedJobs, NewJobPreBackupLockExecutable))

	bd.Logger.Info("bbr", "Finished running pre-backup-lock scripts.") //nolint:staticcheck
	return ConvertErrors(preBackupLockErrors)
}

func (bd *deployment) Backup(exe executor.Executor) error {
	bd.Logger.Info("bbr", "Running backup scripts...") //nolint:staticcheck

	instances := bd.instances.AllBackupable()

	var executables []executor.Executable
	for _, i := range instances {
		i.MarkArtifactDirCreated()
		for _, j := range i.Jobs() {
			executables = append(executables, NewBackupExecutable(j))
		}
	}

	backupErr := exe.Run([][]executor.Executable{executables})

	bd.Logger.Info("bbr", "Finished running backup scripts.") //nolint:staticcheck
	return ConvertErrors(backupErr)
}

func (bd *deployment) PostBackupUnlock(afterSuccessfulBackup bool, lockOrderer LockOrderer, executor executor.Executor) error {
	bd.Logger.Info("bbr", "Running post-backup-unlock scripts...") //nolint:staticcheck

	jobs := bd.instances.Jobs()

	orderedJobs, err := lockOrderer.Order(jobs)
	if err != nil {
		return err
	}
	reversedJobs := Reverse(orderedJobs)

	executableJobConstructor := NewJobPostFailedBackupUnlockExecutable
	if afterSuccessfulBackup {
		executableJobConstructor = NewJobPostSuccessfulBackupUnlockExecutable

	}
	postBackupUnlockErrors := executor.Run(newJobExecutables(reversedJobs, executableJobConstructor))

	bd.Logger.Info("bbr", "Finished running post-backup-unlock scripts.") //nolint:staticcheck
	return ConvertErrors(postBackupUnlockErrors)
}

func (bd *deployment) PreRestoreLock(lockOrderer LockOrderer, executor executor.Executor) error {
	bd.Logger.Info("bbr", "Running pre-restore-lock scripts...") //nolint:staticcheck

	jobs := bd.instances.Jobs()

	orderedJobs, err := lockOrderer.Order(jobs)
	if err != nil {
		return err
	}

	preRestoreLockErrors := executor.Run(newJobExecutables(orderedJobs, NewJobPreRestoreLockExecutable))

	bd.Logger.Info("bbr", "Finished running pre-restore-lock scripts.") //nolint:staticcheck
	return ConvertErrors(preRestoreLockErrors)
}

func (bd *deployment) Restore() error {
	bd.Logger.Info("bbr", "Running restore scripts...") //nolint:staticcheck
	err := bd.instances.AllRestoreable().Restore()
	bd.Logger.Info("bbr", "Finished running restore scripts.") //nolint:staticcheck
	return err
}

func (bd *deployment) PostRestoreUnlock(lockOrderer LockOrderer, executor executor.Executor) error {
	bd.Logger.Info("bbr", "Running post-restore-unlock scripts...") //nolint:staticcheck

	jobs := bd.instances.Jobs()

	orderedJobs, err := lockOrderer.Order(jobs)
	if err != nil {
		return err
	}
	reversedJobs := Reverse(orderedJobs)

	postRestoreUnlockErrors := executor.Run(newJobExecutables(reversedJobs, NewJobPostRestoreUnlockExecutable))

	bd.Logger.Info("bbr", "Finished running post-restore-unlock scripts.") //nolint:staticcheck
	return ConvertErrors(postRestoreUnlockErrors)
}

func newJobExecutables(jobsList [][]Job, newJobExecutable func(Job) executor.Executable) [][]executor.Executable {
	var executablesList [][]executor.Executable
	for _, jobs := range jobsList {
		var executables []executor.Executable
		for _, job := range jobs {
			executables = append(executables, newJobExecutable(job))
		}
		executablesList = append(executablesList, executables)
	}
	return executablesList
}

func (bd *deployment) Cleanup() error {
	return bd.instances.Cleanup()
}

func (bd *deployment) CleanupPrevious() error {
	return bd.instances.AllBackupableOrRestorable().CleanupPrevious()
}

func (bd *deployment) IsRestorable() bool {
	restoreableInstances := bd.instances.AllRestoreable()
	return !restoreableInstances.IsEmpty()
}

func (bd *deployment) RestorableInstances() []Instance {
	return bd.instances.AllRestoreable()
}

func (bd *deployment) Instances() []Instance {
	return bd.instances
}

func getFirstTen(input []string) (output []string) {
	for i := 0; i < len(input); i++ {
		if i == 10 {
			break
		}
		output = append(output, input[i])
	}
	return output
}
