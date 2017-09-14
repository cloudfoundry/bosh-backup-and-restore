package orchestrator

import (
	"fmt"
	"time"

	"github.com/pkg/errors"
)

func NewBackuper(backupManager BackupManager, logger Logger, deploymentManager DeploymentManager, lockOrderer LockOrderer, nowFunc func() time.Time) *Backuper {
	return &Backuper{
		BackupManager:     backupManager,
		Logger:            logger,
		DeploymentManager: deploymentManager,
		NowFunc:           nowFunc,
		LockOrderer:       lockOrderer,
	}
}

//go:generate counterfeiter -o fakes/fake_logger.go . Logger
type Logger interface {
	Debug(tag, msg string, args ...interface{})
	Info(tag, msg string, args ...interface{})
	Warn(tag, msg string, args ...interface{})
	Error(tag, msg string, args ...interface{})
}

//go:generate counterfeiter -o fakes/fake_deployment_manager.go . DeploymentManager
type DeploymentManager interface {
	Find(deploymentName string) (Deployment, error)
	SaveManifest(deploymentName string, artifact Backup) error
}

type Backuper struct {
	BackupManager
	Logger
	LockOrderer

	DeploymentManager
	NowFunc func() time.Time
}

type AuthInfo struct {
	Type   string
	UaaUrl string
}

//Backup checks if a deployment has backupable instances and backs them up.
func (b Backuper) Backup(deploymentName string) Error {
	session := NewSession(deploymentName)
	workflow := b.newBackupWorkflow()

	err := workflow.Run(session)

	if !err.IsFatal() {
		session.CurrentArtifact().AddFinishTime(b.NowFunc())
	}

	return err
}

func (b Backuper) newBackupWorkflow() *Workflow {
	checkDeployment := NewCheckDeploymentStep(b.DeploymentManager, b.Logger)
	backupable := NewBackupableStep(b.LockOrderer)
	createArtifact := NewCreateArtifactStep(b.Logger, b.BackupManager, b.DeploymentManager, b.NowFunc)
	lock := NewLockStep(b.LockOrderer)
	backup := NewBackupStep()
	unlockAfterSuccessfulBackup := NewUnlockStep(b.LockOrderer)
	unlockAfterFailedBackup := NewUnlockStep(b.LockOrderer)
	drain := NewDrainStep(b.Logger)
	cleanup := NewCleanupStep()

	workflow := NewWorkflow()
	workflow.StartWith(checkDeployment).OnSuccess(backupable)
	workflow.Add(backupable).OnSuccess(createArtifact).OnFailure(cleanup)
	workflow.Add(createArtifact).OnSuccess(lock).OnFailure(cleanup)
	workflow.Add(lock).OnSuccess(backup).OnFailure(unlockAfterFailedBackup)
	workflow.Add(backup).OnSuccess(unlockAfterSuccessfulBackup).OnFailure(unlockAfterFailedBackup)
	workflow.Add(unlockAfterSuccessfulBackup).OnSuccessOrFailure(drain)
	workflow.Add(unlockAfterFailedBackup).OnSuccessOrFailure(cleanup)
	workflow.Add(drain).OnSuccessOrFailure(cleanup)
	workflow.Add(cleanup)

	return workflow
}

func (workflow *Workflow) Run(session *Session) Error {
	var errs Error
	currentNode := workflow.StartingNode

	for currentNode != nil {
		err := currentNode.step.Run(session)
		if err != nil {
			errs = append(errs, err)
			currentNode = workflow.findNode(currentNode.failStep)
		} else {
			currentNode = workflow.findNode(currentNode.successStep)
		}
	}

	return errs
}

func (workflow *Workflow) findNode(step Step) *Node {
	if step == nil {
		return nil
	}
	for _, value := range workflow.Nodes {
		if value.step == step {
			return value
		}
	}
	//TODO: replace with something else
	panic("node not found")
	return nil
}

func (node *Node) OnSuccessOrFailure(step Step) *Node {
	return node.OnSuccess(step).OnFailure(step)
}

func (node *Node) OnFailure(failStep Step) *Node {
	node.failStep = failStep
	return node
}

func (node *Node) OnSuccess(successStep Step) *Node {
	node.successStep = successStep
	return node
}

type Node struct {
	step        Step
	successStep Step
	failStep    Step
}

func (workflow *Workflow) Add(step Step) *Node {
	node := NewNode(step)
	workflow.Nodes = append(workflow.Nodes, node)
	return node
}

func (workflow *Workflow) StartWith(step Step) *Node {
	node := workflow.Add(step)
	workflow.StartingNode = node
	return node
}
func NewNode(step Step) *Node {
	return &Node{step: step}
}

type Workflow struct {
	StartingNode *Node
	Nodes        []*Node
}

func NewWorkflow() *Workflow {
	return &Workflow{}
}

type Session struct {
	deploymentName  string
	deployment      Deployment
	currentArtifact Backup
}

func NewSession(deploymentName string) *Session {
	return &Session{deploymentName: deploymentName}
}

func (session *Session) SetCurrentArtifact(artifact Backup) {
	session.currentArtifact = artifact
}

