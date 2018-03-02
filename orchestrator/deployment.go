package orchestrator

import (
	"fmt"

	"strings"

	"io"

	"github.com/pkg/errors"
)

const ArtifactDirectory = "/var/vcap/store/bbr-backup"

//go:generate counterfeiter -o fakes/fake_deployment.go . Deployment
type Deployment interface {
	IsBackupable() bool
	HasUniqueCustomArtifactNames() bool
	CheckArtifactDir() error
	IsRestorable() bool
	PreBackupLock(orderer LockOrderer, jobExecutionStategy JobExecutionStrategy) error
	Backup() error
	PostBackupUnlock(orderer LockOrderer, jobExecutionStategy JobExecutionStrategy) error
	Restore() error
	CopyRemoteBackupToLocal(Backup) error
	CopyRemoteBackupToLocalParallel(Backup) error
	CopyLocalBackupToRemote(Backup) error
	Cleanup() error
	CleanupPrevious() error
	Instances() []Instance
	CustomArtifactNamesMatch() error
	PreRestoreLock(orderer LockOrderer, jobExecutionStategy JobExecutionStrategy) error
	PostRestoreUnlock(orderer LockOrderer, jobExecutionStategy JobExecutionStrategy) error
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
	errs := []string{}

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

func (bd *deployment) PreBackupLock(lockOrderer LockOrderer, jobExecutionStategy JobExecutionStrategy) error {
	bd.Logger.Info("bbr", "Running pre-backup-lock scripts...")

	jobs := bd.instances.Jobs()

	orderedJobs, err := lockOrderer.Order(jobs)
	if err != nil {
		return err
	}

	preBackupLockErrors := jobExecutionStategy.Run(JobPreBackupLocker, orderedJobs)

	bd.Logger.Info("bbr", "Finished running pre-backup-lock scripts.")
	return ConvertErrors(preBackupLockErrors)
}

func (bd *deployment) Backup() error {
	bd.Logger.Info("bbr", "Running backup scripts...")
	err := bd.instances.AllBackupable().Backup()
	bd.Logger.Info("bbr", "Finished running backup scripts.")
	return err
}

func (bd *deployment) PostBackupUnlock(lockOrderer LockOrderer, jobExecutionStategy JobExecutionStrategy) error {
	bd.Logger.Info("bbr", "Running post-backup-unlock scripts...")

	jobs := bd.instances.Jobs()

	orderedJobs, err := lockOrderer.Order(jobs)
	if err != nil {
		return err
	}
	reversedJobs := Reverse(orderedJobs)

	postBackupUnlockErrors := jobExecutionStategy.Run(JobPostBackupUnlocker, reversedJobs)
	bd.Logger.Info("bbr", "Finished running post-backup-unlock scripts.")
	return ConvertErrors(postBackupUnlockErrors)
}

func (bd *deployment) PreRestoreLock(lockOrderer LockOrderer, jobExecutionStategy JobExecutionStrategy) error {
	bd.Logger.Info("bbr", "Running pre-restore-lock scripts...")

	jobs := bd.instances.Jobs()

	orderedJobs, err := lockOrderer.Order(jobs)
	if err != nil {
		return err
	}

	preRestoreLockErrors := jobExecutionStategy.Run(JobPreRestoreLocker, orderedJobs)

	bd.Logger.Info("bbr", "Finished running pre-restore-lock scripts.")
	return ConvertErrors(preRestoreLockErrors)
}

func (bd *deployment) Restore() error {
	bd.Logger.Info("bbr", "Running restore scripts...")
	err := bd.instances.AllRestoreable().Restore()
	bd.Logger.Info("bbr", "Finished running restore scripts.")
	return err
}

func (bd *deployment) PostRestoreUnlock(lockOrderer LockOrderer, jobExecutionStategy JobExecutionStrategy) error {
	bd.Logger.Info("bbr", "Running post-restore-unlock scripts...")

	jobs := bd.instances.Jobs()

	orderedJobs, err := lockOrderer.Order(jobs)
	if err != nil {
		return err
	}
	reversedJobs := Reverse(orderedJobs)

	postRestoreUnlockErrors := jobExecutionStategy.Run(JobPostRestoreUnlocker, reversedJobs)

	bd.Logger.Info("bbr", "Finished running post-restore-unlock scripts.")
	return ConvertErrors(postRestoreUnlockErrors)
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

func (bd *deployment) countArtifactsToStream() int {
	count := 0
	for _, instance := range bd.instances {
		count += len(instance.ArtifactsToBackup())
	}
	return count
}

func (bd *deployment) CopyRemoteBackupToLocalParallel(backup Backup) error {
	var errs []error
	instances := bd.instances.AllBackupable()

	copies := make(chan ArtifactCopy, bd.countArtifactsToStream())

	for _, instance := range instances {
		for _, backupArtifact := range instance.ArtifactsToBackup() {
			go func(b BackupArtifact, i Instance) {
				var (
					artifactCopy                  ArtifactCopy
					size                          string
					w                             io.WriteCloser
					localChecksum, remoteChecksum BackupChecksum
				)

				w, artifactCopy.err = backup.CreateArtifact(b)
				if artifactCopy.err != nil {
					copies <- artifactCopy
					return
				}

				size, artifactCopy.err = b.Size()
				if artifactCopy.err != nil {
					copies <- artifactCopy
					return
				}

				bd.Logger.Info("bbr", "Copying backup -- %s uncompressed -- from %s/%s...", size, i.Name(), i.ID())
				artifactCopy.err = b.StreamFromRemote(w)
				if artifactCopy.err != nil {
					copies <- artifactCopy
					return
				}

				artifactCopy.err = w.Close()
				if artifactCopy.err != nil {
					copies <- artifactCopy
					return
				}
				bd.Logger.Info("bbr", "Finished copying backup -- from %s/%s...", i.Name(), i.ID())

				bd.Logger.Info("bbr", "Starting validity checks")
				localChecksum, artifactCopy.err = backup.CalculateChecksum(b)
				if artifactCopy.err != nil {
					copies <- artifactCopy
					return
				}

				remoteChecksum, artifactCopy.err = b.Checksum()
				if artifactCopy.err != nil {
					copies <- artifactCopy
					return
				}
				bd.Logger.Debug("bbr", "Comparing shasums")

				match, mismatchedFiles := localChecksum.Match(remoteChecksum)
				if !match {
					bd.Logger.Debug("bbr", "Checksums didn't match for:")
					bd.Logger.Debug("bbr", fmt.Sprintf("%v\n", mismatchedFiles))

					artifactCopy.err = errors.Errorf(
						"Backup is corrupted, checksum failed for %s/%s %s - checksums don't match for %v. "+
							"Checksum failed for %d files in total",
						i.Name(), i.ID(), b.Name(), getFirstTen(mismatchedFiles), len(mismatchedFiles))
					copies <- artifactCopy
					return
				}

				artifactCopy.backupArtifact = b
				artifactCopy.checksum = localChecksum
				artifactCopy.err = nil
				copies <- artifactCopy
				bd.Logger.Info("bbr", "Finished validity checks")

			}(backupArtifact, instance)

		}

	}

	for i := 0; i < bd.countArtifactsToStream(); i++ {
		artifactCopy := <-copies
		if artifactCopy.err != nil {
			errs = append(errs, artifactCopy.err)
		} else {
			backup.AddChecksum(artifactCopy.backupArtifact, artifactCopy.checksum)
			err := artifactCopy.backupArtifact.Delete()
			if err != nil {
				errs = append(errs, err)
			}
		}
	}
	return ConvertErrors(errs)
}

func (bd *deployment) CopyRemoteBackupToLocal(backup Backup) error {
	instances := bd.instances.AllBackupable()
	for _, instance := range instances {
		for _, backupArtifact := range instance.ArtifactsToBackup() {
			writer, err := backup.CreateArtifact(backupArtifact)

			if err != nil {
				return err
			}

			size, err := backupArtifact.Size()
			if err != nil {
				return err
			}

			bd.Logger.Info("bbr", "Copying backup -- %s uncompressed -- from %s/%s...", size, instance.Name(), instance.ID())
			if err := backupArtifact.StreamFromRemote(writer); err != nil {
				return err
			}

			if err := writer.Close(); err != nil {
				return err
			}
			bd.Logger.Info("bbr", "Finished copying backup -- from %s/%s...", instance.Name(), instance.ID())

			bd.Logger.Info("bbr", "Starting validity checks")
			localChecksum, err := backup.CalculateChecksum(backupArtifact)
			if err != nil {
				return err
			}

			remoteChecksum, err := backupArtifact.Checksum()
			if err != nil {
				return err
			}
			bd.Logger.Debug("bbr", "Comparing shasums")

			match, mismatchedFiles := localChecksum.Match(remoteChecksum)
			if !match {
				bd.Logger.Debug("bbr", "Checksums didn't match for:")
				bd.Logger.Debug("bbr", fmt.Sprintf("%v\n", mismatchedFiles))
				return errors.Errorf("Backup is corrupted, checksum failed for %s/%s %s - checksums don't match for %v. Checksum failed for %d files in total", instance.Name(), instance.ID(), backupArtifact.Name(), getFirstTen(mismatchedFiles), len(mismatchedFiles))
			}

			backup.AddChecksum(backupArtifact, localChecksum)

			err = backupArtifact.Delete()
			if err != nil {
				return err
			}
			bd.Logger.Info("bbr", "Finished validity checks")
		}
	}
	return nil
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
