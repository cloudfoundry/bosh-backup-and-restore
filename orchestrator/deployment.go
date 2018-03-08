package orchestrator

import (
	"fmt"

	"strings"

	"github.com/pkg/errors"
)

const ArtifactDirectory = "/var/vcap/store/bbr-backup"

//go:generate counterfeiter -o fakes/fake_deployment.go . Deployment
type Deployment interface {
	IsBackupable() bool
	HasUniqueCustomArtifactNames() bool
	CheckArtifactDir() error
	IsRestorable() bool
	PreBackupLock(LockOrderer, Executor) error
	Backup() error
	PostBackupUnlock(LockOrderer, Executor) error
	Restore() error
	CopyRemoteBackupToLocal(Backup, Executor) error
	CopyLocalBackupToRemote(Backup) error
	Cleanup() error
	CleanupPrevious() error
	Instances() []Instance
	CustomArtifactNamesMatch() error
	PreRestoreLock(LockOrderer, Executor) error
	PostRestoreUnlock(LockOrderer, Executor) error
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

func (bd *deployment) PreBackupLock(lockOrderer LockOrderer, executor Executor) error {
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

func (bd *deployment) PostBackupUnlock(lockOrderer LockOrderer, executor Executor) error {
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

func (bd *deployment) PreRestoreLock(lockOrderer LockOrderer, executor Executor) error {
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

func (bd *deployment) PostRestoreUnlock(lockOrderer LockOrderer, executor Executor) error {
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

func newJobExecutables(jobsList [][]Job, newJobExecutable func(Job) Executable) [][]Executable {
	var executablesList [][]Executable
	for _, jobs := range jobsList {
		var executables []Executable
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

func (bd *deployment) CopyRemoteBackupToLocal(localBackup Backup, executor Executor) error {
	instances := bd.instances.AllBackupable()

	var remoteBackupArtifacts []BackupArtifact
	for _, instance := range instances {
		remoteBackupArtifacts = append(remoteBackupArtifacts, instance.ArtifactsToBackup()...)
	}

	errs := executor.Run(newArtifactExecutables(remoteBackupArtifacts, localBackup, bd.Logger, newBackupDownloadExecutable))

	return ConvertErrors(errs)
}

func newArtifactExecutables(artifacts []BackupArtifact, backup Backup, logger Logger, newExecutable func(Backup, BackupArtifact, Logger) BackupDownloadExecutable) [][]Executable {
	var executables []Executable
	for _, artifact := range artifacts {
		executables = append(executables, newExecutable(backup, artifact, logger))
	}
	return [][]Executable{executables}
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

	e.Logger.Info("bbr", "Finished validity checks")
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
	e.Logger.Info("bbr", "Starting validity checks")

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

func (bd *deployment) CopyLocalBackupToRemote(backup Backup) error {
	instances := bd.instances.AllRestoreable()

	for _, instance := range instances {
		for _, artifact := range instance.ArtifactsToRestore() {
			reader, err := backup.ReadArtifact(artifact)

			if err != nil {
				return err
			}

			bd.Logger.Info("bbr", "Copying backup to %s/%s...", instance.Name(), instance.Index())
			if err := artifact.StreamToRemote(reader); err != nil {
				return err
			} else {
				instance.MarkArtifactDirCreated()
			}

			localChecksum, err := backup.FetchChecksum(artifact)
			if err != nil {
				return err
			}

			remoteChecksum, err := artifact.Checksum()
			if err != nil {
				return err
			}

			match, mismatchedFiles := localChecksum.Match(remoteChecksum)
			if !match {
				bd.Logger.Debug("bbr", "Checksums didn't match for:")
				bd.Logger.Debug("bbr", fmt.Sprintf("%v\n", mismatchedFiles))
				return errors.Errorf("Backup couldn't be transferred, checksum failed for %s/%s %s - checksums don't match for %v. Checksum failed for %d files in total",
					instance.Name(),
					instance.ID(),
					artifact.Name(),
					getFirstTen(mismatchedFiles),
					len(mismatchedFiles),
				)
			}
			bd.Logger.Info("bbr", "Finished copying backup to %s/%s.", instance.Name(), instance.Index())
		}
	}
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