func (session *Session) DeploymentName() string {
	return session.deploymentName
}

func (session *Session) CurrentDeployment() Deployment {
	return session.deployment
}

func (session *Session) SetCurrentDeployment(deployment Deployment) {
	session.deployment = deployment
}

func (session *Session) CurrentArtifact() Backup {
	return session.currentArtifact
}

type Step interface {
	Run(*Session) error
}

type CleanupStep struct{}

func NewCleanupStep() Step {
	return &CleanupStep{}
}

func (s *CleanupStep) Run(session *Session) error {

	if err := session.CurrentDeployment().Cleanup(); err != nil {
		return NewCleanupError(
			fmt.Sprintf("Deployment '%s' failed while cleaning up with error: %v", session.DeploymentName(), err))
	}
	return nil
}

type DrainStep struct {
	logger Logger
}

func NewDrainStep(logger Logger) Step {
	return &DrainStep{
		logger: logger,
	}
}

func (s *DrainStep) Run(session *Session) error {
	defer s.logger.Info("bbr", "Backup created of %s on %v\n", session.DeploymentName(), time.Now())
	return session.CurrentDeployment().CopyRemoteBackupToLocal(session.CurrentArtifact())

}

type UnlockStep struct {
	lockOrderer LockOrderer
}

func NewUnlockStep(lockOrderer LockOrderer) Step {
	return &UnlockStep{
		lockOrderer: lockOrderer,
	}
}

func (s *UnlockStep) Run(session *Session) error {
	err := session.CurrentDeployment().PostBackupUnlock(s.lockOrderer)
	if err != nil {
		return NewPostBackupUnlockError(err.Error())
	}
	return nil
}

type BackupStep struct{}

func (s *BackupStep) Run(session *Session) error {
	err := session.CurrentDeployment().Backup()
	if err != nil {
		return NewBackupError(err.Error())
	}
	return nil
}

func NewBackupStep() Step {
	return &BackupStep{}
}

type LockStep struct {
	lockOrderer LockOrderer
}

func (s *LockStep) Run(session *Session) error {
	err := session.CurrentDeployment().PreBackupLock(s.lockOrderer)
	if err != nil {
		return NewLockError(err.Error())
	}
	return nil
}

func NewLockStep(lockOrderer LockOrderer) Step {
	return &LockStep{lockOrderer: lockOrderer}
}

type CreateArtifactStep struct {
	logger            Logger
	backupManager     BackupManager
	deploymentManager DeploymentManager
	nowFunc           func() time.Time
}

func (s *CreateArtifactStep) Run(session *Session) error {
	s.logger.Info("bbr", "Starting backup of %s...\n", session.DeploymentName())
	artifact, err := s.backupManager.Create(session.DeploymentName(), s.logger, time.Now)
	if err != nil {
		return err
	}
	artifact.CreateMetadataFileWithStartTime(s.nowFunc())
	session.SetCurrentArtifact(artifact)

	err = s.deploymentManager.SaveManifest(session.DeploymentName(), artifact)
	if err != nil {
		return err
	}
	return nil
}

func NewCreateArtifactStep(logger Logger, backupManager BackupManager, deploymentManager DeploymentManager, nowFunc func() time.Time) Step {
	return &CreateArtifactStep{logger: logger, backupManager: backupManager, deploymentManager: deploymentManager, nowFunc: nowFunc}
}

type BackupableStep struct {
	lockOrderer LockOrderer
}

func NewBackupableStep(lockOrderer LockOrderer) Step {
	return &BackupableStep{lockOrderer: lockOrderer}
}

func (s *BackupableStep) Run(session *Session) error {
	deployment := session.CurrentDeployment()
	if !deployment.IsBackupable() {
		return errors.Errorf("Deployment '%s' has no backup scripts", session.DeploymentName())
	}

	err := deployment.CheckArtifactDir()
	if err != nil {
		return err
	}

	if !deployment.HasUniqueCustomArtifactNames() {
		return errors.Errorf("Multiple jobs in deployment '%s' specified the same backup name", session.DeploymentName())
	}

	if err := deployment.CustomArtifactNamesMatch(); err != nil {
		return err
	}

	if err := deployment.ValidateLockingDependencies(s.lockOrderer); err != nil {
		return err
	}
	return nil
}

type CheckDeploymentStep struct {
	deploymentManager DeploymentManager
	logger            Logger
}

func NewCheckDeploymentStep(deploymentManager DeploymentManager, logger Logger) Step {
	return &CheckDeploymentStep{deploymentManager: deploymentManager, logger: logger}
}

func (s *CheckDeploymentStep) Run(session *Session) error {
	s.logger.Info("bbr", "Running pre-checks for backup of %s...\n", session.DeploymentName())

	s.logger.Info("bbr", "Scripts found:")
	deployment, err := s.deploymentManager.Find(session.DeploymentName())
	if err != nil {
		return err
	}

	session.SetCurrentDeployment(deployment)

	return nil
}

func (b Backuper) CanBeBackedUp(deploymentName string) (bool, Error) {
	bw := newBackupCheckWorkflow(b, deploymentName)

	err := bw.Run()
	return err == nil, err
}
