package orchestrator

import (
	"fmt"

	"strings"

	"github.com/pkg/errors"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/executor"
)

const ArtifactDirectory = "/var/vcap/store/bbr-backup"

//go:generate counterfeiter -o fakes/fake_deployment.go . Deployment
type Deployment interface {
	IsBackupable() bool
	HasUniqueCustomArtifactNames() bool
	CheckArtifactDir() error
	IsRestorable() bool
	PreBackupLock(LockOrderer, executor.Executor) error
	Backup() error
	PostBackupUnlock(LockOrderer, executor.Executor) error
	Restore() error
	CopyRemoteBackupToLocal(Backup, executor.Executor) error
	CopyLocalBackupToRemote(Backup, executor.Executor) error
	Cleanup() error
	CleanupPrevious() error
	Instances() []Instance
	CustomArtifactNamesMatch() error
	PreRestoreLock(LockOrderer, executor.Executor) error
	PostRestoreUnlock(LockOrderer, executor.Executor) error
	ValidateLockingDependencies(orderer LockOrderer) error
}

//go:generate counterfeiter -o fakes/fake_lock_orderer.go . LockOrderer
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

func (bd *deployment) HasUniqueCustomArtifactNames() bool {
	names := bd.instances.CustomArtifactNames()

	uniqueNames := map[string]bool{}
	for _, name := range names {
		if _, found := uniqueNames[name]; found {
			return false
		}
		uniqueNames[name] = true
	}
	return true
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
	bd.Logger.Info("bbr", "Running pre-backup-lock scripts...")

	jobs := bd.instances.Jobs()

	orderedJobs, err := lockOrderer.Order(jobs)
	if err != nil {
		return err
	}

	preBackupLockErrors := executor.Run(newJobExecutables(orderedJobs, NewJobPreBackupLockExecutable))

	bd.Logger.Info("bbr", "Finished running pre-backup-lock scripts.")
	return ConvertErrors(preBackupLockErrors)
}

func (bd *deployment) Backup() error {
	bd.Logger.Info("bbr", "Running backup scripts...")
	err := bd.instances.AllBackupable().Backup()
	bd.Logger.Info("bbr", "Finished running backup scripts.")
	return err
}

func (bd *deployment) PostBackupUnlock(lockOrderer LockOrderer, executor executor.Executor) error {
	bd.Logger.Info("bbr", "Running post-backup-unlock scripts...")

	jobs := bd.instances.Jobs()

	orderedJobs, err := lockOrderer.Order(jobs)
	if err != nil {
		return err
	}
	reversedJobs := Reverse(orderedJobs)

	postBackupUnlockErrors := executor.Run(newJobExecutables(reversedJobs, NewJobPostBackupUnlockExecutable))
	bd.Logger.Info("bbr", "Finished running post-backup-unlock scripts.")
	return ConvertErrors(postBackupUnlockErrors)
}

func (bd *deployment) PreRestoreLock(lockOrderer LockOrderer, executor executor.Executor) error {
	bd.Logger.Info("bbr", "Running pre-restore-lock scripts...")

	jobs := bd.instances.Jobs()

	orderedJobs, err := lockOrderer.Order(jobs)
	if err != nil {
		return err
	}

	preRestoreLockErrors := executor.Run(newJobExecutables(orderedJobs, NewJobPreRestoreLockExecutable))

	bd.Logger.Info("bbr", "Finished running pre-restore-lock scripts.")
	return ConvertErrors(preRestoreLockErrors)
}

func (bd *deployment) Restore() error {
	bd.Logger.Info("bbr", "Running restore scripts...")
	err := bd.instances.AllRestoreable().Restore()
	bd.Logger.Info("bbr", "Finished running restore scripts.")
	return err
}

