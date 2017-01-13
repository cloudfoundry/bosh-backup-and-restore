package backuper

import (
	"github.com/looplab/fsm"
	"time"
	"fmt"
)

type backupWorkflow struct {
	Backuper
	*fsm.FSM

	backupErrors Error
	deploymentName string
	events       []fsm.EventDesc
	deployment Deployment
	artifact Artifact
}

const (
	StateReady           = "ready"
	StateIsBackupable    = "is-backupable"
	StateArtifactCreated = "artifact-created"
	StateLocked          = "locked"
	StateBackedup        = "backed-up"
	StateUnlocked        = "unlocked"
	StateDrained         = "drained"
	StateFinished        = "finished"
)
const (
	EventCheckIsBackupable = "check-is-backupable"
	EventCreateArtifact    = "create-artifact"
	EventPrebackupLock     = "pre-backup-lock"
	EventBackup            = "backup"
	EventPostBackupUnlock  = "post-backup-unlock"
	EventDrain             = "drain"
	EventCleanup           = "cleanup"
)

func newbackupWorkflow(backuper Backuper, deploymentName string, deployment Deployment) *backupWorkflow {
	bw := &backupWorkflow{
		Backuper:backuper,
		deployment:deployment,
		deploymentName:deploymentName,
		events: fsm.Events{
			{Name: EventCheckIsBackupable, Src: []string{StateReady}, Dst: StateIsBackupable},
			{Name: EventCreateArtifact, Src: []string{StateIsBackupable}, Dst: StateArtifactCreated},
			{Name: EventPrebackupLock, Src: []string{StateArtifactCreated}, Dst: StateLocked},
			{Name: EventBackup, Src: []string{StateLocked}, Dst: StateBackedup},
			{Name: EventPostBackupUnlock, Src: []string{StateBackedup}, Dst: StateUnlocked},
			{Name: EventDrain, Src: []string{StateUnlocked}, Dst: StateDrained},
			{Name: EventCleanup, Src: []string{StateReady, StateIsBackupable, StateArtifactCreated, StateUnlocked, StateDrained}, Dst: StateFinished},
		},
	}


	bw.FSM = fsm.NewFSM(
		StateReady,
		bw.events,
		fsm.Callbacks{
			beforeEvent(EventCheckIsBackupable): bw.checkIsBackupable,
			beforeEvent(EventCreateArtifact):    bw.createArtifact,
			beforeEvent(EventPrebackupLock):     bw.prebackupLock,
			beforeEvent(EventBackup):            bw.backup,
			beforeEvent(EventPostBackupUnlock):  bw.postBackupUnlock,
			beforeEvent(EventDrain):             bw.drain,
			EventCleanup:                        bw.cleanup,
		},
	)

	return bw
}

func(bw *backupWorkflow) Run() Error {
	for _, e := range bw.events {
		if bw.Can(e.Name) {
			bw.Event(e.Name) //TODO: err
		}
	}
	return bw.backupErrors
}

func(bw *backupWorkflow) checkIsBackupable(e *fsm.Event) {
	backupable, err := bw.deployment.IsBackupable()

	if err != nil {
		bw.backupErrors = append(bw.backupErrors, err)
		e.Cancel()
		return
	}

	if !backupable {
		bw.backupErrors = append(bw.backupErrors, fmt.Errorf("Deployment '%s' has no backup scripts", bw.deploymentName))
		e.Cancel()
	}
}

func(bw *backupWorkflow) cleanup(e *fsm.Event) {
	if err := bw.deployment.Cleanup(); err != nil {
		bw.backupErrors = append(bw.backupErrors, CleanupError{fmt.Errorf("Deployment '%s' failed while cleaning up with error: %v", bw.deploymentName, err)})
	}
}

func(bw *backupWorkflow) createArtifact(e *fsm.Event) {
	var err error
	bw.artifact, err = bw.ArtifactManager.Create(bw.deploymentName, bw.Logger)
	if err != nil {
		bw.backupErrors = append(bw.backupErrors, err)
		e.Cancel()
		return
	}

	manifest, err := bw.GetManifest(bw.deploymentName)
	if err != nil {
		bw.backupErrors = append(bw.backupErrors, err)
		e.Cancel()
		return
	}

	err = bw.artifact.SaveManifest(manifest)
	if err != nil {
		bw.backupErrors = append(bw.backupErrors, err)
		e.Cancel()
		return
	}
}

func(bw *backupWorkflow) prebackupLock(e *fsm.Event) {
	err := bw.deployment.PreBackupLock()

	if err != nil {
		bw.backupErrors = append(bw.backupErrors, err)
		e.Cancel()
	}
}

func(bw *backupWorkflow) backup(e *fsm.Event) {
	err := bw.deployment.Backup()

	if err != nil {
		bw.backupErrors = append(bw.backupErrors, err)
	}
}

func(bw *backupWorkflow) drain(e *fsm.Event) {
	if bw.backupErrors.IsFatal() {
		e.Cancel()
		return
	}
	err := bw.deployment.CopyRemoteBackupToLocal(bw.artifact)

	if err != nil {
		bw.backupErrors = append(bw.backupErrors, err)
		return
	}

	bw.Logger.Info("", "Backup created of %s on %v\n", bw.deploymentName, time.Now())
}

func(bw *backupWorkflow) postBackupUnlock(e *fsm.Event) {
	err := bw.deployment.PostBackupUnlock()

	if err != nil {
		bw.backupErrors = append(bw.backupErrors, PostBackupUnlockError{err})
	}
}