func (bd *deployment) PostRestoreUnlock(lockOrderer LockOrderer, executor executor.Executor) error {
	bd.Logger.Info("bbr", "Running post-restore-unlock scripts...")

	jobs := bd.instances.Jobs()

	orderedJobs, err := lockOrderer.Order(jobs)
	if err != nil {
		return err
	}
	reversedJobs := Reverse(orderedJobs)

	postRestoreUnlockErrors := executor.Run(newJobExecutables(reversedJobs, NewJobPostRestoreUnlockExecutable))

	bd.Logger.Info("bbr", "Finished running post-restore-unlock scripts.")
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

func (bd *deployment) CustomArtifactNamesMatch() error {
	for _, instance := range bd.Instances() {
		jobName := instance.Name()
		for _, restoreName := range instance.CustomRestoreArtifactNames() {
			var found bool
			for _, backupName := range bd.instances.CustomArtifactNames() {
				if restoreName == backupName {
					found = true
				}
			}
			if !found {
				return errors.New(
					fmt.Sprintf(
						"The %s restore script expects a backup script which produces %s artifact which is not present in the deployment.",
						jobName,
						restoreName,
					),
				)
			}
		}
	}
	return nil
}

func (bd *deployment) CopyRemoteBackupToLocal(localBackup Backup, execr executor.Executor) error {
	instances := bd.instances.AllBackupable()

	var executables []executor.Executable
	for _, instance := range instances {
		for _, remoteBackupArtifact := range instance.ArtifactsToBackup() {
			executables = append(executables, newBackupDownloadExecutable(localBackup, remoteBackupArtifact, bd.Logger))
		}
	}

	errs := execr.Run([][]executor.Executable{executables})

	return ConvertErrors(errs)
}

type BackupDownloadExecutable struct {
	localBackup    Backup
	remoteArtifact BackupArtifact
	Logger
}

func newBackupDownloadExecutable(localBackup Backup, remoteArtifact BackupArtifact, logger Logger) BackupDownloadExecutable {
	return BackupDownloadExecutable{
		localBackup:    localBackup,
		remoteArtifact: remoteArtifact,
		Logger:         logger,
	}
}

func (e BackupDownloadExecutable) Execute() error {
	err := e.downloadBackupArtifact(e.localBackup, e.remoteArtifact)
	if err != nil {
		return err
	}

	checksum, err := e.compareChecksums(e.localBackup, e.remoteArtifact)
	if err != nil {
		return err
	}

	err = e.localBackup.AddChecksum(e.remoteArtifact, checksum)
	if err != nil {
		return err
	}

	err = e.remoteArtifact.Delete()
	if err != nil {
		return err
	}

	e.Logger.Info("bbr", "Finished validity checks -- from %s/%s...", e.remoteArtifact.InstanceName(), e.remoteArtifact.InstanceID())
	return nil
}

func (e BackupDownloadExecutable) downloadBackupArtifact(localBackup Backup, remoteBackupArtifact BackupArtifact) error {
	localBackupArtifactWriter, err := localBackup.CreateArtifact(remoteBackupArtifact)
	if err != nil {
		return err
	}

	size, err := remoteBackupArtifact.Size()
	if err != nil {
		return err
	}

	e.Logger.Info("bbr", "Copying backup -- %s uncompressed -- from %s/%s...", size, remoteBackupArtifact.InstanceName(), remoteBackupArtifact.InstanceID())
	err = remoteBackupArtifact.StreamFromRemote(localBackupArtifactWriter)
	if err != nil {
		return err
	}

	err = localBackupArtifactWriter.Close()
	if err != nil {
		return err
	}

	e.Logger.Info("bbr", "Finished copying backup -- from %s/%s...", remoteBackupArtifact.InstanceName(), remoteBackupArtifact.InstanceID())
	return nil
}

func (e BackupDownloadExecutable) compareChecksums(localBackup Backup, remoteBackupArtifact BackupArtifact) (BackupChecksum, error) {
	e.Logger.Info("bbr", "Starting validity checks -- from %s/%s...", remoteBackupArtifact.InstanceName(), remoteBackupArtifact.InstanceID())

	localChecksum, err := localBackup.CalculateChecksum(remoteBackupArtifact)
	if err != nil {
		return nil, err
	}

	remoteChecksum, err := remoteBackupArtifact.Checksum()
	if err != nil {
		return nil, err
	}

	e.Logger.Debug("bbr", "Comparing shasums")

	match, mismatchedFiles := localChecksum.Match(remoteChecksum)
	if !match {
		e.Logger.Debug("bbr", "Checksums didn't match for:")
		e.Logger.Debug("bbr", fmt.Sprintf("%v\n", mismatchedFiles))

		err = errors.Errorf(
			"Backup is corrupted, checksum failed for %s/%s %s - checksums don't match for %v. "+
				"Checksum failed for %d files in total",
			remoteBackupArtifact.InstanceName(), remoteBackupArtifact.InstanceID(), remoteBackupArtifact.Name(), getFirstTen(mismatchedFiles), len(mismatchedFiles))
		return nil, err
	}

	return localChecksum, nil
}

func (bd *deployment) CopyLocalBackupToRemote(localBackup Backup, execr executor.Executor) error {
	instances := bd.instances.AllRestoreable()

	var executables []executor.Executable
	for _, instance := range instances {
		for _, remoteBackupArtifact := range instance.ArtifactsToRestore() {
			executables = append(executables, newBackupUploadExecutable(localBackup, remoteBackupArtifact, instance, bd.Logger))
		}
	}

	errs := execr.Run([][]executor.Executable{executables})

	return ConvertErrors(errs)
}

type BackupUploadExecutable struct {
	localBackup    Backup
	remoteArtifact BackupArtifact
	instance       Instance
	Logger
}

func newBackupUploadExecutable(localBackup Backup, remoteArtifact BackupArtifact, instance Instance, logger Logger) BackupUploadExecutable {
	return BackupUploadExecutable{
		localBackup:    localBackup,
		remoteArtifact: remoteArtifact,
		instance:       instance,
		Logger:         logger,
	}
}

func (e BackupUploadExecutable) Execute() error {
	localBackupArtifactReader, err := e.localBackup.ReadArtifact(e.remoteArtifact)
	if err != nil {
		return err
	}

	e.Logger.Info("bbr", "Copying backup to %s/%s...", e.instance.Name(), e.instance.Index())
	err = e.remoteArtifact.StreamToRemote(localBackupArtifactReader)
	if err != nil {
		return err
	}

	e.instance.MarkArtifactDirCreated()

	localChecksum, err := e.localBackup.FetchChecksum(e.remoteArtifact)
	if err != nil {
		return err
	}

	remoteChecksum, err := e.remoteArtifact.Checksum()
	if err != nil {
		return err
	}

	match, mismatchedFiles := localChecksum.Match(remoteChecksum)
	if !match {
		e.Logger.Debug("bbr", "Checksums didn't match for:")
		e.Logger.Debug("bbr", fmt.Sprintf("%v\n", mismatchedFiles))
		return errors.Errorf("Backup couldn't be transferred, checksum failed for %s/%s %s - checksums don't match for %v. Checksum failed for %d files in total",
			e.instance.Name(),
			e.instance.ID(),
			e.remoteArtifact.Name(),
			getFirstTen(mismatchedFiles),
			len(mismatchedFiles),
		)
	}
	e.Logger.Info("bbr", "Finished copying backup to %s/%s.", e.instance.Name(), e.instance.Index())

	return nil
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
